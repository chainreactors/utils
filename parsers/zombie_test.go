package parsers

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestZombieResultWeakpassFinding(t *testing.T) {
	result := &ZombieResult{
		IP:       "127.0.0.1",
		Port:     "22",
		Service:  "ssh",
		Username: "root",
		Password: "toor",
		Mod:      ZombieModBrute,
	}

	want := "[weakpass] ssh://127.0.0.1:22 user=root pass=toor mod=brute"
	if got := result.WeakpassFinding(); got != want {
		t.Fatalf("WeakpassFinding() = %q, want %q", got, want)
	}
	if got := result.Format(ZombieFormatWeakpassFinding); got != want {
		t.Fatalf("Format(%q) = %q, want %q", ZombieFormatWeakpassFinding, got, want)
	}
}

func TestZombieServiceHelpers(t *testing.T) {
	if got, ok := ZombieServiceFromName("mariadb"); !ok || got != "mysql" {
		t.Fatalf("ZombieServiceFromName(mariadb) = %q, %v", got, ok)
	}
	if got, ok := ZombieServiceFromName("mongodb"); !ok || got != "mongo" {
		t.Fatalf("ZombieServiceFromName(mongodb) = %q, %v", got, ok)
	}
}

func TestZombieResultCarriesExecutionOutput(t *testing.T) {
	result := &ZombieResult{
		IP:        "127.0.0.1",
		Port:      "22",
		Service:   "ssh",
		Username:  "root",
		Password:  "bad",
		Mod:       ZombieModBrute,
		ErrString: "authentication failed",
		Vulns: Vulns{
			"weak-password": {Name: "weak-password"},
		},
		Extracteds: Extracteds{&Extracted{
			Name:          "banner",
			ExtractResult: []string{"OpenSSH"},
		}},
		Loot: map[string][]byte{"banner": []byte("OpenSSH")},
	}

	if result.Success() {
		t.Fatal("failed result reported success")
	}
	if got := result.Full(); !strings.Contains(got, "login failed") || !strings.Contains(got, result.ErrString) {
		t.Fatalf("Full() = %q, want failure details", got)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(raw)
	for _, field := range []string{`"ok":false`, `"error":"authentication failed"`, `"vulns"`, `"extracteds"`, `"loot"`} {
		if !strings.Contains(encoded, field) {
			t.Fatalf("JSON %s does not contain %s", encoded, field)
		}
	}

	result.OK = true
	result.ErrString = ""
	if !result.Success() {
		t.Fatal("successful result reported failure")
	}
	if got := result.Full(); !strings.Contains(got, "login successfully") {
		t.Fatalf("Full() = %q, want success details", got)
	}
}
