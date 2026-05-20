package canonical

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dewitt/dx/pkg/ast"
)

// minimalDecl is a fully-populated Declaration covering every block;
// every test that doesn't need a custom fixture builds from this.
func minimalDecl() *ast.Declaration {
	return &ast.Declaration{
		System: "t",
		Intent: []string{"second item", "first item"}, // deliberately unsorted
		Invariants: map[string]ast.Entry{
			"perf_x":  {Heading: "Perf x", Body: "perf body"},
			"iface_a": {Heading: "Iface a", Body: "iface body\nspans two lines"},
		},
		Assumptions: map[string]ast.Entry{
			"z_late":  {Heading: "Z late", Body: "late"},
			"a_early": {Heading: "A early", Body: "early"},
		},
		Contracts: map[string]ast.Contract{
			"second": {Heading: "Second", Given: "g2", When: "w2", Then: "t2"},
			"first":  {Heading: "First", Given: "g1", When: "w1", Then: "t1"},
		},
		Unconstrained: map[string]ast.Entry{
			"language": {Heading: "Language", Body: "any"},
		},
	}
}

func TestMarshal_BlockOrder(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// SPEC §4.5 canonical order: Intent, Invariants, Assumptions,
	// Contracts, Unconstrained (the `#` system heading is first).
	wantOrder := []string{
		"# t",
		"## Intent",
		"## Invariants",
		"## Assumptions",
		"## Contracts",
		"## Unconstrained",
	}
	pos := -1
	for _, k := range wantOrder {
		idx := bytes.Index(out, []byte(k))
		if idx < 0 {
			t.Fatalf("heading %q missing from output:\n%s", k, out)
		}
		if idx <= pos {
			t.Errorf("heading %q appears at offset %d, expected after %d:\n%s",
				k, idx, pos, out)
		}
		pos = idx
	}
}

func TestMarshal_AlphabetizesSlugs(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)

	// Within Invariants: iface_a's heading 'Iface a' should appear
	// before perf_x's heading 'Perf x' because the slugs sort
	// alphabetically.
	if strings.Index(s, "### Iface a") > strings.Index(s, "### Perf x") {
		t.Errorf("invariants not slug-sorted:\n%s", s)
	}
	// Within Assumptions: a_early before z_late.
	if strings.Index(s, "### A early") > strings.Index(s, "### Z late") {
		t.Errorf("assumptions not slug-sorted:\n%s", s)
	}
	// Within Contracts: first before second.
	firstIdx := strings.Index(s, "### First")
	secondIdx := strings.Index(s, "### Second")
	if firstIdx == -1 || secondIdx == -1 || firstIdx > secondIdx {
		t.Errorf("contracts not slug-sorted:\n%s", s)
	}
}

func TestMarshal_PreservesIntentListOrder(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// "second item" was authored before "first item"; we must NOT
	// alphabetize.
	s := string(out)
	if strings.Index(s, "- second item") > strings.Index(s, "- first item") {
		t.Errorf("intent order changed (must preserve authored order):\n%s", s)
	}
}

func TestMarshal_SingleIntentEmitsAsParagraph(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      []string{"a single intent"},
		Invariants:  map[string]ast.Entry{},
		Assumptions: map[string]ast.Entry{},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	// Single intent renders as paragraph, NOT as a list item.
	if !strings.Contains(s, "\na single intent\n") {
		t.Errorf("single intent should render as a paragraph; got:\n%s", s)
	}
	if strings.Contains(s, "- a single intent") {
		t.Errorf("single intent should NOT render as a list item; got:\n%s", s)
	}
}

func TestMarshal_ContractSubfieldOrder(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	g := strings.Index(s, "**Given:** g1")
	w := strings.Index(s, "**When:** w1")
	th := strings.Index(s, "**Then:** t1")
	if g < 0 || w < 0 || th < 0 {
		t.Fatalf("missing Given/When/Then for `first`:\n%s", s)
	}
	if !(g < w && w < th) {
		t.Errorf("contract sub-fields not in fixed Given/When/Then order:\n%s", s)
	}
}

func TestMarshal_MultiparagraphLeafPreserved(t *testing.T) {
	d := &ast.Declaration{
		System: "t",
		Intent: []string{"p"},
		Invariants: map[string]ast.Entry{
			"iface_complex": {
				Heading: "Iface complex",
				Body:    "first paragraph\n\nsecond paragraph",
			},
		},
		Assumptions: map[string]ast.Entry{},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "first paragraph\n\nsecond paragraph") {
		t.Errorf("multi-paragraph leaf body not preserved:\n%s", s)
	}
}

func TestMarshal_EmptyRequiredBlocksRenderAsHeadingOnly(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      []string{"p"},
		Invariants:  nil, // nil and empty map both produce heading-only
		Assumptions: map[string]ast.Entry{},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "## Invariants") {
		t.Errorf("missing `## Invariants` heading:\n%s", s)
	}
	if !strings.Contains(s, "## Assumptions") {
		t.Errorf("missing `## Assumptions` heading:\n%s", s)
	}
	if strings.Contains(s, "### ") {
		t.Errorf("empty blocks should have no `###` children:\n%s", s)
	}
}

func TestMarshal_OmitsOptionalEmptyBlocks(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      []string{"p"},
		Invariants:  map[string]ast.Entry{},
		Assumptions: map[string]ast.Entry{},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "## Contracts") {
		t.Errorf("empty optional `## Contracts` should be omitted:\n%s", s)
	}
	if strings.Contains(s, "## Unconstrained") {
		t.Errorf("empty optional `## Unconstrained` should be omitted:\n%s", s)
	}
}

func TestMarshal_NoTrailingWhitespace(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	for i, line := range bytes.Split(out, []byte("\n")) {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}
}

func TestMarshal_ExactlyOneTrailingNewline(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Errorf("output must end with a newline; got %q", out[max(0, len(out)-3):])
	}
	if len(out) >= 2 && out[len(out)-2] == '\n' {
		t.Errorf("output must NOT end with multiple newlines; got %q",
			out[max(0, len(out)-3):])
	}
}

func TestMarshal_Idempotent(t *testing.T) {
	once, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	twice, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(once, twice) {
		t.Fatalf("Marshal not deterministic:\n--- once ---\n%s\n--- twice ---\n%s",
			once, twice)
	}
}

func TestMarshal_NilDeclaration(t *testing.T) {
	if _, err := Marshal(nil, Options{}); err == nil {
		t.Fatal("expected error for nil declaration")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
