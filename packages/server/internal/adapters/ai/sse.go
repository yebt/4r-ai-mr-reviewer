package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// streamIdleTimeout aborts a stream that goes quiet for this long. It bounds the
// gap BETWEEN tokens, not the total duration, so an arbitrarily long-but-flowing
// generation (including a reasoning model's silent-looking thinking phase, which
// still streams thinking deltas) runs to completion; only a truly stalled
// connection is killed. Overridable via AIR_AI_STREAM_IDLE_TIMEOUT (seconds); a
// var so tests can shrink it.
var streamIdleTimeout = envIdleTimeout()

// envIdleTimeout reads AIR_AI_STREAM_IDLE_TIMEOUT (whole seconds), defaulting to
// 90s when unset or invalid.
func envIdleTimeout() time.Duration {
	if v := strings.TrimSpace(os.Getenv("AIR_AI_STREAM_IDLE_TIMEOUT")); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			return time.Duration(secs) * time.Second
		}
	}
	return 90 * time.Second
}

// errStreamDone is returned by an event callback to end consumption cleanly (the
// provider's terminal marker: OpenAI's "[DONE]", Anthropic's "message_stop").
var errStreamDone = errors.New("ai: stream done")

// streamDefault reports whether new clients stream by default. Streaming is on
// unless AIR_AI_STREAM is explicitly set to "false", giving an escape hatch for
// an endpoint that does not support server-sent events.
func streamDefault() bool {
	return !strings.EqualFold(strings.TrimSpace(os.Getenv("AIR_AI_STREAM")), "false")
}

// streamSSE marshals in (already marked stream:true by the caller), POSTs it, and
// dispatches each server-sent event's (event, data) to onEvent until the stream
// ends or onEvent returns errStreamDone. It applies the SAME 429/5xx retry policy
// as postJSON to the INITIAL response status. Once the stream is flowing it is
// bounded by an idle timeout (reset on every line): a stall aborts, but a slow,
// steadily-progressing generation does not. A mid-stream failure is NOT retried —
// a long generation is not a transient error, and retrying re-pays the full cost.
func streamSSE(ctx context.Context, hc *http.Client, provider, url string, headers map[string]string, in any, onEvent func(event string, data []byte) error) error {
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
				delay = retryBaseDelay << (attempt - 1)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		done, err := streamOnce(ctx, hc, provider, url, headers, buf, onEvent)
		if done {
			return err
		}
		// Not done: a retryable transport or status error. Keep it and back off.
		lastErr = err
		retryAfter = 0
		var apiErr *APIError
		if errors.As(err, &apiErr) {
			retryAfter = parseRetryAfter(apiErr.retryAfter)
		}
	}
	return lastErr
}

// streamOnce performs a single attempt. done reports whether the caller should
// stop (success, non-retryable error, or context cancellation); when done is
// false the returned error is retryable (a transient transport failure or a
// 429/5xx status).
func streamOnce(ctx context.Context, hc *http.Client, provider, url string, headers map[string]string, buf []byte, onEvent func(event string, data []byte) error) (done bool, err error) {
	// Idle-bounded child context: the watchdog cancels it if no line arrives within
	// streamIdleTimeout. idle records that the abort was ours (a stall), so it is
	// reported as an idle timeout rather than a bare context.Canceled.
	reqCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var idle atomic.Bool
	timer := time.AfterFunc(streamIdleTimeout, func() {
		idle.Store(true)
		cancel()
	})
	defer timer.Stop()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return true, fmt.Errorf("ai: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := hc.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err() // parent cancelled: stop
		}
		if idle.Load() {
			return true, fmt.Errorf("ai: %s stream idle timeout after %s", provider, streamIdleTimeout)
		}
		return false, fmt.Errorf("ai: request: %w", err) // transport blip: retry
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
		apiErr := &APIError{Provider: provider, Status: resp.StatusCode, Body: string(body), retryAfter: resp.Header.Get("Retry-After")}
		if retryableStatus(resp.StatusCode) {
			return false, apiErr // 429/5xx: retry
		}
		return true, apiErr // other 4xx (e.g. a thinking 400): surface unchanged
	}

	if err := consumeSSE(resp.Body, timer, onEvent); err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		if idle.Load() {
			return true, fmt.Errorf("ai: %s stream idle timeout after %s", provider, streamIdleTimeout)
		}
		return true, err // a parse error is not retryable
	}
	return true, nil
}

// consumeSSE reads the SSE body line by line, assembling events at each blank
// line and dispatching them. It resets the idle timer on every line so progress
// keeps the stream alive.
func consumeSSE(r io.Reader, timer *time.Timer, onEvent func(event string, data []byte) error) error {
	sc := bufio.NewScanner(r)
	// SSE data lines are normally tiny (one token), but a provider may batch a
	// large chunk into one line; allow up to the shared response cap.
	sc.Buffer(make([]byte, 0, 64*1024), maxResponseBytes)

	var event string
	var data bytes.Buffer
	dispatch := func() error {
		if data.Len() == 0 {
			event = ""
			return nil
		}
		payload := bytes.TrimRight(data.Bytes(), "\n")
		out := append([]byte(nil), payload...)
		ev := event
		event = ""
		data.Reset()
		return onEvent(ev, out)
	}

	for sc.Scan() {
		timer.Reset(streamIdleTimeout)
		line := sc.Text()
		switch {
		case line == "": // event boundary
			if err := dispatch(); err != nil {
				if errors.Is(err, errStreamDone) {
					return nil
				}
				return err
			}
		case strings.HasPrefix(line, ":"): // comment / heartbeat
			continue
		default:
			field, val, _ := strings.Cut(line, ":")
			val = strings.TrimPrefix(val, " ")
			switch field {
			case "event":
				event = val
			case "data":
				if data.Len() > 0 {
					data.WriteByte('\n')
				}
				data.WriteString(val)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("ai: read stream: %w", err)
	}
	// Flush a trailing event that had no terminating blank line.
	if err := dispatch(); err != nil && !errors.Is(err, errStreamDone) {
		return err
	}
	return nil
}

// streamAnthropic consumes the Anthropic messages SSE stream, accumulating text
// and thinking deltas plus usage/model into out (the same shape the buffered
// decode produces, so parseAnthropicContent works unchanged).
func streamAnthropic(ctx context.Context, hc *http.Client, url string, headers map[string]string, body anthropicRequest, out *anthropicResponse) error {
	var text, thinking strings.Builder
	onEvent := func(event string, data []byte) error {
		switch event {
		case "message_start":
			var e struct {
				Message struct {
					Model string         `json:"model"`
					Usage anthropicUsage `json:"usage"`
				} `json:"message"`
			}
			if err := json.Unmarshal(data, &e); err == nil {
				out.Model = e.Message.Model
				out.Usage.InputTokens = e.Message.Usage.InputTokens
			}
		case "content_block_delta":
			var e struct {
				Delta struct {
					Type     string `json:"type"`
					Text     string `json:"text"`
					Thinking string `json:"thinking"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(data, &e); err == nil {
				switch e.Delta.Type {
				case "text_delta":
					text.WriteString(e.Delta.Text)
				case "thinking_delta":
					thinking.WriteString(e.Delta.Thinking)
				}
			}
		case "message_delta":
			var e struct {
				Usage anthropicUsage `json:"usage"`
			}
			if err := json.Unmarshal(data, &e); err == nil && e.Usage.OutputTokens > 0 {
				out.Usage.OutputTokens = e.Usage.OutputTokens
			}
		case "error":
			var e struct {
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			_ = json.Unmarshal(data, &e)
			return fmt.Errorf("ai: anthropic stream error: %s: %s", e.Error.Type, e.Error.Message)
		case "message_stop":
			return errStreamDone
		}
		return nil
	}

	if err := streamSSE(ctx, hc, "anthropic", url, headers, body, onEvent); err != nil {
		return err
	}
	if text.Len() > 0 {
		out.Content = append(out.Content, anthropicContentBlock{Type: "text", Text: text.String()})
	}
	if thinking.Len() > 0 {
		out.Content = append(out.Content, anthropicContentBlock{Type: "thinking", Thinking: thinking.String()})
	}
	return nil
}

// streamOpenAI consumes an OpenAI-compatible chat-completions SSE stream,
// accumulating content and reasoning deltas plus usage/model into out.
func streamOpenAI(ctx context.Context, hc *http.Client, url string, headers map[string]string, body openaiRequest, out *openaiResponse) error {
	var content, reasoning strings.Builder
	var gotChoice bool
	onEvent := func(event string, data []byte) error {
		if string(data) == "[DONE]" {
			return errStreamDone
		}
		var chunk struct {
			Model   string `json:"model"`
			Choices []struct {
				Delta openaiRespMessage `json:"delta"`
			} `json:"choices"`
			Usage *openaiUsage `json:"usage"`
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			return nil // skip a keepalive or non-JSON frame
		}
		if chunk.Model != "" {
			out.Model = chunk.Model
		}
		if len(chunk.Choices) > 0 {
			gotChoice = true
			d := chunk.Choices[0].Delta
			content.WriteString(d.Content)
			reasoning.WriteString(firstNonEmpty(d.ReasoningContent, d.Reasoning))
		}
		if chunk.Usage != nil {
			out.Usage = *chunk.Usage
		}
		return nil
	}

	if err := streamSSE(ctx, hc, "openai-compat", url, headers, body, onEvent); err != nil {
		return err
	}
	if gotChoice {
		out.Choices = []openaiChoice{{Message: openaiRespMessage{
			Content:   content.String(),
			Reasoning: reasoning.String(),
		}}}
	}
	return nil
}
