package jev

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestClaimAndRulingJSON(t *testing.T) {
	claim := testClaim("The response is produced by this product.")
	data, err := json.Marshal(claim)
	if err != nil {
		t.Fatal(err)
	}
	var restored Claim
	if err := json.Unmarshal(data, &restored); err != nil || !reflect.DeepEqual(claim, restored) {
		t.Fatalf("claim roundtrip: %s, %v", data, err)
	}
	ruling := Ruling{Option: "yes", Confidence: 0.2}
	data, err = json.Marshal(ruling)
	if err != nil || string(data) != `{"option":"yes","confidence":0.2}` {
		t.Fatalf("provider vocabulary leaked into ruling: %s, %v", data, err)
	}
	var got Ruling
	if err := json.Unmarshal(data, &got); err != nil || got != ruling || restored.Resolve(got, 0.5) != Insufficient || restored.Resolve(got, 0.1) != Holds {
		t.Fatalf("ruling or interpretation changed: %+v, %v", got, err)
	}
	var unjudged Outcome
	for _, outcome := range []Outcome{unjudged, Holds, Refuted, Insufficient} {
		data, err := json.Marshal(outcome)
		var decoded Outcome
		if err != nil || json.Unmarshal(data, &decoded) != nil || decoded != outcome {
			t.Fatalf("outcome roundtrip: %q, %s, %v", outcome, data, err)
		}
	}
	if unjudged == Insufficient || unjudged.valid() {
		t.Fatal("unjudged outcome became an abstention")
	}
}

func TestInvalidClaimsNeverReachProvider(t *testing.T) {
	for _, change := range []string{"statement", "batch key", "option key", "missing outcome", "unknown outcome"} {
		t.Run(change, func(t *testing.T) {
			claim := testClaim("Is this product running?")
			key := "product"
			switch change {
			case "statement":
				claim.Statement = " \n"
			case "batch key":
				key = " "
			case "option key":
				claim.Options[" "] = Option{Outcome: Holds}
			case "missing outcome":
				claim.Options["yes"] = Option{}
			case "unknown outcome":
				claim.Options["yes"] = Option{Outcome: "accepted"}
			}
			provider := answerProvider(func(map[string]Claim) map[string]Ruling {
				t.Fatal("invalid claim reached provider")
				return nil
			})
			claims := map[string]Claim{key: claim}
			if _, err := Judge(context.Background(), provider, "state", claims); err == nil {
				t.Fatal("Judge accepted invalid claim")
			}
			if _, err := Cached(provider, 1).Judge(context.Background(), "state", claims); err == nil {
				t.Fatal("cache accepted invalid claim")
			}
		})
	}
}

const testInsufficient = "The evidence decides neither way."

func testClaim(statement string) Claim {
	return Claim{Statement: statement, Options: map[string]Option{"yes": {"", Holds}, "no": {"", Refuted}, OptionInsufficient: {testInsufficient, Insufficient}}}
}

type answerProvider func(map[string]Claim) map[string]Ruling

func (answerProvider) ID() string { return "answer-provider" }

func (p answerProvider) Judge(_ context.Context, _ interface{}, questions map[string]Claim) (map[string]Ruling, error) {
	return p(questions), nil
}

func TestExplicitAbstentionAndCompleteContract(t *testing.T) {
	claim := testClaim("?")
	for _, confidence := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		p := answerProvider(func(map[string]Claim) map[string]Ruling {
			return map[string]Ruling{"c": {Option: "yes", Confidence: confidence}}
		})
		if _, e := Judge(context.Background(), p, "state", map[string]Claim{"c": claim}); e == nil {
			t.Fatalf("accepted confidence %v", confidence)
		}
	}
	for _, conflict := range []bool{false, true} {
		c := testClaim("?")
		if conflict {
			c.Options[OptionInsufficient] = Option{Outcome: Holds}
		} else {
			delete(c.Options, OptionInsufficient)
		}
		called := false
		p := answerProvider(func(map[string]Claim) map[string]Ruling { called = true; return nil })
		if _, e := Judge(context.Background(), p, nil, map[string]Claim{"c": c}); e == nil || called {
			t.Fatal("invalid abstention reached provider")
		}
	}
	if r, e := Judge(context.Background(), nil, nil, nil); e != nil || len(r) != 0 {
		t.Fatal("empty batch required a provider")
	}
	if _, e := Judge(context.Background(), nil, nil, map[string]Claim{"c": claim}); e == nil {
		t.Fatal("missing provider accepted")
	}
}
