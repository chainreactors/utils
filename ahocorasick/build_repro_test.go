package ahocorasick

import (
	"os"
	"runtime"
	"strconv"
	"sync"
	"testing"
)

func TestByteClassesLargeAlphabet(t *testing.T) {
	patterns255 := make([][]byte, 0, 255)
	for i := 0; i < 255; i++ {
		patterns255 = append(patterns255, []byte{byte(i)})
	}
	ac, err := NewBuilder().AddPatterns(patterns255).Build()
	if err != nil {
		t.Fatal(err)
	}
	if !ac.IsMatch([]byte{254}) {
		t.Fatal("expected match for byte 254")
	}

	patterns256 := make([][]byte, 0, 256)
	for i := 0; i < 256; i++ {
		patterns256 = append(patterns256, []byte{byte(i)})
	}
	ac, err = NewBuilder().AddPatterns(patterns256).Build()
	if err != nil {
		t.Fatal(err)
	}
	if !ac.IsMatch([]byte{255}) {
		t.Fatal("expected match for byte 255")
	}
}

func TestBuildMemoryShape(t *testing.T) {
	if testing.Short() {
		t.Skip("memory shape test is skipped in short mode")
	}

	patterns := makeSparsePatterns(1024, 48)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	ac, err := NewBuilder().AddStrings(patterns).Build()
	if err != nil {
		t.Fatal(err)
	}

	runtime.GC()
	runtime.ReadMemStats(&after)

	states := ac.StateCount()
	dfaBytes := ac.dfa.MemoryUsage()
	transBytes := states * ac.dfa.stride * 4
	if dfaBytes < transBytes {
		t.Fatalf("dfa memory usage %d smaller than transition table %d", dfaBytes, transBytes)
	}

	t.Logf("patterns=%d pattern_len=%d states=%d alphabet=%d stride=%d dfa_mib=%.2f heap_delta_mib=%.2f",
		len(patterns), len(patterns[0]), states, ac.dfa.alphabetLen, ac.dfa.stride,
		float64(dfaBytes)/(1024*1024),
		float64(after.Alloc-before.Alloc)/(1024*1024))
}

func TestBuildStress(t *testing.T) {
	if os.Getenv("AC_STRESS") == "" {
		t.Skip("set AC_STRESS=1 to run the large AC build stress repro")
	}

	n := envInt("AC_STRESS_PATTERNS", 50000)
	patternLen := envInt("AC_STRESS_PATTERN_LEN", 64)
	workers := envInt("AC_STRESS_WORKERS", 1)
	if n <= 0 || patternLen <= 0 || workers <= 0 {
		t.Fatalf("invalid stress settings: patterns=%d pattern_len=%d workers=%d", n, patternLen, workers)
	}

	patterns := makeSparsePatterns(n, patternLen)

	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ac, err := NewBuilder().AddStrings(patterns).Build()
			if err != nil {
				errs <- err
				return
			}
			t.Logf("stress automaton states=%d dfa_mib=%.2f", ac.StateCount(), float64(ac.dfa.MemoryUsage())/(1024*1024))
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func BenchmarkBuildSparsePatterns(b *testing.B) {
	for _, n := range []int{1000, 5000, 20000} {
		patterns := makeSparsePatterns(n, 64)
		b.Run("patterns_"+itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ac, err := NewBuilder().AddStrings(patterns).Build()
				if err != nil {
					b.Fatal(err)
				}
				if i == 0 {
					b.Logf("states=%d dfa_mib=%.2f", ac.StateCount(), float64(ac.dfa.MemoryUsage())/(1024*1024))
				}
			}
		})
	}
}

func makeSparsePatterns(n, patternLen int) []string {
	patterns := make([]string, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, patternLen)
		x := uint32(i)*2654435761 + 2246822519
		for j := range buf {
			x ^= x << 13
			x ^= x >> 17
			x ^= x << 5
			buf[j] = byte(33 + x%94)
		}
		patterns[i] = string(buf)
	}
	return patterns
}

func envInt(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
