package memory

import "testing"

func TestSaveDedupesByID(t *testing.T) {
	items := []Item{
		{ID: "x", Summary: "first"},
		{ID: "y", Summary: "keep"},
		{ID: "x", Summary: "second"},
	}
	out := dedupe(items)
	if len(out) != 2 {
		t.Fatalf("expected 2 items, got %d", len(out))
	}
	if out[0].Summary != "second" {
		t.Fatalf("expected later duplicate to win, got %q", out[0].Summary)
	}
}
