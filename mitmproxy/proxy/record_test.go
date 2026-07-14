package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func mustParseURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}

func TestNewFlowRecord(t *testing.T) {
	f := newFlow()
	f.Request = &Request{
		Method: "POST",
		URL:    mustParseURL("https://example.com/api"),
		Header: http.Header{"Content-Type": {"application/json"}},
		Body:   []byte(`{"key":"value"}`),
	}
	f.Response = &Response{
		StatusCode: 201,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       make([]byte, 8000),
	}
	f.ConnContext = &ConnContext{
		ClientConn: &ClientConn{Tls: true},
	}

	r := NewFlowRecord(f, 4096)

	if r.Method != "POST" {
		t.Fatalf("method: expected POST, got %s", r.Method)
	}
	if r.Host != "example.com" {
		t.Fatalf("host: expected example.com, got %s", r.Host)
	}
	if r.StatusCode != 201 {
		t.Fatalf("status: expected 201, got %d", r.StatusCode)
	}
	if !r.TLS {
		t.Fatal("expected TLS=true")
	}
	if len(r.ResponseBody) != 4096 {
		t.Fatalf("body snip: expected 4096, got %d", len(r.ResponseBody))
	}
}

func TestSnipBytes(t *testing.T) {
	if s := snipBytes(nil, 100); s != nil {
		t.Fatal("nil input should return nil")
	}
	if s := snipBytes([]byte("hello"), 0); len(s) != 5 {
		t.Fatalf("max=0 should keep all, got %d", len(s))
	}
	if s := snipBytes([]byte("hello world"), 5); len(s) != 5 {
		t.Fatalf("should truncate to 5, got %d", len(s))
	}
}

func TestRecordAddon_CapturesFlows(t *testing.T) {
	var mu sync.Mutex
	var captured []*FlowRecord

	addon := NewRecordAddon(RecordAddonConfig{
		MaxBodySnip: 1024,
		OnRecord: func(r *FlowRecord) {
			mu.Lock()
			captured = append(captured, r)
			mu.Unlock()
		},
	})

	p, err := NewProxy(&Options{
		Addr:        "127.0.0.1:0",
		SslInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.AddAddon(addon)

	addr, _, err := p.StartAsync()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Shutdown(context.Background())

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(200)
		fmt.Fprint(w, "hello from target")
	}))
	defer target.Close()

	client, err := NewProxyHTTPClient("http://"+addr.String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(target.URL + "/test")
	if err != nil {
		t.Fatal(err)
	}
	io.ReadAll(resp.Body)
	resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	if len(captured) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(captured))
	}
	r := captured[0]
	if r.Method != "GET" {
		t.Fatalf("method: expected GET, got %s", r.Method)
	}
	if r.StatusCode != 200 {
		t.Fatalf("status: expected 200, got %d", r.StatusCode)
	}
	if r.Duration <= 0 {
		t.Fatal("expected positive duration")
	}
	if string(r.ResponseBody) != "hello from target" {
		t.Fatalf("body: expected 'hello from target', got %q", string(r.ResponseBody))
	}
}

func TestRecordAddon_TagExtraction(t *testing.T) {
	var mu sync.Mutex
	var captured []*FlowRecord

	addon := NewRecordAddon(RecordAddonConfig{
		TagHeader: "X-Test-Tag",
		TagKey:    "test",
		StripTag:  true,
		OnRecord: func(r *FlowRecord) {
			mu.Lock()
			captured = append(captured, r)
			mu.Unlock()
		},
	})

	p, err := NewProxy(&Options{
		Addr:        "127.0.0.1:0",
		SslInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.AddAddon(addon)

	addr, _, err := p.StartAsync()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Shutdown(context.Background())

	var receivedTag string
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedTag = r.Header.Get("X-Test-Tag")
		w.WriteHeader(200)
	}))
	defer target.Close()

	client, err := NewTaggedProxyClient("http://"+addr.String(), "X-Test-Tag", "my-scan-123", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(target.URL + "/check")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	if receivedTag != "" {
		t.Fatalf("tag should have been stripped, but target received %q", receivedTag)
	}

	mu.Lock()
	defer mu.Unlock()

	if len(captured) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(captured))
	}
	if captured[0].Tags["test"] != "my-scan-123" {
		t.Fatalf("tag: expected 'my-scan-123', got %q", captured[0].Tags["test"])
	}
}

func TestRecordAddon_Enrich(t *testing.T) {
	enriched := false
	var mu sync.Mutex
	var captured []*FlowRecord

	addon := NewRecordAddon(RecordAddonConfig{
		MaxBodySnip: 1024,
		Enrich: func(r *FlowRecord, raw *Flow) {
			r.Tags["custom"] = "enriched"
			enriched = true
		},
		OnRecord: func(r *FlowRecord) {
			mu.Lock()
			captured = append(captured, r)
			mu.Unlock()
		},
	})

	p, err := NewProxy(&Options{
		Addr:        "127.0.0.1:0",
		SslInsecure: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.AddAddon(addon)

	addr, _, err := p.StartAsync()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Shutdown(context.Background())

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer target.Close()

	client, err := NewProxyHTTPClient("http://"+addr.String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	time.Sleep(100 * time.Millisecond)

	if !enriched {
		t.Fatal("enrich callback was not called")
	}

	mu.Lock()
	defer mu.Unlock()

	if len(captured) != 1 {
		t.Fatalf("expected 1 record, got %d", len(captured))
	}
	if captured[0].Tags["custom"] != "enriched" {
		t.Fatalf("expected enriched tag, got %v", captured[0].Tags)
	}
}
