// Package jev asks a model to rule on claims about shared evidence.
//
// A Claim is a statement with a finite set of options; each option declares
// what choosing it means (Holds, Refuted or Insufficient), and every claim
// offers an explicit "insufficient" abstention. A Provider picks one option
// per claim with a confidence; Claim.Resolve is the only interpretation of
// that pick. The model never writes free text back.
//
//	c, err := jev.NewClient("") // TYPESAFE_API_KEY
//	p := jev.Cached(c, jev.DefaultCacheSize)
//	rulings, err := jev.Judge(ctx, p, state, claims)
//	outcome := claims["k"].Resolve(rulings["k"], jev.DefaultMinConfidence)
//
// Client speaks the TypeSafe Jev (System One) HTTP API
// (https://docs.typesafe.ai/api.md). Other models plug in as a Provider.
package jev
