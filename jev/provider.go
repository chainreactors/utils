package jev

import (
	"context"
	"encoding/json"
	"fmt"
)

// Provider evaluates finite-option claims against shared evidence. It returns
// the original selections; Claim.Resolve alone interprets their meaning.
// Implementations must not mutate evidence, claims or their options.
type Provider interface {
	// ID identifies the endpoint and model. Change it when answers may change.
	ID() string
	Judge(context.Context, interface{}, map[string]Claim) (map[string]Ruling, error)
}

// Judge validates a batch, freezes the evidence as JSON once so every provider
// sees the same bytes, and validates the complete answer. A failed batch
// returns no rulings. Resolve each ruling against its claim before acting.
func Judge(ctx context.Context, p Provider, state interface{}, claims map[string]Claim) (map[string]Ruling, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateClaims(claims); err != nil {
		return nil, err
	}
	if len(claims) == 0 {
		return map[string]Ruling{}, nil
	}
	if p == nil {
		return nil, fmt.Errorf("jev: no provider")
	}
	data, err := json.Marshal(state)
	if err != nil {
		return nil, err
	}
	got, err := p.Judge(ctx, json.RawMessage(data), claims)
	if err != nil {
		return nil, err
	}
	if err := ValidateRulings(claims, got); err != nil {
		return nil, err
	}
	return copyRulings(got), nil
}

func copyRulings(in map[string]Ruling) map[string]Ruling {
	out := make(map[string]Ruling, len(in))
	for key, r := range in {
		out[key] = r
	}
	return out
}
