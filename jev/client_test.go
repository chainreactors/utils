package jev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientWireFormatAndRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer k" {
			t.Errorf("auth header: %q", r.Header.Get("Authorization"))
		}
		if calls == 1 {
			w.WriteHeader(529)
			return
		}
		var req struct {
			Model     string
			Questions map[string]map[string]interface{}
		}
		json.NewDecoder(r.Body).Decode(&req)
		c := req.Questions["c"]
		criteria, _ := c["criteria"].(map[string]interface{})
		if req.Model != DefaultModel || c["type"] != "choice" || criteria["x"] != nil || criteria["y"] != "why" || criteria["z"] != "another description" {
			t.Errorf("request: %+v", req)
		}
		w.Write([]byte(`{"model":"jev-1.13.0","answers":{"c":{"type":"choice","choice":"y","confidence":0.8}},"usage":{"input_tokens":10,"output_tokens":1}}`))
	}))
	defer srv.Close()

	p, _ := NewClient("k")
	p.Endpoint = srv.URL
	answers, err := p.Judge(context.Background(), "state", map[string]Claim{
		"c": {Statement: "?", Options: map[string]Option{"x": {Outcome: Refuted}, "y": {Description: "why", Outcome: Holds}, "z": {Description: "another description", Outcome: Holds}, OptionInsufficient: {Outcome: Insufficient}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 || answers["c"].Option != "y" || p.InputTokens != 10 {
		t.Fatalf("calls=%d answers=%+v tokens=%d", calls, answers, p.InputTokens)
	}
	if p.ID() != "jev/"+srv.URL+"/jev-1.13.0" {
		t.Fatalf("id %s", p.ID())
	}
}

func TestClientRequiresExplicitWireConfidence(t *testing.T) {
	for _, tc := range []struct {
		name, answer string
		valid        bool
	}{
		{"missing", `{"choice":"yes"}`, false},
		{"null", `{"choice":"yes","confidence":null}`, false},
		{"zero", `{"choice":"yes","confidence":0}`, true},
		{"unknown option", `{"choice":"other","confidence":0.9}`, false},
		{"core format", `{"option":"yes","confidence":0.9}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Write([]byte(`{"answers":{"c":` + tc.answer + `}}`))
			}))
			defer srv.Close()
			p, _ := NewClient("local-test")
			p.Endpoint = srv.URL
			claims := map[string]Claim{"c": {Statement: "Is it running?", Options: map[string]Option{
				"yes": {Outcome: Holds}, OptionInsufficient: {Outcome: Insufficient},
			}}}
			rulings, err := p.Judge(context.Background(), "state", claims)
			if (err == nil) != tc.valid || calls != 1 {
				t.Fatalf("rulings=%v error=%v calls=%d", rulings, err, calls)
			}
			if tc.valid && (rulings["c"].Option != "yes" || rulings["c"].Confidence != 0) {
				t.Fatalf("zero confidence changed: %v", rulings)
			}
			invalid := claims["c"]
			invalid.Statement = ""
			claims["c"] = invalid
			if _, err := p.Judge(context.Background(), "state", claims); err == nil || calls != 1 {
				t.Fatal("invalid claim reached HTTP boundary")
			}
		})
	}
}
