package engine

import (
	"encoding/json"
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

// parseResponse extracts and decodes the model's JSON, mapping it to domain
// findings. It tolerates responses wrapped in markdown code fences or padded
// with surrounding prose.
func parseResponse(content string) (summary string, findings []review.Finding, err error) {
	raw := extractJSON(content)

	var dto responseDTO
	if err := json.Unmarshal([]byte(raw), &dto); err != nil {
		return "", nil, fmt.Errorf("decode model output: %w", err)
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
