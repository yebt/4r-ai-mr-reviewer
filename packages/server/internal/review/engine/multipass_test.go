package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// seqClient returns a queued response per call.
type seqClient struct {
	responses []string
	i         int
	err       error
}

func (c *seqClient) Complete(_ context.Context, _ llm.Request) (llm.Response, error) {
	if c.err != nil {
		return llm.Response{}, c.err
	}
	content := `{"findings":[]}`
	if c.i < len(c.responses) {
		content = c.responses[c.i]
	}
	c.i++
	return llm.Response{Content: content, InputTokens: 10, OutputTokens: 5}, nil
}

func newMultiPass(t *testing.T) *MultiPass {
	t.Helper()
	set, err := skills.Load("")
	if err != nil {
		t.Fatalf("skills.Load: %v", err)
	}
	return NewMultiPass(set)
}

func TestMultiPassRunsEachDimension(t *testing.T) {
	mp := newMultiPass(t)
	client := &seqClient{responses: []string{
		`{"findings":[{"severity":"high","file":"a.go","line":1,"issue":"secret","blocking":true}]}`,
		`{"findings":[{"severity":"low","file":"b.go","line":2,"issue":"naming"}]}`,
		`{"findings":[{"severity":"medium","file":"c.go","line":3,"issue":"no test"}]}`,
		`{"findings":[]}`,
	}}

	var phases []string
	rv, err := mp.Run(context.Background(), client, "m", nil, sampleInput(), func(p string) {
		phases = append(phases, p)
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if want := []string{"risk", "readability", "reliability", "resilience"}; !equalStrings(phases, want) {
		t.Fatalf("phases = %v, want %v", phases, want)
	}
	if len(rv.Findings) != 3 {
		t.Fatalf("findings = %d, want 3", len(rv.Findings))
	}
	// Each finding must be pinned to the dimension of its pass.
	if rv.Findings[0].Dimension != review.Risk ||
		rv.Findings[1].Dimension != review.Readability ||
		rv.Findings[2].Dimension != review.Reliability {
		t.Fatalf("dimensions not pinned: %+v", rv.Findings)
	}
	if rv.InputTokens != 40 || rv.OutputTokens != 20 {
		t.Fatalf("tokens = %d/%d, want 40/20", rv.InputTokens, rv.OutputTokens)
	}
	if rv.Status != review.StatusDone || rv.Recommendation != review.RequestChanges || rv.Summary == "" {
		t.Fatalf("unexpected review: status=%s rec=%s summary=%q", rv.Status, rv.Recommendation, rv.Summary)
	}
}

func TestMultiPassPropagatesError(t *testing.T) {
	mp := newMultiPass(t)
	client := &seqClient{err: errors.New("rate limited")}
	if _, err := mp.Run(context.Background(), client, "m", nil, sampleInput(), nil); err == nil {
		t.Fatal("expected client error to propagate")
	}
}

func TestMultiPassEmptyDiff(t *testing.T) {
	mp := newMultiPass(t)
	in := sampleInput()
	in.Diff = ""
	if _, err := mp.Run(context.Background(), &seqClient{}, "m", nil, in, nil); err == nil {
		t.Fatal("expected empty-diff error")
	}
}

// failClient fails the test if Complete is ever called — used to prove
// cancellation short-circuits before any LLM call.
type failClient struct{ t *testing.T }

func (c *failClient) Complete(_ context.Context, _ llm.Request) (llm.Response, error) {
	c.t.Fatal("Complete must not be called on an already-cancelled context")
	return llm.Response{}, nil
}

func TestMultiPassCancelledContext(t *testing.T) {
	mp := newMultiPass(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Run so the first pass check trips.

	_, err := mp.Run(ctx, &failClient{t: t}, "m", nil, sampleInput(), nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
