package diff

import (
	"strings"
	"testing"

	"github.com/dewitt/dx/pkg/ast"
)

func TestDiff_NoChange(t *testing.T) {
	d := &ast.Declaration{
		System: "t",
		Intent: []string{"do the thing"},
		Invariants: map[string]ast.Entry{
			"iface_a": {Heading: "Iface a", Body: "a"},
		},
		Assumptions: map[string]ast.Entry{},
	}
	if got := Diff(d, d); len(got) != 0 {
		t.Fatalf("expected no changes, got: %v", got)
	}
}

func TestDiff_AddRemoveMutate(t *testing.T) {
	old := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"iface_a": {Heading: "Iface a", Body: "value-a-old"},
			"iface_b": {Heading: "Iface b", Body: "value-b"},
		},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"iface_a": {Heading: "Iface a", Body: "value-a-new"},
			"iface_c": {Heading: "Iface c", Body: "value-c"},
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
	// invariant. The diff must recognize this as a PROMOTED
	// operation, not as REMOVED+ADDED, when the body is unchanged.
	old := &ast.Declaration{
		Assumptions: map[string]ast.Entry{
			"cli_default_format": {Heading: "CLI default format", Body: "default to text output"},
		},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"interface_default_format": {Heading: "Interface: default format", Body: "default to text output"},
		},
	}
	got := Diff(old, new_)
	want := []string{
		"[PROMOTED] assumptions.cli_default_format -> invariants.interface_default_format",
	}
	assertChanges(t, got, want)
}

func TestDiff_DemotionFromInvariantsToUnconstrained(t *testing.T) {
	old := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"perf_x": {Heading: "Perf x", Body: "fast enough"},
		},
	}
	new_ := &ast.Declaration{
		Unconstrained: map[string]ast.Entry{
			"perf": {Heading: "Perf", Body: "fast enough"},
		},
	}
	got := Diff(old, new_)
	want := []string{
		"[DEMOTED] invariants.perf_x -> unconstrained.perf",
	}
	assertChanges(t, got, want)
}

func TestDiff_RenameWithinSameBlock(t *testing.T) {
	old := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"iface_legacy_name": {Heading: "Iface legacy name", Body: "body"},
		},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"iface_modern_name": {Heading: "Iface modern name", Body: "body"},
		},
	}
	got := Diff(old, new_)
	want := []string{
		"[RENAMED] invariants.iface_legacy_name -> invariants.iface_modern_name",
	}
	assertChanges(t, got, want)
}

func TestDiff_HeadingPolishIsSilent(t *testing.T) {
	// Polishing a heading without changing its slug or body should
	// produce no diff output: the entry's identity (slug) and
	// content (body) are both unchanged.
	old := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"single_line_on_stdout": {
				Heading: "single line on stdout",
				Body:    "body",
			},
		},
	}
	new_ := &ast.Declaration{
		Invariants: map[string]ast.Entry{
			"single_line_on_stdout": {
				Heading: "Single line on stdout",
				Body:    "body",
			},
		},
	}
	if got := Diff(old, new_); len(got) != 0 {
		t.Fatalf("expected no changes (heading polish only); got: %v", got)
	}
}

func TestDiff_DeterministicOrdering(t *testing.T) {
	// Block order should follow SPEC §4.5 (system, intent,
	// invariants, assumptions, contracts, unconstrained).
	old := &ast.Declaration{}
	new_ := &ast.Declaration{
		System: "t",
		Intent: []string{"p"},
		Invariants: map[string]ast.Entry{
			"iface_a": {Heading: "Iface a", Body: "a"},
		},
		Assumptions: map[string]ast.Entry{
			"x": {Heading: "X", Body: "y"},
		},
		Contracts: map[string]ast.Contract{
			"c": {Heading: "C", Given: "g", When: "w", Then: "t"},
		},
		Unconstrained: map[string]ast.Entry{
			"lang": {Heading: "Lang", Body: "any"},
		},
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

func TestDiff_IntentMutation(t *testing.T) {
	old := &ast.Declaration{Intent: []string{"old purpose"}}
	new_ := &ast.Declaration{Intent: []string{"new purpose"}}
	got := Diff(old, new_)
	assertChanges(t, got, []string{"[MUTATED] intent[0]"})
}

func TestDiff_NilSafe(t *testing.T) {
	// Passing nil on either side should not panic; nil represents
	// an empty declaration.
	if got := Diff(nil, nil); len(got) != 0 {
		t.Fatalf("expected no changes, got: %v", got)
	}
	new_ := &ast.Declaration{System: "t"}
	got := Diff(nil, new_)
	assertChanges(t, got, []string{"[ADDED] system"})
}

// assertChanges compares the String() form of each Change against
// want. It tolerates extra whitespace but requires exact substring
// matches.
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
