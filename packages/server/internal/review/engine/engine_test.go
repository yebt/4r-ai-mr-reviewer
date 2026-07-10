package engine

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/webcloster-dev/ai-reviewer/internal/domain/llm"
	"github.com/webcloster-dev/ai-reviewer/internal/domain/review"
	"github.com/webcloster-dev/ai-reviewer/internal/review/skills"
)

// fakeClient records the request and returns a canned response.
type fakeClient struct {
	content string
	err     error
	gotReq  llm.Request
}

func (f *fakeClient) Complete(_ context.Context, req llm.Request) (llm.Response, error) {
	f.gotReq = req
	if f.err != nil {
		return llm.Response{}, f.err
	}
	return llm.Response{Content: f.content, InputTokens: 100, OutputTokens: 50}, nil
}

func newEngine(t *testing.T) *Engine {
	t.Helper()
	set, err := skills.Load("")
	if err != nil {
		t.Fatalf("skills.Load: %v", err)
	}
	return New(set)
}

func sampleInput() Input {
	return Input{RepoID: "r1", MRIID: 7, Title: "Add login", Diff: "@@ -1 +1 @@\n-old\n+new"}
}

func TestRunCleanReview(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{content: `{"summary":"clean change","findings":[]}`}

	rv, err := e.Run(context.Background(), fc, "model-x", nil, sampleInput(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rv.Status != review.StatusDone || rv.Recommendation != review.Approve || rv.Score != 100 {
		t.Fatalf("unexpected review: status=%s rec=%s score=%d", rv.Status, rv.Recommendation, rv.Score)
	}
	if rv.Summary != "clean change" || len(rv.Findings) != 0 {
		t.Fatalf("unexpected findings/summary: %+v", rv)
	}
	if rv.InputTokens != 100 || rv.OutputTokens != 50 {
		t.Fatalf("tokens not carried through: %d/%d", rv.InputTokens, rv.OutputTokens)
	}
}

func TestRunWithBlockingFinding(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{content: `{"summary":"has issues","findings":[
		{"dimension":"risk","severity":"high","file":"auth.go","line":42,"issue":"hardcoded secret","why":"leak","fix":"use env","blocking":true}
	]}`}

	rv, err := e.Run(context.Background(), fc, "model-x", nil, sampleInput(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rv.Recommendation != review.RequestChanges {
		t.Fatalf("recommendation = %s, want request_changes", rv.Recommendation)
	}
	if len(rv.Findings) != 1 {
		t.Fatalf("len findings = %d, want 1", len(rv.Findings))
	}
	f := rv.Findings[0]
	if f.Dimension != review.Risk || f.File != "auth.go" || f.Line != 42 || !f.Blocking {
		t.Fatalf("finding not mapped correctly: %+v", f)
	}
}

func TestRunParsesFencedJSON(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{content: "Here is the review:\n```json\n{\"summary\":\"ok\",\"findings\":[]}\n```\n"}

	rv, err := e.Run(context.Background(), fc, "m", nil, sampleInput(), nil)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if rv.Summary != "ok" {
		t.Fatalf("fenced JSON not parsed, summary=%q", rv.Summary)
	}
}

func TestRunPromptContainsSkillsAndDiff(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{content: `{"summary":"x","findings":[]}`}

	if _, err := e.Run(context.Background(), fc, "m", nil, sampleInput(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(fc.gotReq.Messages) != 2 {
		t.Fatalf("expected system+user messages, got %d", len(fc.gotReq.Messages))
	}
	system := fc.gotReq.Messages[0].Content
	user := fc.gotReq.Messages[1].Content
	if !strings.Contains(system, "R1") || !strings.Contains(system, "JSON") {
		t.Fatal("system prompt missing rules or JSON contract")
	}
	if !strings.Contains(user, "@@ -1 +1 @@") {
		t.Fatal("user prompt missing the diff")
	}
	if fc.gotReq.Temperature != nil {
		t.Fatalf("temperature = %v, want nil (not sent by default)", fc.gotReq.Temperature)
	}
}

func TestRunForwardsTemperature(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{content: `{"summary":"x","findings":[]}`}
	temp := 0.2

	if _, err := e.Run(context.Background(), fc, "m", &temp, sampleInput(), nil); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if fc.gotReq.Temperature == nil || *fc.gotReq.Temperature != 0.2 {
		t.Fatalf("temperature not forwarded: %v", fc.gotReq.Temperature)
	}
}

func TestRunEmptyDiffIsError(t *testing.T) {
	e := newEngine(t)
	in := sampleInput()
	in.Diff = ""
	if _, err := e.Run(context.Background(), &fakeClient{}, "m", nil, in, nil); err == nil {
		t.Fatal("expected error for empty diff")
	}
}

func TestRunPropagatesClientError(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{err: errors.New("rate limited")}
	if _, err := e.Run(context.Background(), fc, "m", nil, sampleInput(), nil); err == nil {
		t.Fatal("expected client error to propagate")
	}
}

func TestRunMalformedOutputIsError(t *testing.T) {
	e := newEngine(t)
	fc := &fakeClient{content: "I could not produce JSON, sorry."}
	if _, err := e.Run(context.Background(), fc, "m", nil, sampleInput(), nil); err == nil {
		t.Fatal("expected parse error for non-JSON output")
	}
}
