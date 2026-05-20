package lint

import (
	"strings"
	"testing"
)

// minimalValid is the smallest declaration body that passes every
// lint pass. It is a v0.2.0 CommonMark declaration with the three
// REQUIRED blocks present and a one-sentence intent.
const minimalValid = `# t

## Intent

A test declaration.

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
	if len(res.Declaration.Intent) != 1 {
		t.Fatalf("Intent length = %d, want 1; got %v",
			len(res.Declaration.Intent), res.Declaration.Intent)
	}
	if res.Declaration.Intent[0] != "A test declaration." {
		t.Errorf("Intent[0] = %q, want %q",
			res.Declaration.Intent[0], "A test declaration.")
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

A test declaration.
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

func TestLint_FlagsMissingSystemAndIntent(t *testing.T) {
	src := `# 

## Intent

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected system+intent issues")
	}
	want := []string{
		"`#` system heading body is empty",
		"missing required `#` system heading",
		"missing intent body under `## Intent`",
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

body

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

body

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

### Orphan key

body without an enclosing ## block

## Intent

body

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

### Primary

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

func TestLint_RejectsSlugCollision(t *testing.T) {
	// Two different heading bodies that reduce to the same slug.
	src := `# t

## Intent

body

## Invariants

### Greets a named user

first body

### Greets a Named User

second body

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected slug-collision issue")
	}
	if !containsMessage(res.Issues, "collides with") {
		t.Errorf("missing slug-collision diagnostic; got: %v", res.Issues)
	}
}

func TestLint_AcceptsCompleteContract(t *testing.T) {
	src := `# t

## Intent

body

## Invariants

## Assumptions

## Contracts

### Simple contract

**Given** a precondition

**When** an action

**Then** an outcome
`
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	c, ok := res.Declaration.Contracts["simple_contract"]
	if !ok {
		t.Fatalf("contract not present in AST; got keys: %v", contractKeys(res))
	}
	if c.Heading != "Simple contract" {
		t.Errorf("Heading = %q, want %q", c.Heading, "Simple contract")
	}
	if c.Given != "a precondition" || c.When != "an action" || c.Then != "an outcome" {
		t.Errorf("contract = %+v; want Given/When/Then triple", c)
	}
}

func TestLint_FlagsMissingContractSubfield(t *testing.T) {
	src := `# t

## Intent

body

## Invariants

## Assumptions

## Contracts

### Incomplete contract

**Given** a precondition

**When** an action
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected missing-Then issue")
	}
	if !containsMessage(res.Issues, "missing `**Then**`") {
		t.Errorf("missing Then diagnostic; got: %v", res.Issues)
	}
}

func TestLint_AcceptsExplicitlyEmptyAssumptions(t *testing.T) {
	// SPEC §4.3.4: an `## Assumptions` heading with zero `###`
	// children is the explicitly-empty form.
	src := `# t

## Intent

body

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

func TestLint_IntentAsListParses(t *testing.T) {
	src := `# t

## Intent

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
	got := res.Declaration.Intent
	want := []string{"first", "second", "third"}
	if len(got) != len(want) {
		t.Fatalf("Intent length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("Intent[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestLint_RejectsMixedIntent(t *testing.T) {
	// A paragraph followed by a list is mutually exclusive per
	// SPEC §4.3.2.
	src := `# t

## Intent

a paragraph

- and a list item

## Invariants

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if res.OK() {
		t.Fatal("expected mixed-intent issue")
	}
	if !containsMessage(res.Issues, "mixes a paragraph and a list") {
		t.Errorf("missing mixed-intent diagnostic; got: %v", res.Issues)
	}
}

func TestLint_HeadingBodyWithHumanProseDerivedSlug(t *testing.T) {
	// Human-prose heading bodies must be captured verbatim and
	// reduced to a stable slug for the AST key.
	src := `# t

## Intent

body

## Invariants

### Single line on stdout

Writes a single UTF-8 line to stdout terminated by ` + "`" + `\n` + "`" + `.

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	const wantSlug = "single_line_on_stdout"
	entry, ok := res.Declaration.Invariants[wantSlug]
	if !ok {
		t.Fatalf("invariant slug %q not in AST; got keys %v",
			wantSlug, invariantKeys(res))
	}
	if entry.Heading != "Single line on stdout" {
		t.Errorf("Heading = %q, want %q", entry.Heading, "Single line on stdout")
	}
}

func TestLint_InlineBoldInLeafIsNotMistakenForLabel(t *testing.T) {
	// A paragraph that *starts* with inline bold (no trailing
	// space after the closing **) must NOT be parsed as a
	// sub-field label. Otherwise leaf prose like '**important**:'
	// or '**foo**.' would be silently corrupted.
	//
	// Also, a paragraph that starts with **Note** (recognized
	// label shape but unrecognized vocabulary) must be left as
	// leaf content, preserving the bold formatting verbatim.
	src := `# t

## Intent

body

## Invariants

### Important rule

**Note** that this paragraph begins with a non-vocabulary bold
keyword. The toolchain must leave the bold intact and treat the
entire paragraph as ordinary leaf prose.

**important**: a paragraph that uses inline bold followed by a
colon-space pattern also stays as prose.

## Assumptions
`
	res := Lint("t.md", []byte(src))
	if !res.OK() {
		t.Fatalf("expected zero issues, got: %v", res.Issues)
	}
	entry, ok := res.Declaration.Invariants["important_rule"]
	if !ok {
		t.Fatalf("invariant not present in AST; got keys: %v", invariantKeys(res))
	}
	if !strings.Contains(entry.Body, "**Note** that this paragraph") {
		t.Errorf("'**Note**' bold should be preserved in leaf prose; body:\n%s", entry.Body)
	}
	if !strings.Contains(entry.Body, "**important**: a paragraph") {
		t.Errorf("inline bold without trailing space should be preserved; body:\n%s", entry.Body)
	}
}

func TestLint_HeadingsInsideFencedCodeAreIgnored(t *testing.T) {
	// A `#` inside a fenced code block in a leaf must NOT be
	// treated as a structural heading.
	src := "# t\n" +
		"\n" +
		"## Intent\n" +
		"\n" +
		"body\n" +
		"\n" +
		"## Invariants\n" +
		"\n" +
		"### Interface with code\n" +
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
	if _, ok := res.Declaration.Invariants["interface_with_code"]; !ok {
		t.Errorf("invariant `interface_with_code` not captured; got: %v",
			invariantKeys(res))
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

func invariantKeys(res *Result) []string {
	keys := make([]string, 0, len(res.Declaration.Invariants))
	for k := range res.Declaration.Invariants {
		keys = append(keys, k)
	}
	return keys
}

func contractKeys(res *Result) []string {
	keys := make([]string, 0, len(res.Declaration.Contracts))
	for k := range res.Declaration.Contracts {
		keys = append(keys, k)
	}
	return keys
}
