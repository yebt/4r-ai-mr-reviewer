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
	"strconv"
	"strings"
	"time"
)

// maxResponseBytes caps how much of a provider response we read, so a
// misbehaving endpoint cannot exhaust memory.
const maxResponseBytes = 16 << 20

// Retry policy for transient provider failures (429 and 5xx). Kept small so a
// blip on one pass of a multi-pass review does not discard the prior, already
// paid passes, without adding much latency. retryBaseDelay is a var so tests can
// shrink it; retryAfterCap bounds a hostile/large Retry-After.
const (
	maxAttempts   = 3
	retryAfterCap = 10 * time.Second
)

var retryBaseDelay = 500 * time.Millisecond

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

	var lastErr error
	var retryAfter time.Duration
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			delay := retryAfter
			if delay <= 0 {
				delay = retryBaseDelay << (attempt - 1) // exponential: 1x, 2x, ...
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
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
			// Transport error: retry the remaining attempts.
			lastErr = fmt.Errorf("ai: request: %w", err)
			retryAfter = 0
			continue
		}

		body, rerr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		resp.Body.Close()
		if rerr != nil {
			return fmt.Errorf("ai: read body: %w", rerr)
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("ai: decode response: %w", err)
			}
			return nil
		}

		apiErr := &APIError{Provider: providerName, Status: resp.StatusCode, Body: string(body)}
		if !retryableStatus(resp.StatusCode) {
			return apiErr
		}
		// Transient (429/5xx): keep the error and back off before the next attempt.
		lastErr = apiErr
		retryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
	}
	return lastErr
}

// retryableStatus reports whether a non-2xx status is worth retrying: rate limits
// (429) and server-side errors (5xx). Client errors (other 4xx) are not retried.
func retryableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// parseRetryAfter reads a Retry-After header expressed in whole seconds, capped
// so a hostile value cannot stall a request. Other forms (HTTP date) are ignored
// and the caller falls back to exponential backoff.
func parseRetryAfter(h string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(h))
	if err != nil || secs <= 0 {
		return 0
	}
	d := time.Duration(secs) * time.Second
	if d > retryAfterCap {
		return retryAfterCap
	}
	return d
}
