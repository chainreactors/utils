package jev

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
)

type providerFunc struct {
	id   string
	call func(context.Context, interface{}, map[string]Claim) (map[string]Ruling, error)
}

func (p *providerFunc) ID() string { return p.id }
func (p *providerFunc) Judge(ctx context.Context, s interface{}, c map[string]Claim) (map[string]Ruling, error) {
	return p.call(ctx, s, c)
}

func TestExactCacheIdentityCloneAndEviction(t *testing.T) {
	calls := 0
	p := &providerFunc{id: "endpoint/model", call: func(_ context.Context, _ interface{}, cs map[string]Claim) (map[string]Ruling, error) {
		calls++
		out := map[string]Ruling{}
		for k := range cs {
			out[k] = Ruling{Option: "yes", Confidence: 1}
		}
		return out, nil
	}}
	cached := Cached(p, 2)
	claims := map[string]Claim{"c": testClaim("is it running?")}
	ask := func(state string) map[string]Ruling {
		t.Helper()
		r, e := cached.Judge(context.Background(), state, claims)
		if e != nil {
			t.Fatal(e)
		}
		return r
	}
	first := ask("a")
	first["c"] = Ruling{Option: "no"}
	if ask("a")["c"].Option != "yes" || calls != 1 {
		t.Fatal("cache shares output or misses identical input")
	}
	ask("b")
	ask("a")
	ask("c")
	ask("b")
	if calls != 4 {
		t.Fatalf("LRU calls=%d", calls)
	}
	p.id = "endpoint/other-model"
	ask("b")
	p.id = "other-endpoint/other-model"
	ask("b")
	claim := claims["c"]
	claim.Options["yes"] = Option{"new evidence meaning", Holds}
	claims["c"] = claim
	ask("b")
	claim.Statement = "another claim"
	claims["c"] = claim
	ask("b")
	if calls != 8 {
		t.Fatalf("identity/claim changes reused cached output: %d", calls)
	}
}

func TestCacheDoesNotStoreFailuresOrInvalidBatches(t *testing.T) {
	for _, mode := range []string{"error", "missing", "unknown", "nan", "extra"} {
		t.Run(mode, func(t *testing.T) {
			calls := 0
			p := &providerFunc{call: func(_ context.Context, _ interface{}, cs map[string]Claim) (map[string]Ruling, error) {
				calls++
				if calls > 1 {
					return map[string]Ruling{"c": {Option: "yes", Confidence: 1}}, nil
				}
				switch mode {
				case "error":
					return nil, errors.New("unavailable")
				case "missing":
					return nil, nil
				case "unknown":
					return map[string]Ruling{"c": {Option: "invented", Confidence: 1}}, nil
				case "nan":
					return map[string]Ruling{"c": {Option: "yes", Confidence: math.NaN()}}, nil
				default:
					return map[string]Ruling{"c": {Option: "yes", Confidence: 1}, "extra": {}}, nil
				}
			}}
			c := Cached(p, 1)
			claims := map[string]Claim{"c": testClaim("?")}
			if _, err := c.Judge(context.Background(), "state", claims); err == nil {
				t.Fatal("invalid batch succeeded")
			}
			for i := 0; i < 2; i++ {
				if _, err := c.Judge(context.Background(), "state", claims); err != nil {
					t.Fatal(err)
				}
			}
			if calls != 2 {
				t.Fatalf("failed batch cached, or good batch missed: %d", calls)
			}
		})
	}
}

func TestCacheConcurrentSharingAndCancellation(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var calls int64
	p := &providerFunc{call: func(ctx context.Context, _ interface{}, _ map[string]Claim) (map[string]Ruling, error) {
		atomic.AddInt64(&calls, 1)
		close(started)
		select {
		case <-release:
			return map[string]Ruling{"c": {Option: "yes", Confidence: 1}}, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}}
	c := Cached(p, 2)
	claims := map[string]Claim{"c": testClaim("?")}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, e := c.Judge(context.Background(), "s", claims); e != nil {
			t.Error(e)
		}
	}()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	canceled := make(chan error, 1)
	go func() { _, e := c.Judge(ctx, "s", claims); canceled <- e }()
	cancel()
	if !errors.Is(<-canceled, context.Canceled) {
		t.Fatal("waiter did not cancel")
	}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := c.Judge(context.Background(), "s", claims)
			if e != nil || r["c"].Option != "yes" {
				t.Errorf("ruling=%v error=%v", r, e)
			}
		}()
	}
	close(release)
	wg.Wait()
	if calls != 1 {
		t.Fatalf("duplicate provider calls: %d", calls)
	}
	if _, e := c.Judge(ctx, "s", claims); !errors.Is(e, context.Canceled) {
		t.Fatal("canceled request got cached success")
	}
}

func TestCanceledLeaderIsNotCached(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	p := &providerFunc{call: func(ctx context.Context, _ interface{}, _ map[string]Claim) (map[string]Ruling, error) {
		calls++
		if calls == 1 {
			cancel()
			return nil, ctx.Err()
		}
		return map[string]Ruling{"c": {Option: "yes", Confidence: 1}}, nil
	}}
	c := Cached(p, 1)
	claims := map[string]Claim{"c": testClaim("?")}
	if _, e := c.Judge(ctx, "s", claims); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if _, e := c.Judge(context.Background(), "s", claims); e != nil || calls != 2 {
		t.Fatalf("retry %v %d", e, calls)
	}
}
