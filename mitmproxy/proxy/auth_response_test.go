package proxy

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthenticationDenialIsVisibleAndDoesNotExposeCallbackError(t *testing.T) {
	const privateDetail = "private-proxy-credential"
	for _, method := range []string{http.MethodGet, http.MethodConnect} {
		t.Run(method, func(t *testing.T) {
			for _, tc := range []struct {
				name string
				err  error
			}{
				{"nil-error", nil},
				{"callback-error", errors.New(privateDetail)},
			} {
				t.Run(tc.name, func(t *testing.T) {
					proxy := &Proxy{authProxy: func(http.ResponseWriter, *http.Request) (bool, error) {
						return false, tc.err
					}}
					target := "http://example.invalid/"
					if method == http.MethodConnect {
						target = "example.invalid:443"
					}
					res := httptest.NewRecorder()
					// No interceptor or connection context is installed: rejected
					// requests must return before forwarding or CONNECT handling.
					(&entry{proxy: proxy}).ServeHTTP(res, httptest.NewRequest(method, target, nil))
					if res.Code != http.StatusProxyAuthRequired {
						t.Fatalf("status = %d, want 407", res.Code)
					}
					if got := res.Header().Get("Proxy-Authenticate"); got != `Basic realm="proxy"` {
						t.Fatalf("authentication challenge = %q", got)
					}
					if got := res.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
						t.Fatalf("content type = %q", got)
					}
					body := res.Body.String()
					if !strings.Contains(body, "request was not forwarded") {
						t.Fatalf("missing actionable denial: %q", body)
					}
					if strings.Contains(body, privateDetail) {
						t.Fatalf("callback details leaked in denial: %q", body)
					}
				})
			}
		})
	}
}
