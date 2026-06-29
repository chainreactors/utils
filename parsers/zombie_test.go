package parsers

import "testing"

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
