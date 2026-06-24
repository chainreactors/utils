package cert

import (
	"strings"
	"testing"
)

func TestRandomSubject(t *testing.T) {
	s := RandomSubject("test.example.com")
	if s.CommonName != "test.example.com" {
		t.Fatalf("expected CN=test.example.com, got %s", s.CommonName)
	}
	if len(s.Organization) == 0 || s.Organization[0] == "" {
		t.Fatal("expected non-empty Organization")
	}
	if len(s.Country) == 0 || s.Country[0] == "" {
		t.Fatal("expected non-empty Country")
	}
}

func TestRandomSubjectEmptyCN(t *testing.T) {
	s := RandomSubject("")
	if s.CommonName == "" {
		t.Fatal("expected auto-generated CN")
	}
	if !strings.Contains(s.CommonName, ".") {
		t.Fatalf("expected domain-style CN, got %q", s.CommonName)
	}
}

func TestRandomSubjectUniqueness(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		s := RandomSubject("test")
		key := s.Organization[0] + "|" + s.Country[0] + "|" + s.Province[0]
		seen[key] = true
	}
	if len(seen) < 5 {
		t.Fatalf("expected variety, got only %d unique combos", len(seen))
	}
}

func TestRandomSubjectWith_CustomWordLists(t *testing.T) {
	s := RandomSubjectWith("test",
		WithWordLists([]string{"custom"}, []string{"words"}),
	)
	org := s.Organization[0]
	if !strings.Contains(strings.ToLower(org), "custom") && !strings.Contains(strings.ToLower(org), "words") {
		t.Fatalf("expected custom words in org, got %q", org)
	}
}

func TestRandomSubjectWith_CustomGeoData(t *testing.T) {
	geo := []GeoEntry{{"XX", "TestProvince", "TestCity", []string{"123 Test St"}}}
	s := RandomSubjectWith("test", WithGeoData(geo))
	if s.Country[0] != "XX" {
		t.Fatalf("expected country XX, got %s", s.Country[0])
	}
}
