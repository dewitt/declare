package diff

import (
	"strings"
	"testing"

	"github.com/dewitt/declare/pkg/ast"
)

func TestDiff_NoChange(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      ast.Intent{Primary: "do the thing"},
		Invariants:  map[string]string{"iface_a": "a"},
		Assumptions: map[string]string{},
	}
	if got := Diff(d, d); len(got) != 0 {
		t.Fatalf("expected no changes, got: %v", got)
	}
}

func TestDiff_AddRemoveMutate(t *testing.T) {
	old := &ast.Declaration{
		Invariants: map[string]string{
			"iface_a": "value-a-old",
			"iface_b": "value-b",
		},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]string{
			"iface_a": "value-a-new",
			"iface_c": "value-c",
		},
	}
	got := Diff(old, new_)

	want := []string{
		"[MUTATED] invariants.iface_a",
		"[REMOVED] invariants.iface_b",
		"[ADDED] invariants.iface_c",
	}
	assertChanges(t, got, want)
}

func TestDiff_PromotionFromAssumptionsToInvariants(t *testing.T) {
	// The canonical architect workflow: an assumption becomes an
	// invariant via cut-and-paste. The diff must recognize this as a
	// PROMOTED operation, not as REMOVED+ADDED.
	old := &ast.Declaration{
		Assumptions: map[string]string{
			"cli.default_format": "default to text output",
		},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]string{
			"iface_default_format": "default to text output",
		},
	}
	got := Diff(old, new_)
	want := []string{
		"[PROMOTED] assumptions.cli.default_format -> invariants.iface_default_format",
	}
	assertChanges(t, got, want)
}

func TestDiff_DemotionFromInvariantsToUnconstrained(t *testing.T) {
	old := &ast.Declaration{
		Invariants: map[string]string{"perf_x": "fast enough"},
	}
	new_ := &ast.Declaration{
		Unconstrained: map[string]string{"perf": "fast enough"},
	}
	got := Diff(old, new_)
	want := []string{
		"[DEMOTED] invariants.perf_x -> unconstrained.perf",
	}
	assertChanges(t, got, want)
}

func TestDiff_RenameWithinSameBlock(t *testing.T) {
	old := &ast.Declaration{
		Invariants: map[string]string{"iface_legacy_name": "body"},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]string{"iface_modern_name": "body"},
	}
	got := Diff(old, new_)
	want := []string{
		"[RENAMED] invariants.iface_legacy_name -> invariants.iface_modern_name",
	}
	assertChanges(t, got, want)
}

func TestDiff_DeterministicOrdering(t *testing.T) {
	// Block order should follow SPEC §2 (system, intent, invariants,
	// assumptions, contracts, unconstrained).
	old := &ast.Declaration{}
	new_ := &ast.Declaration{
		System:        "t",
		Intent:        ast.Intent{Primary: "p"},
		Invariants:    map[string]string{"iface_a": "a"},
		Assumptions:   map[string]string{"x": "y"},
		Contracts:     map[string]ast.Contract{"c": {Given: "g", When: "w", Then: "t"}},
		Unconstrained: map[string]string{"lang": "any"},
	}
	got := Diff(old, new_)
	prevBlock := -1
	for _, c := range got {
		b := blockOrder(c.Path)
		if b < prevBlock {
			t.Fatalf("non-monotonic block order: %v", got)
		}
		prevBlock = b
	}
}

func TestDiff_IntentPrimaryMutation(t *testing.T) {
	old := &ast.Declaration{Intent: ast.Intent{Primary: "old purpose"}}
	new_ := &ast.Declaration{Intent: ast.Intent{Primary: "new purpose"}}
	got := Diff(old, new_)
	assertChanges(t, got, []string{"[MUTATED] intent.primary"})
}

func TestDiff_NilSafe(t *testing.T) {
	// Passing nil on either side should not panic; nil represents an
	// empty declaration.
	if got := Diff(nil, nil); len(got) != 0 {
		t.Fatalf("expected no changes, got: %v", got)
	}
	new_ := &ast.Declaration{System: "t"}
	got := Diff(nil, new_)
	assertChanges(t, got, []string{"[ADDED] system"})
}

// assertChanges compares the String() form of each Change against want.
// It tolerates extra whitespace but requires exact substring matches.
func assertChanges(t *testing.T, got []Change, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("change count: got %d, want %d\n got: %v\nwant: %v",
			len(got), len(want), formatChanges(got), want)
	}
	for i, w := range want {
		if g := got[i].String(); !strings.Contains(g, w) {
			t.Errorf("change[%d]:\n got: %s\nwant substring: %s", i, g, w)
		}
	}
}

func formatChanges(cs []Change) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.String()
	}
	return out
}
