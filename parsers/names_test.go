package parsers

import "testing"

func TestNormalizeNames(t *testing.T) {
	got := NormalizeNames([]string{" nginx ", "Nginx", "", "redis"})
	want := []string{"nginx", "redis"}
	if len(got) != len(want) {
		t.Fatalf("NormalizeNames() = %#v, want %#v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("NormalizeNames() = %#v, want %#v", got, want)
		}
	}
}
