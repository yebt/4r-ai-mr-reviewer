package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// seqClient returns a queued response per call, optionally attaching reasoning
// and recording the thinking budget it was asked for.
type seqClient struct {
	responses  []string
	reasoning  string
	i          int
	err        error
	gotBudgets []int
}

func (c *seqClient) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	c.gotBudgets = append(c.gotBudgets, req.ThinkingBudget)
	if c.err != nil {
		return llm.Response{}, c.err
	}
	content := `{"findings":[]}`
	if c.i < len(c.responses) {
		content = c.responses[c.i]
	}
	c.i++
	return llm.Response{Content: content, InputTokens: 10, OutputTokens: 5, Reasoning: c.reasoning}, nil
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
	rv, err := mp.Run(context.Background(), client, RunParams{
		Model:   "m",
		In:      sampleInput(),
		OnPhase: func(p string) { phases = append(phases, p) },
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
	if _, err := mp.Run(context.Background(), client, RunParams{Model: "m", In: sampleInput()}); err == nil {
		t.Fatal("expected client error to propagate")
	}
}

func TestMultiPassEmptyDiff(t *testing.T) {
	mp := newMultiPass(t)
	in := sampleInput()
	in.Diff = ""
	if _, err := mp.Run(context.Background(), &seqClient{}, RunParams{Model: "m", In: in}); err == nil {
		t.Fatal("expected empty-diff error")
	}
}

func TestMultiPassCapturesReasoning(t *testing.T) {
	mp := newMultiPass(t)
	client := &seqClient{reasoning: "because X"}

	type rc struct {
		phase, text string
	}
	var got []rc
	_, err := mp.Run(context.Background(), client, RunParams{
		Model:          "m",
		ThinkingBudget: 2048,
		In:             sampleInput(),
		OnReasoning:    func(phase, text string) { got = append(got, rc{phase, text}) },
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// One reasoning callback per 4R lens, in order.
	want := []rc{
		{"risk", "because X"},
		{"readability", "because X"},
		{"reliability", "because X"},
		{"resilience", "because X"},
	}
	if len(got) != len(want) {
		t.Fatalf("reasoning callbacks = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("reasoning[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
	// The thinking budget must be threaded into every request.
	for i, b := range client.gotBudgets {
		if b != 2048 {
			t.Fatalf("request[%d] budget = %d, want 2048", i, b)
		}
	}
}

func TestMultiPassSkipsEmptyReasoning(t *testing.T) {
	mp := newMultiPass(t)
	client := &seqClient{} // no reasoning returned

	called := false
	if _, err := mp.Run(context.Background(), client, RunParams{
		Model:       "m",
		In:          sampleInput(),
		OnReasoning: func(string, string) { called = true },
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if called {
		t.Fatal("OnReasoning must not be called when reasoning is empty")
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

	_, err := mp.Run(ctx, &failClient{t: t}, RunParams{Model: "m", In: sampleInput()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

// cancelAfterClient returns a canned response and fires cancel once it has been
// called `after` times, simulating an adapter that ignores ctx: the run's
// per-pass cooperative check must then trip on the next pass.
type cancelAfterClient struct {
	cancel    context.CancelFunc
	after     int
	calls     int
	reasoning string
}

func (c *cancelAfterClient) Complete(_ context.Context, _ llm.Request) (llm.Response, error) {
	c.calls++
	resp := llm.Response{Content: `{"findings":[]}`, InputTokens: 1, OutputTokens: 1, Reasoning: c.reasoning}
	if c.calls == c.after {
		c.cancel()
	}
	return resp, nil
}

// TestMultiPassCancelBetweenPasses cancels the context after the first pass
// completes (not before the first): the reasoning captured for that completed
// phase must already have been reported, and the run must then abort cleanly
// with a context error before any later pass runs.
func TestMultiPassCancelBetweenPasses(t *testing.T) {
	mp := newMultiPass(t)
	ctx, cancel := context.WithCancel(context.Background())
	client := &cancelAfterClient{cancel: cancel, after: 1, reasoning: "because X"}

	var captured []string
	_, err := mp.Run(ctx, client, RunParams{
		Model:          "m",
		ThinkingBudget: 2048,
		In:             sampleInput(),
		OnReasoning:    func(phase, _ string) { captured = append(captured, phase) },
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	// Only the first (risk) pass completed before cancellation, so exactly its
	// reasoning was reported; later passes never ran.
	if len(captured) != 1 || captured[0] != "risk" {
		t.Fatalf("captured reasoning phases = %v, want [risk]", captured)
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
