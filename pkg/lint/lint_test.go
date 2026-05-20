package lint

import (
	"strings"
	"testing"
)

// minimalValid is the smallest declaration body that passes every
// lint pass. It is a v0.2.0 CommonMark declaration with the three
// REQUIRED blocks present and a non-empty `**Primary:**`.
const minimalValid = `# t

## Intent

**Primary:** A test declaration.

## Invariants

## Assumptions
`

func TestLint_MinimalValid_OK(t *testing.T) {
	res := Lint("t.md", []byte(minimalValid))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	if res.Declaration == nil {
		t.Fatal("expected a populated Declaration on success")
	}
	if res.Declaration.System != "t" {
		t.Errorf("System = %q, want %q", res.Declaration.System, "t")
	}
	if res.Declaration.Intent.Primary != "A test declaration." {
		t.Errorf("Intent.Primary = %q, want %q",
			res.Declaration.Intent.Primary, "A test declaration.")
	}
}

func TestLint_EmptyFile(t *testing.T) {
	res := Lint("t.md", []byte(""))
	if res.OK() {
		t.Fatal("expected empty-file issue")
	}
	if !containsMessage(res.Issues, "empty file") {
		t.Errorf("missing empty-file diagnostic; got: %v", res.Issues)
	}
}

func TestLint_FlagsMissingRequiredBlocks(t *testing.T) {
	// Only `# system` and `## Intent` present; missing Invariants
	// and Assumptions blocks.
	src := `# t

## Intent

**Primary:** A test declaration.
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected required-block issues")
	}
	want := []string{
		"missing required `## Invariants` block",
		"missing required `## Assumptions` block",
	}
	for _, w := range want {
		if !containsMessage(res.Issues, w) {
			t.Errorf("missing %q in: %v", w, res.Issues)
		}
	}
}

func TestLint_FlagsMissingSystemAndPrimary(t *testing.T) {
	src := `# 

## Intent

**Secondary:**

- missing primary

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected system+primary issues")
	}
	want := []string{
		"`#` system heading body is empty",
		"missing required `#` system heading",
		"missing required `**Primary:**` sub-field",
	}
	for _, w := range want {
		if !containsMessage(res.Issues, w) {
			t.Errorf("missing %q in: %v", w, res.Issues)
		}
	}
}

func TestLint_RejectsUnknownBlock(t *testing.T) {
	src := `# t

## Intent

**Primary:** body

## Invariants

## Assumptions

## NotAKnownBlock

body
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected unknown-block issue")
	}
	if !containsMessage(res.Issues, "NotAKnownBlock") {
		t.Errorf("missing unknown-block diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsDuplicateBlock(t *testing.T) {
	src := `# t

## Intent

**Primary:** body

## Invariants

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected duplicate-block issue")
	}
	if !containsMessage(res.Issues, "duplicate `## Invariants`") {
		t.Errorf("missing duplicate-block diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsOrphanKey(t *testing.T) {
	src := `# t

### orphan_key

body without an enclosing ## block

## Intent

**Primary:** body

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected orphan-key issue")
	}
	if !containsMessage(res.Issues, "outside any recognized `##` block") {
		t.Errorf("missing orphan-key diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsIntentKey(t *testing.T) {
	src := `# t

## Intent

### primary

body

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected intent-key issue")
	}
	if !containsMessage(res.Issues, "`## Intent` does not use `###` keys") {
		t.Errorf("missing intent-key diagnostic; got: %v", res.Issues)
	}
}

func TestLint_RejectsDuplicateInvariant(t *testing.T) {
	src := `# t

## Intent

**Primary:** body

## Invariants

### iface_foo

first body

### iface_foo

second body

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected duplicate-invariant issue")
	}
	if !containsMessage(res.Issues, "duplicate invariant") {
		t.Errorf("missing duplicate-invariant diagnostic; got: %v", res.Issues)
	}
}

func TestLint_AcceptsCompleteContract(t *testing.T) {
	src := `# t

## Intent

**Primary:** body

## Invariants

## Assumptions

## Contracts

### simple_contract

**Given:** a precondition

**When:** an action

**Then:** an outcome
`
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	c, ok := res.Declaration.Contracts["simple_contract"]
	if !ok {
		t.Fatal("contract not present in AST")
	}
	if c.Given != "a precondition" || c.When != "an action" || c.Then != "an outcome" {
		t.Errorf("contract = %+v; want Given/When/Then triple", c)
	}
}

func TestLint_FlagsMissingContractSubfield(t *testing.T) {
	src := `# t

## Intent

**Primary:** body

## Invariants

## Assumptions

## Contracts

### incomplete

**Given:** a precondition

**When:** an action
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected missing-Then issue")
	}
	if !containsMessage(res.Issues, "missing `**Then:**`") {
		t.Errorf("missing Then diagnostic; got: %v", res.Issues)
	}
}

func TestLint_AcceptsExplicitlyEmptyAssumptions(t *testing.T) {
	// SPEC §4.3.4: an `## Assumptions` heading with zero `###`
	// children is the explicitly-empty form.
	src := `# t

## Intent

**Primary:** body

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	if len(res.Declaration.Assumptions) != 0 {
		t.Errorf("Assumptions = %v, want empty", res.Declaration.Assumptions)
	}
	if !res.Declaration.BlocksPresent["Assumptions"] {
		t.Errorf("BlocksPresent[Assumptions] should be true (heading present)")
	}
}

func TestLint_IntentSecondaryListParses(t *testing.T) {
	src := `# t

## Intent

**Primary:** body

**Secondary:**

- first
- second
- third

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	got := res.Declaration.Intent.Secondary
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("Secondary length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("Secondary[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestLint_HeadingsInsideFencedCodeAreIgnored(t *testing.T) {
	// A `#` inside a fenced code block in a leaf must NOT be
	// treated as a structural heading.
	src := "# t\n" +
		"\n" +
		"## Intent\n" +
		"\n" +
		"**Primary:** body\n" +
		"\n" +
		"## Invariants\n" +
		"\n" +
		"### iface_with_code\n" +
		"\n" +
		"Here is an example output:\n" +
		"\n" +
		"```\n" +
		"# this is not a heading\n" +
		"## neither is this\n" +
		"```\n" +
		"\n" +
		"## Assumptions\n"
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	if _, ok := res.Declaration.Invariants["iface_with_code"]; !ok {
		t.Errorf("invariant `iface_with_code` not captured; got: %v",
			res.Declaration.Invariants)
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
