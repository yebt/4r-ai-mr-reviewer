package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
)

// responseDTO mirrors the JSON contract the model is asked to return.
type responseDTO struct {
	Summary  string       `json:"summary"`
	Findings []findingDTO `json:"findings"`
}

type findingDTO struct {
	Dimension string `json:"dimension"`
	Severity  string `json:"severity"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Issue     string `json:"issue"`
	Why       string `json:"why"`
	Fix       string `json:"fix"`
	Blocking  bool   `json:"blocking"`
}

// ParseError is returned when the model's output cannot be decoded into the
// review contract. It carries the raw model output so callers can persist and
// surface exactly what the model returned, plus the phase and token counts the
// engine enriches it with so a failed review can show where and at what cost the
// failure happened. Phase/InputTokens/OutputTokens are optional (zero when the
// producer did not set them).
type ParseError struct {
	Raw          string // the raw model output that failed to parse
	Err          error  // the underlying decode error
	Phase        string // the 4R phase whose response failed (empty when unknown)
	InputTokens  int    // input tokens accumulated up to the failure (0 when unknown)
	OutputTokens int    // output tokens accumulated up to the failure (0 when unknown)
}

func (e *ParseError) Error() string { return e.Err.Error() }
func (e *ParseError) Unwrap() error { return e.Err }

// parseResponse extracts and decodes the model's JSON, mapping it to domain
// findings. It tolerates responses wrapped in markdown code fences or padded
// with surrounding prose.
func parseResponse(content string) (summary string, findings []review.Finding, err error) {
	// An empty response is a distinct, human failure mode (the model produced no
	// output at all) — surface it plainly rather than as a cryptic JSON error.
	if strings.TrimSpace(content) == "" {
		return "", nil, &ParseError{Raw: content, Err: errors.New("the model returned an empty response")}
	}
	raw := extractJSON(content)

	var dto responseDTO
	if err := json.Unmarshal([]byte(raw), &dto); err != nil {
		return "", nil, &ParseError{Raw: content, Err: fmt.Errorf("could not parse the model's response as JSON: %w", err)}
	}

	out := make([]review.Finding, 0, len(dto.Findings))
	for _, f := range dto.Findings {
		out = append(out, review.Finding{
			Dimension: review.Dimension(strings.ToLower(f.Dimension)),
			Severity:  review.Severity(strings.ToLower(f.Severity)),
			File:      f.File,
			Line:      f.Line,
			Issue:     f.Issue,
			Why:       f.Why,
			Fix:       f.Fix,
			Blocking:  f.Blocking,
		})
	}
	return dto.Summary, out, nil
}

// extractJSON pulls the JSON object out of a model response that may include
// code fences or leading/trailing prose.
func extractJSON(s string) string {
	s = strings.TrimSpace(s)

	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}

	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return s
}
