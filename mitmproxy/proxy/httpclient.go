package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// NewProxyHTTPClient creates an *http.Client routed through the proxy at proxyAddr.
func NewProxyHTTPClient(proxyAddr string, timeout time.Duration) (*http.Client, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	proxyURL, err := url.Parse(proxyAddr)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:             http.ProxyURL(proxyURL),
			ForceAttemptHTTP2: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			TLSHandshakeTimeout: timeout,
			MaxConnsPerHost:     50,
			IdleConnTimeout:     timeout,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}, nil
}

// TagRoundTripper injects a custom header on every outbound request.
type TagRoundTripper struct {
	Inner  http.RoundTripper
	Header string
	Value  string
}

func (t *TagRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r2 := req.Clone(req.Context())
	if r2.Header == nil {
		r2.Header = make(http.Header)
	}
	r2.Header.Set(t.Header, t.Value)
	return t.Inner.RoundTrip(r2)
}

// NewTaggedProxyClient creates a proxy HTTP client that injects a tag header
// on every request, enabling per-session flow attribution.
func NewTaggedProxyClient(proxyAddr, tagHeader, tagValue string, timeout time.Duration) (*http.Client, error) {
	c, err := NewProxyHTTPClient(proxyAddr, timeout)
	if err != nil {
		return nil, err
	}
	if tagHeader != "" && tagValue != "" {
		c.Transport = &TagRoundTripper{
			Inner:  c.Transport,
			Header: tagHeader,
			Value:  tagValue,
		}
	}
	return c, nil
}
