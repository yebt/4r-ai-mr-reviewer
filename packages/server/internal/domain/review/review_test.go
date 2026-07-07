package review

import "testing"

func TestEvaluateApproveWhenClean(t *testing.T) {
	rec, score := Evaluate(nil)
	if rec != Approve || score != 100 {
		t.Fatalf("clean review: got %s/%d, want approve/100", rec, score)
	}
}

func TestEvaluateCommentOnLowFindings(t *testing.T) {
	rec, score := Evaluate([]Finding{{Severity: SeverityLow}, {Severity: SeverityMedium}})
	if rec != Comment {
		t.Fatalf("recommendation = %s, want comment", rec)
	}
	if score != 100-penaltyLow-penaltyMedium {
		t.Fatalf("score = %d, want %d", score, 100-penaltyLow-penaltyMedium)
	}
}

func TestEvaluateRequestChangesOnHigh(t *testing.T) {
	rec, _ := Evaluate([]Finding{{Severity: SeverityHigh}})
	if rec != RequestChanges {
		t.Fatalf("recommendation = %s, want request_changes", rec)
	}
}

func TestEvaluateRequestChangesOnBlocking(t *testing.T) {
	rec, _ := Evaluate([]Finding{{Severity: SeverityLow, Blocking: true}})
	if rec != RequestChanges {
		t.Fatalf("blocking low finding should request changes, got %s", rec)
	}
}

func TestEvaluateScoreFloorsAtZero(t *testing.T) {
	many := make([]Finding, 20)
	for i := range many {
		many[i] = Finding{Severity: SeverityHigh, Blocking: true}
	}
	_, score := Evaluate(many)
	if score != 0 {
		t.Fatalf("score = %d, want floored at 0", score)
	}
}

func TestDimensionValid(t *testing.T) {
	if !Risk.Valid() || Dimension("nope").Valid() {
		t.Fatal("Dimension.Valid is wrong")
	}
}
