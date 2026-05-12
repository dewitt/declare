package lint

import (
	"strings"
	"testing"
)

// minimalValid is the smallest .dx body that should pass every lint pass.
const minimalValid = `system: t

intent:
  primary: |
    A test declaration.

invariants: {}

assumptions: {}
`

func TestLint_MinimalValid_OK(t *testing.T) {
	res := Lint("t.dx", []byte(minimalValid))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	if res.Declaration == nil {
		t.Fatal("expected a populated Declaration on success")
	}
	if res.Declaration.System != "t" {
		t.Errorf("System = %q, want %q", res.Declaration.System, "t")
	}
}

func TestLint_RejectsFoldedScalar(t *testing.T) {
	src := `system: t

intent:
  primary: >
    A folded multiline scalar should be rejected per SPEC §2.

invariants: {}
assumptions: {}
`
	res := Lint("t.dx", []byte(src))
	if res.OK() {
		t.Fatal("expected at least one folded-scalar issue")
	}
	if !containsMessage(res.Issues, "folded block scalar") {
		t.Errorf("missing folded-scalar diagnostic; got: %v", res.Issues)
	}
}

func TestLint_AcceptsLiteralScalar(t *testing.T) {
	src := `system: t

intent:
  primary: |
    A literal multiline scalar.
    Spans two lines.

invariants: {}
assumptions: {}
`
	res := Lint("t.dx", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues for literal scalar, got: %v", res.Issues)
	}
}

func TestLint_RejectsAnchorAndAlias(t *testing.T) {
	src := `system: t

intent: &intent_anchor
  primary: |
    Body.

invariants:
  iface_dup: *intent_anchor

assumptions: {}
`
	res := Lint("t.dx", []byte(src))
	if res.OK() {
		t.Fatal("expected anchor/alias issues")
	}
	gotAnchor := containsMessage(res.Issues, "anchor `&intent_anchor`")
	gotAlias := containsMessage(res.Issues, "alias node forbidden")
	if !gotAnchor {
		t.Errorf("missing anchor diagnostic; got: %v", res.Issues)
	}
	if !gotAlias {
		t.Errorf("missing alias diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsCustomTag(t *testing.T) {
	src := `system: t

intent:
  primary: !!binary aGVsbG8=

invariants: {}
assumptions: {}
`
	res := Lint("t.dx", []byte(src))
	if res.OK() {
		t.Fatal("expected custom-tag issue")
	}
	if !containsMessage(res.Issues, "explicit YAML tag") {
		t.Errorf("missing custom-tag diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsNestedInvariant(t *testing.T) {
	// SPEC §3: invariants leaves must be scalar strings, not maps.
	src := `system: t

intent:
  primary: |
    Body.

invariants:
  iface_complex:
    rule: do the thing
    rationale: because

assumptions: {}
`
	res := Lint("t.dx", []byte(src))
	if res.OK() {
		t.Fatal("expected nested-invariant issue")
	}
	if !containsMessage(res.Issues, "must be a scalar string") {
		t.Errorf("missing leaf-type diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsUnknownTopLevelField(t *testing.T) {
	src := `system: t

intent:
  primary: |
    Body.

invariants: {}
assumptions: {}

extra_field: nope
`
	res := Lint("t.dx", []byte(src))
	if res.OK() {
		t.Fatal("expected unknown-field issue")
	}
	if !containsMessage(res.Issues, "extra_field") {
		t.Errorf("missing unknown-field diagnostic; got: %v", res.Issues)
	}
}

func TestLint_FlagsMissingRequiredKeys(t *testing.T) {
	src := `system: ""
intent:
  secondary:
    - missing primary
`
	res := Lint("t.dx", []byte(src))
	if res.OK() {
		t.Fatal("expected required-key issues")
	}
	want := []string{
		"missing required key `system`",
		"missing required key `intent.primary`",
		"missing required key `invariants`",
		"missing required key `assumptions`",
	}
	for _, w := range want {
		if !containsMessage(res.Issues, w) {
			t.Errorf("missing %q in: %v", w, res.Issues)
		}
	}
}

func TestLint_EmptyFile(t *testing.T) {
	res := Lint("t.dx", []byte(""))
	if res.OK() {
		t.Fatal("expected empty-file issue")
	}
	if !containsMessage(res.Issues, "empty file") {
		t.Errorf("missing empty-file diagnostic; got: %v", res.Issues)
	}
}

func TestLint_AcceptsExplicitlyEmptyAssumptions(t *testing.T) {
	// SPEC §3 explicitly calls out the zero-assumption state: the key
	// must be present, but an empty map is valid.
	src := `system: t
intent:
  primary: |
    Body.
invariants: {}
assumptions: {}
`
	res := Lint("t.dx", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
}

// containsMessage reports whether any issue's message contains the
// given substring.
func containsMessage(issues []Issue, sub string) bool {
	for _, i := range issues {
		if strings.Contains(i.Message, sub) {
			return true
		}
	}
	return false
}
