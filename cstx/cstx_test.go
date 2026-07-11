//go:build cstx_native

package cstx

import (
	"testing"

	"github.com/chainreactors/utils/parsers"
)

func TestParseGogoResults(t *testing.T) {
	results := []*parsers.GOGOResult{
		{
			Ip: "192.168.1.1", Port: "80", Protocol: "tcp",
			Status: "200", Uri: "http://192.168.1.1/",
			Title: "Test Page", Midware: "nginx",
			Frameworks: parsers.Frameworks{
				"jquery": &parsers.Framework{Name: "jquery"},
				"nginx":  &parsers.Framework{Name: "nginx"},
			},
			Vulns: parsers.Vulns{
				"CVE-2021-1234": &parsers.Vuln{Name: "CVE-2021-1234", SeverityLevel: 3},
			},
		},
	}

	nodes, err := Parse("gogo", results)
	if err != nil {
		t.Fatalf("Parse gogo: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected nodes, got empty")
	}

	types := make(map[string]int)
	for _, n := range nodes {
		types[n.CstxType()]++
		t.Logf("  %s %s", n.CstxType(), n.CstxID())
	}

	for _, expect := range []string{"ip", "port", "app", "framework", "vuln"} {
		if types[expect] == 0 {
			t.Errorf("missing node type: %s", expect)
		}
	}
	t.Logf("total nodes: %d, types: %v", len(nodes), types)
}

func TestParseSprayResults(t *testing.T) {
	results := []*parsers.SprayResult{
		{
			UrlString: "http://example.com/login", Title: "Login",
			Status: 200, Host: "example.com", Path: "/login",
			BodyLength: 1234, ContentType: "text/html",
			Frameworks: parsers.Frameworks{
				"react": &parsers.Framework{Name: "react"},
			},
		},
	}

	nodes, err := Parse("spray", results)
	if err != nil {
		t.Fatalf("Parse spray: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected nodes, got empty")
	}

	types := make(map[string]int)
	for _, n := range nodes {
		types[n.CstxType()]++
		t.Logf("  %s %s", n.CstxType(), n.CstxID())
	}

	if types["framework"] == 0 {
		t.Error("missing framework node from spray with map-format Frameworks")
	}
}

func TestParseRawBytes(t *testing.T) {
	jsonl := []byte(`{"ip":"10.0.0.1","port":"443","protocol":"tcp","status":"200"}` + "\n")

	nodes, err := Parse("gogo", jsonl)
	if err != nil {
		t.Fatalf("Parse raw: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected nodes from raw JSONL")
	}

	for _, n := range nodes {
		t.Logf("  %s %s", n.CstxType(), n.CstxID())
	}
}

func TestParseZombieResults(t *testing.T) {
	results := []*parsers.ZombieResult{
		{
			IP: "10.0.0.1", Port: "22", Service: "ssh",
			Username: "root", Password: "toor",
		},
	}

	nodes, err := Parse("zombie", results)
	if err != nil {
		t.Fatalf("Parse zombie: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected nodes, got empty")
	}

	hasVuln := false
	for _, n := range nodes {
		if vuln, ok := n.(*Vuln); ok {
			hasVuln = true
			t.Logf("  vuln: %s user=%s pass=%s severity=%s", vuln.Name, vuln.Username, vuln.Password, vuln.Severity)
		} else {
			t.Logf("  %s %s", n.CstxType(), n.CstxID())
		}
	}
	if !hasVuln {
		t.Error("expected vuln node from zombie result")
	}
}

func TestParseSingleResult(t *testing.T) {
	cases := []struct {
		tool  string
		input any
	}{
		{"gogo", &parsers.GOGOResult{Ip: "172.16.0.1", Port: "8443", Protocol: "https", Title: "Admin"}},
		{"spray", &parsers.SprayResult{UrlString: "https://api.example.com/health", Status: 200, Host: "api.example.com"}},
		{"zombie", &parsers.ZombieResult{IP: "10.10.10.1", Port: "3306", Service: "mysql", Username: "admin", Password: "admin123"}},
	}
	for _, tc := range cases {
		nodes, err := Parse(tc.tool, tc.input)
		if err != nil {
			t.Errorf("Parse single %s: %v", tc.tool, err)
			continue
		}
		if len(nodes) == 0 {
			t.Errorf("single %s: expected nodes, got none", tc.tool)
			continue
		}
		t.Logf("single %s -> %d nodes", tc.tool, len(nodes))
	}
}
