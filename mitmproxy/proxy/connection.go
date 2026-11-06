package proxy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net"
	"net/http"
	"sync"

	uuid "github.com/satori/go.uuid"
	"go.uber.org/atomic"
)

// client connection
type ClientConn struct {
	Id                 uuid.UUID
	Conn               net.Conn
	Tls                bool
	NegotiatedProtocol string
	UpstreamCert       bool // Connect to upstream server to look up certificate details. Default: True
	clientHello        *tls.ClientHelloInfo
}

func newClientConn(c net.Conn) *ClientConn {
	return &ClientConn{
		Id:           uuid.NewV4(),
		Conn:         c,
		Tls:          false,
		UpstreamCert: true,
	}
}

func (c *ClientConn) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["id"] = c.Id
	m["tls"] = c.Tls
	m["address"] = c.Conn.RemoteAddr().String()
	return json.Marshal(m)
}

// server connection
type ServerConn struct {
	Id      uuid.UUID
	Address string
	Conn    net.Conn

	client   *http.Client
	tlsConn  *tls.Conn
	tlsState *tls.ConnectionState
}

func newServerConn() *ServerConn {
	return &ServerConn{
		Id: uuid.NewV4(),
	}
}

func (c *ServerConn) MarshalJSON() ([]byte, error) {
	m := make(map[string]interface{})
	m["id"] = c.Id
	m["address"] = c.Address
	peername := ""
	if c.Conn != nil {
		peername = c.Conn.RemoteAddr().String()
	}
	m["peername"] = peername
	return json.Marshal(m)
}

func (c *ServerConn) TlsState() *tls.ConnectionState {
	return c.tlsState
}

// connection context ctx key
var connContextKey = new(struct{})

type proxyAuthUserContextKey struct{}

// ProxyAuthUser returns the request's local proxy username. Requests inside a
// CONNECT tunnel inherit the username captured when the tunnel was opened.
func ProxyAuthUser(ctx context.Context) string {
	if user, ok := ctx.Value(proxyAuthUserContextKey{}).(string); ok {
		return user
	}
	if conn, ok := ctx.Value(connContextKey).(*ConnContext); ok && conn != nil {
		return conn.ProxyAuthUser
	}
	return ""
}

// connection context
type ConnContext struct {
	ClientConn *ClientConn   `json:"clientConn"`
	ServerConn *ServerConn   `json:"serverConn"`
	Intercept  bool          `json:"intercept"` // Indicates whether to parse HTTPS
	FlowCount  atomic.Uint32 `json:"-"`         // Number of HTTP requests made on the same connection
	// ProxyAuthUser is the proxy username from the first request on this
	// connection. Use Flow.ProxyAuthUser for an individual request, since HTTP
	// keep-alive requests may carry different credentials. Never used for
	// authorization here.
	ProxyAuthUser string `json:"-"`

	authOnce           sync.Once
	proxy              *Proxy
	closeAfterResponse bool                        // after http response, http server will close the connection
	dialFn             func(context.Context) error // when begin request, if there no ServerConn, use this func to dial
}

func newConnContext(c net.Conn, proxy *Proxy) *ConnContext {
	clientConn := newClientConn(c)
	return &ConnContext{
		ClientConn: clientConn,
		proxy:      proxy,
	}
}

func (connCtx *ConnContext) Id() uuid.UUID {
	return connCtx.ClientConn.Id
}
