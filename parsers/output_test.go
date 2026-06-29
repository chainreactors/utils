package parsers

import (
	"strings"
	"testing"
)

func TestOutputLineUnifiesGogoAndSprayFrameworks(t *testing.T) {
	frameworks := Frameworks{
		"nginx":   NewFramework("nginx", FrameFromFingers),
		"struts2": NewFramework("struts2", FrameFromFingers),
	}

	gogo := NewGOGOResult("127.0.0.1", "8080")
	gogo.Protocol = "http"
	gogo.Status = "200"
	gogo.Frameworks = frameworks

	spray := &SprayResult{
		UrlString:  "http://127.0.0.1:8080",
		Source:     CheckSource,
		Status:     200,
		BodyLength: 12,
		Frameworks: frameworks,
	}

	for _, got := range []string{gogo.OutputLine(), spray.OutputLine()} {
		if !strings.Contains(got, "[nginx,struts2]") {
			t.Fatalf("OutputLine() = %q, want unified framework token", got)
		}
	}
}

func TestSprayOutputLineOmitsSourceName(t *testing.T) {
	result := &SprayResult{
		UrlString:  "https://example.com/#/login",
		Source:     CheckSource,
		Status:     200,
		BodyLength: 3209,
		Spended:    122,
	}

	got := result.OutputLine()
	if strings.Contains(got, " check ") {
		t.Fatalf("SprayResult.OutputLine() = %q, should not include source name", got)
	}
	if !strings.Contains(got, "https://example.com/#/login 200 3209 122ms") {
		t.Fatalf("SprayResult.OutputLine() = %q, want compact spray fields", got)
	}
}

func TestLegacyStringOutputIsUnchanged(t *testing.T) {
	frameworks := Frameworks{"nginx": NewFramework("nginx", FrameFromFingers)}

	gogo := NewGOGOResult("127.0.0.1", "8080")
	gogo.Protocol = "http"
	gogo.Status = "200"
	gogo.Frameworks = frameworks
	if got := gogo.String(); !strings.Contains(got, "status=200") || !strings.Contains(got, "frameworks=nginx") {
		t.Fatalf("GOGOResult.String() = %q, want legacy key/value style", got)
	}

	spray := &SprayResult{
		UrlString:  "http://127.0.0.1:8080",
		Source:     CheckSource,
		Status:     200,
		BodyLength: 12,
		Frameworks: frameworks,
	}
	if got := spray.String(); !strings.Contains(got, "[check]") || !strings.Contains(got, " [nginx]") {
		t.Fatalf("SprayResult.String() = %q, want legacy spray style", got)
	}
}

func TestZombieOutputLine(t *testing.T) {
	result := &ZombieResult{
		IP:       "127.0.0.1",
		Port:     "22",
		Service:  "ssh",
		Username: "root",
		Password: "toor",
		Mod:      ZombieModBrute,
	}

	got := result.OutputLine()
	for _, want := range []string{"ssh://127.0.0.1:22", "root", "toor", "ssh", "brute"} {
		if !strings.Contains(got, want) {
			t.Fatalf("ZombieResult.OutputLine() = %q, missing %q", got, want)
		}
	}
}
