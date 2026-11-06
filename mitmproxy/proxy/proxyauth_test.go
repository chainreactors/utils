package proxy

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseProxyAuthUser(t *testing.T) {
	basic := func(s string) string { return "Basic " + base64.StdEncoding.EncodeToString([]byte(s)) }
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"user and password", basic("call-123:secret"), "call-123"},
		{"user only, no colon", basic("call-123"), "call-123"},
		{"empty user with colon", basic(":secret"), ""},
		{"lowercase scheme", "basic " + base64.StdEncoding.EncodeToString([]byte("abc:")), "abc"},
		{"empty header", "", ""},
		{"non-basic scheme", "Bearer token", ""},
		{"malformed base64", "Basic !!!not-base64!!!", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseProxyAuthUser(c.header); got != c.want {
				t.Fatalf("parseProxyAuthUser(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

type identityAddon struct {
	BaseAddon
	flows []*Flow
}

func (a *identityAddon) Requestheaders(f *Flow) { f.Response = &Response{StatusCode: 200} }
func (a *identityAddon) FlowFinished(f *Flow)   { a.flows = append(a.flows, f) }

func TestKeepAliveRequestsKeepTheirOwnProxyIdentity(t *testing.T) {
	addon := &identityAddon{}
	p := &Proxy{Addons: []Addon{addon}}
	p.interceptor = &interceptor{proxy: p}
	conn := &ConnContext{proxy: p, ClientConn: &ClientConn{}}
	users := []string{"call-A", "call-B", ""}
	for _, user := range users {
		req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
		if user != "" {
			req.Header.Set("Proxy-Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(user+":")))
		}
		req = req.WithContext(context.WithValue(req.Context(), connContextKey, conn))
		(&entry{proxy: p}).ServeHTTP(httptest.NewRecorder(), req)
	}
	if len(addon.flows) != len(users) {
		t.Fatalf("got %d flows", len(addon.flows))
	}
	for i, f := range addon.flows {
		if f.ProxyAuthUser != users[i] {
			t.Fatalf("request %d identity changed: got %q, want %q", i, f.ProxyAuthUser, users[i])
		}
		if f.Request.Header.Get("Proxy-Authorization") != "" {
			t.Fatal("proxy credentials leaked into forwarded request")
		}
	}
	if conn.ProxyAuthUser != "call-A" {
		t.Fatalf("connection identity changed: %q", conn.ProxyAuthUser)
	}
}

func TestProxyAuthUserContextOverridesTunnelIdentity(t *testing.T) {
	ctx := context.WithValue(t.Context(), connContextKey, &ConnContext{ProxyAuthUser: "tunnel"})
	if got := ProxyAuthUser(ctx); got != "tunnel" {
		t.Fatalf("tunnel identity = %q", got)
	}
	for _, user := range []string{"request", ""} {
		requestCtx := context.WithValue(ctx, proxyAuthUserContextKey{}, user)
		if got := ProxyAuthUser(requestCtx); got != user {
			t.Fatalf("request identity = %q, want %q", got, user)
		}
	}
}
