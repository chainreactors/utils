package jev

import (
	"fmt"
	"math"
	"strings"
)

// Claim defines a statement and the meaning of each finite option. Evidence
// is shared by the batch passed to Judge. Every claim explicitly abstains.
type Claim struct {
	Statement string            `json:"statement"`
	Options   map[string]Option `json:"options"`
}

// Option is part of a Claim, not a separate judgement mechanism.
type Option struct {
	Description string  `json:"description,omitempty"`
	Outcome     Outcome `json:"outcome"`
}

// Outcome is a resolved Claim meaning. The zero value means unjudged.
// String values are shared by live results, audit records and JSON snapshots.
type Outcome string

const (
	Insufficient Outcome = "insufficient"
	Holds        Outcome = "holds"
	Refuted      Outcome = "refuted"
)

func (o Outcome) String() string { return string(o) }
func (o Outcome) valid() bool    { return o == Insufficient || o == Holds || o == Refuted }

// Ruling records the original selection and its confidence. Its outcome is
// always derived from the claim; low confidence does not rewrite the option.
type Ruling struct {
	Option     string  `json:"option"`
	Confidence float64 `json:"confidence"`
}

// OptionInsufficient is the explicit abstention key required by every Claim.
const OptionInsufficient = "insufficient"

func probability(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1
}

func (c Claim) validate() error {
	if strings.TrimSpace(c.Statement) == "" {
		return fmt.Errorf("a claim requires a statement")
	}
	if len(c.Options) < 2 {
		return fmt.Errorf("a claim requires options and an explicit insufficient option")
	}
	abstain, ok := c.Options[OptionInsufficient]
	if !ok || abstain.Outcome != Insufficient {
		return fmt.Errorf("insufficient must explicitly mean Insufficient")
	}
	for key, option := range c.Options {
		if strings.TrimSpace(key) == "" || !option.Outcome.valid() {
			return fmt.Errorf("invalid option %q", key)
		}
	}
	return nil
}

func (c Claim) validateRuling(r Ruling) error {
	if _, ok := c.Options[r.Option]; !ok {
		return fmt.Errorf("unknown option %q", r.Option)
	}
	if !probability(r.Confidence) {
		return fmt.Errorf("invalid confidence %v", r.Confidence)
	}
	return nil
}

// Resolve is the only interpretation of a ruling. Invalid input cannot
// establish a claim; execution reports invalid inputs as errors beforehand.
func (c Claim) Resolve(r Ruling, minConfidence float64) Outcome {
	if c.validate() != nil || c.validateRuling(r) != nil || !probability(minConfidence) || r.Confidence < minConfidence {
		return Insufficient
	}
	return c.Options[r.Option].Outcome
}

// ValidateClaims checks inputs before a Provider is invoked.
func ValidateClaims(claims map[string]Claim) error {
	for key, c := range claims {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("jev: empty claim key")
		}
		if err := c.validate(); err != nil {
			return fmt.Errorf("jev: claim %q: %w", key, err)
		}
	}
	return nil
}

// ValidateRulings checks the complete Claim/Ruling contract, including exact keys.
// Provider wrappers can use it before persisting results.
func ValidateRulings(claims map[string]Claim, rulings map[string]Ruling) error {
	if err := ValidateClaims(claims); err != nil {
		return err
	}
	if len(rulings) != len(claims) {
		return fmt.Errorf("jev: provider returned %d rulings for %d claims", len(rulings), len(claims))
	}
	for key, c := range claims {
		r, ok := rulings[key]
		if !ok {
			return fmt.Errorf("jev: provider omitted claim %q", key)
		}
		if err := c.validateRuling(r); err != nil {
			return fmt.Errorf("jev: claim %q: %w", key, err)
		}
	}
	return nil
}
