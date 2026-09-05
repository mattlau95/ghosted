package resolver

import "testing"

func TestBackfillKnownValues(t *testing.T) {
	companies := []Company{
		{Name: "AutoHigh"}, // should backfill
		{Name: "AlreadyKnown", KnownATS: "ashby", KnownToken: "x"}, // should not change
		{Name: "LowConfidence"}, // should not backfill (low)
		{Name: "Missed"},        // should not backfill (manual/cut)
	}
	results := []Result{
		{Company: "AutoHigh", ATS: "greenhouse", BoardToken: "autohigh", Confidence: "high"},
		{Company: "AlreadyKnown", ATS: "ashby", BoardToken: "x", Confidence: "high", Notes: "known (skip probing)"},
		{Company: "LowConfidence", ATS: "ashby", BoardToken: "lowconf", Confidence: "low"},
		{Company: "Missed", ATS: "manual", Confidence: "none"},
	}

	updated, changed := BackfillKnownValues(companies, results)
	if changed != 1 {
		t.Fatalf("changed = %d, want 1", changed)
	}
	if updated[0].KnownATS != "greenhouse" || updated[0].KnownToken != "autohigh" {
		t.Errorf("AutoHigh not backfilled: %+v", updated[0])
	}
	if updated[1].KnownToken != "x" {
		t.Errorf("AlreadyKnown was overwritten: %+v", updated[1])
	}
	if updated[2].KnownATS != "" {
		t.Errorf("LowConfidence should not be backfilled: %+v", updated[2])
	}
	if updated[3].KnownATS != "" {
		t.Errorf("Missed should not be backfilled: %+v", updated[3])
	}
}
