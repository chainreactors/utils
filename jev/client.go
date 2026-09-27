package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

const (
	DefaultEndpoint = "https://api.typesafe.ai/v1/systemone"
	// DefaultMinConfidence is the threshold chosen for the pinned Jev model.
	DefaultMinConfidence = 0.3
	DefaultModel         = "jev-1.13.0"
	EnvAPIKey            = "TYPESAFE_API_KEY"
)

// Client is the Provider that sends claims to the Jev HTTP API.
type Client struct {
	Endpoint   string
	Model      string
	APIKey     string
	HTTP       *http.Client
	MaxRetries int // on 429 / 529, with exponential backoff

	// InputTokens sent so far; read with atomic.LoadInt64.
	InputTokens int64
}

// NewClient reads the API key from TYPESAFE_API_KEY when apiKey is empty.
func NewClient(apiKey string) (*Client, error) {
	if apiKey == "" {
		apiKey = os.Getenv(EnvAPIKey)
	}
	if apiKey == "" {
		return nil, fmt.Errorf("%s is not set", EnvAPIKey)
	}
	return &Client{
		Endpoint:   DefaultEndpoint,
		Model:      DefaultModel,
		APIKey:     apiKey,
		HTTP:       &http.Client{Timeout: 30 * time.Second},
		MaxRetries: 3,
	}, nil
}

func (p *Client) ID() string { return "jev/" + p.Endpoint + "/" + p.Model }

// wireClaim is only the Jev HTTP representation. Core Claim has no wire policy.
type wireClaim struct {
	Type         string             `json:"type"`
	Instructions string             `json:"instructions"`
	Criteria     map[string]*string `json:"criteria"`
}

func toWire(q Claim) wireClaim {
	opts := make(map[string]*string, len(q.Options))
	for k, v := range q.Options {
		if v.Description == "" {
			opts[k] = nil // Jev's "no description"
		} else {
			description := v.Description
			opts[k] = &description
		}
	}
	return wireClaim{Type: "choice", Instructions: q.Statement, Criteria: opts}
}

type APIError struct {
	Status int
	Body   string
}

func (e *APIError) Error() string { return fmt.Sprintf("typesafe: http %d: %s", e.Status, e.Body) }

func (p *Client) Judge(ctx context.Context, state interface{}, claims map[string]Claim) (map[string]Ruling, error) {
	if err := ValidateClaims(claims); err != nil {
		return nil, err
	}
	req := struct {
		State     interface{}          `json:"state"`
		Model     string               `json:"model"`
		Questions map[string]wireClaim `json:"questions"`
	}{State: state, Model: p.Model, Questions: make(map[string]wireClaim, len(claims))}
	for id, claim := range claims {
		req.Questions[id] = toWire(claim)
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	backoff := time.Second
	for attempt := 0; ; attempt++ {
		rulings, err := p.do(ctx, body)
		if err == nil {
			if err := ValidateRulings(claims, rulings); err != nil {
				return nil, err
			}
			return rulings, nil
		}
		apiErr, ok := err.(*APIError)
		if !ok || (apiErr.Status != 429 && apiErr.Status != 529) || attempt >= p.MaxRetries {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		backoff *= 2
	}
}

func (p *Client) do(ctx context.Context, body []byte) (map[string]Ruling, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &APIError{Status: resp.StatusCode, Body: string(raw)}
	}
	var out struct {
		Answers map[string]struct {
			Choice     string   `json:"choice"`
			Confidence *float64 `json:"confidence"`
		} `json:"answers"`
		Usage struct {
			InputTokens int64 `json:"input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("typesafe: decode response: %w", err)
	}
	atomic.AddInt64(&p.InputTokens, out.Usage.InputTokens)
	rulings := make(map[string]Ruling, len(out.Answers))
	for id, answer := range out.Answers {
		if answer.Confidence == nil {
			return nil, fmt.Errorf("typesafe: claim %q omitted confidence", id)
		}
		rulings[id] = Ruling{Option: answer.Choice, Confidence: *answer.Confidence}
	}
	return rulings, nil
}
