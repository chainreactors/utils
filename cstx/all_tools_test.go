//go:build cstx_native

package cstx

import (
	"testing"

	"github.com/chainreactors/utils/parsers"
)

func TestAllToolTypes(t *testing.T) {
	type testCase struct {
		tool   string
		data   any
		expect []string
	}

	cases := []testCase{
		{
			tool: "gogo",
			data: &parsers.GOGOResult{
				Ip: "10.0.0.1", Port: "443", Protocol: "https",
				Status: "200", Title: "Dashboard",
				Frameworks: parsers.Frameworks{"vue": &parsers.Framework{Name: "vue"}},
				Vulns:      parsers.Vulns{"CVE-2024-9999": &parsers.Vuln{Name: "CVE-2024-9999", SeverityLevel: 3}},
			},
			expect: []string{"ip", "port", "app", "framework", "vuln"},
		},
		{
			tool: "spray",
			data: &parsers.SprayResult{
				UrlString: "https://app.example.com/api", Status: 200,
				Host: "app.example.com", Path: "/api",
				Frameworks: parsers.Frameworks{"express": &parsers.Framework{Name: "express"}},
			},
			expect: []string{"url", "app", "framework"},
		},
		{
			tool: "zombie",
			data: &parsers.ZombieResult{
				IP: "10.0.0.1", Port: "22", Service: "ssh",
				Username: "root", Password: "123456",
			},
			expect: []string{"ip", "port", "vuln"},
		},
		{
			tool:   "neutron",
			data:   []byte(`{"template":"shiro-detect","name":"Shiro Default Key","severity":"high","target":"http://10.0.0.1:8080"}` + "\n"),
			expect: []string{"app", "vuln"},
		},
		{
			tool:   "proton",
			data:   []byte(`{"template-id":"aws-access-key","template-name":"AWS Access Key","severity":"critical","file":"/app/.env","extracts":[{"value":"AKIA1234","line":3}]}` + "\n"),
			expect: []string{"vuln"},
		},
		{
			tool:   "katana",
			data:   []byte(`{"request":{"endpoint":"https://target.com/api","method":"GET"},"response":{"status_code":200,"technologies":["React"]}}` + "\n"),
			expect: []string{"url", "endpoint", "framework"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.tool, func(t *testing.T) {
			nodes, err := Parse(tc.tool, tc.data)
			if err != nil {
				t.Fatalf("Parse %s: %v", tc.tool, err)
			}
			if len(nodes) == 0 {
				t.Fatalf("%s: got 0 nodes", tc.tool)
			}

			types := make(map[string]bool)
			for _, n := range nodes {
				types[n.CstxType()] = true
				t.Logf("  %s %s", n.CstxType(), n.CstxID())
			}

			for _, req := range tc.expect {
				if !types[req] {
					t.Errorf("%s: missing required node type %q", tc.tool, req)
				}
			}
			t.Logf("%s: %d nodes OK", tc.tool, len(nodes))
		})
	}
}
