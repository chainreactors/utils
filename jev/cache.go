package jev

import (
	"container/list"
	"context"
	"crypto/sha256"
	"encoding/json"
	"sync"
)

const DefaultCacheSize = 4096

// Cached adds an exact, bounded in-memory cache and coalesces identical
// concurrent batches. Wrap Provider yourself for other storage or policies.
// A non-positive capacity disables this wrapper. Failed batches are not cached.
func Cached(p Provider, capacity int) Provider {
	if p == nil || capacity <= 0 {
		return p
	}
	return &cachedProvider{Provider: p, capacity: capacity, order: list.New(),
		entries: map[[32]byte]*list.Element{}, pending: map[[32]byte]*pendingBatch{}}
}

type cachedProvider struct {
	Provider
	capacity int
	mu       sync.Mutex
	order    *list.List
	entries  map[[32]byte]*list.Element
	pending  map[[32]byte]*pendingBatch
}

type cacheEntry struct {
	key     [32]byte
	rulings map[string]Ruling
}

type pendingBatch struct {
	done    chan struct{}
	rulings map[string]Ruling
	err     error
}

func (p *cachedProvider) Judge(ctx context.Context, state interface{}, claims map[string]Claim) (map[string]Ruling, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateClaims(claims); err != nil {
		return nil, err
	}
	data, err := json.Marshal([]interface{}{p.ID(), state, claims})
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256(data)
	p.mu.Lock()
	if entry := p.entries[key]; entry != nil {
		p.order.MoveToFront(entry)
		out := copyRulings(entry.Value.(*cacheEntry).rulings)
		p.mu.Unlock()
		return out, nil
	}
	if pending := p.pending[key]; pending != nil {
		p.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-pending.done:
			if pending.err != nil {
				return nil, pending.err
			}
			return copyRulings(pending.rulings), nil
		}
	}
	pending := &pendingBatch{done: make(chan struct{})}
	p.pending[key] = pending
	p.mu.Unlock()

	got, err := p.Provider.Judge(ctx, state, claims)
	if err == nil {
		err = ValidateRulings(claims, got)
	}
	p.mu.Lock()
	if err == nil {
		pending.rulings = copyRulings(got)
		p.entries[key] = p.order.PushFront(&cacheEntry{key, pending.rulings})
		if p.order.Len() > p.capacity {
			last := p.order.Back()
			delete(p.entries, last.Value.(*cacheEntry).key)
			p.order.Remove(last)
		}
	}
	pending.err = err
	delete(p.pending, key)
	close(pending.done)
	p.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return copyRulings(pending.rulings), nil
}
