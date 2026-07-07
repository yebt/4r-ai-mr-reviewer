// Package ai implements the llm.Client contract for the supported providers:
// OpenAI-compatible endpoints (Groq, OpenAI, Moonshot, Kimi, OpenRouter) and
// the Anthropic messages API.
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// maxResponseBytes caps how much of a provider response we read, so a
// misbehaving endpoint cannot exhaust memory.
const maxResponseBytes = 16 << 20

// APIError is returned for non-2xx responses from a provider.
type APIError struct {
	Provider string
	Status   int
	Body     string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("ai: %s returned status %d: %s", e.Provider, e.Status, strings.TrimSpace(e.Body))
}

// postJSON marshals in, POSTs it to url with the given headers, and decodes a
// 2xx JSON response into out. providerName is only used for error messages.
func postJSON(ctx context.Context, hc *http.Client, providerName, url string, headers map[string]string, in, out any) error {
	buf, err := json.Marshal(in)
	if err != nil {
		return fmt.Errorf("ai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := hc.Do(req)
	if err != nil {
		return fmt.Errorf("ai: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("ai: read body: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{Provider: providerName, Status: resp.StatusCode, Body: string(body)}
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("ai: decode response: %w", err)
	}
	return nil
}
