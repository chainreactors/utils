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
		switch v := n.(type) {
		case *Ip:
			t.Logf("  ip: %s (cidr=%s)", v.Ip, v.Cidr)
		case *Port:
			t.Logf("  port: %s:%s/%s", v.Ip, v.Port, v.Protocol)
		case *App:
			t.Logf("  app: %s title=%q frameworks=%v", v.AppId, v.Title, v.Frameworks)
		case *Framework:
			t.Logf("  framework: %s", v.Name)
		case *Vuln:
			t.Logf("  vuln: %s severity=%s", v.Name, v.Severity)
		default:
			t.Logf("  %s %s", n.CstxType(), n.CstxID())
		}
	}

	for _, expect := range []string{"ip", "port", "app"} {
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

	for _, n := range nodes {
		if app, ok := n.(*App); ok {
			t.Logf("  app: %s host=%s content_type=%s body_length=%d", app.AppId, app.Host, app.ContentType, app.BodyLength)
		} else {
			t.Logf("  %s %s", n.CstxType(), n.CstxID())
		}
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

	for _, n := range nodes {
		if vuln, ok := n.(*Vuln); ok {
			t.Logf("  vuln: %s user=%s pass=%s severity=%s", vuln.Name, vuln.Username, vuln.Password, vuln.Severity)
		} else {
			t.Logf("  %s %s", n.CstxType(), n.CstxID())
		}
	}
}
