package canonical

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/dewitt/dx/pkg/ast"
)

// minimalDecl is a fully-populated Declaration covering every block;
// every test that doesn't need a custom fixture builds from this.
func minimalDecl() *ast.Declaration {
	return &ast.Declaration{
		System: "t",
		Intent: ast.Intent{
			Primary:   "single line primary",
			Secondary: []string{"second", "first"}, // deliberately unsorted
		},
		Invariants: map[string]string{
			"perf_x":  "perf body\n",
			"iface_a": "iface body\nspans two lines\n",
		},
		Assumptions: map[string]string{
			"z_late":  "late\n",
			"a_early": "early\n",
		},
		Contracts: map[string]ast.Contract{
			"second": {Given: "g2", When: "w2", Then: "t2"},
			"first":  {Given: "g1", When: "w1", Then: "t1"},
		},
		Unconstrained: map[string]string{
			"language": "any\n",
		},
	}
}

func TestMarshal_TopLevelKeyOrder(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// SPEC §2 canonical order: system, intent, invariants,
	// assumptions, contracts, unconstrained.
	wantOrder := []string{
		"system:", "intent:", "invariants:", "assumptions:",
		"contracts:", "unconstrained:",
	}
	pos := -1
	for _, k := range wantOrder {
		idx := bytes.Index(out, []byte(k))
		if idx < 0 {
			t.Fatalf("key %q missing from output:\n%s", k, out)
		}
		if idx <= pos {
			t.Errorf("key %q appears at offset %d, expected after %d:\n%s",
				k, idx, pos, out)
		}
		pos = idx
	}
}

func TestMarshal_AlphabetizesMapKeys(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)

	// Within invariants: iface_a should appear before perf_x.
	if strings.Index(s, "iface_a:") > strings.Index(s, "perf_x:") {
		t.Errorf("invariants not alphabetized:\n%s", s)
	}
	// Within assumptions: a_early before z_late.
	if strings.Index(s, "a_early:") > strings.Index(s, "z_late:") {
		t.Errorf("assumptions not alphabetized:\n%s", s)
	}
	// Within contracts: first before second.
	firstIdx := strings.Index(s, "first:")
	secondIdx := strings.Index(s, "second:")
	if firstIdx == -1 || secondIdx == -1 || firstIdx > secondIdx {
		t.Errorf("contracts not alphabetized:\n%s", s)
	}
}

func TestMarshal_PreservesSecondaryListOrder(t *testing.T) {
	out, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	// "second" was authored before "first"; we must NOT alphabetize.
	s := string(out)
	if strings.Index(s, "- second") > strings.Index(s, "- first") {
		t.Errorf("intent.secondary order changed (must preserve authored order):\n%s", s)
	}
}

func TestMarshal_LiteralScalarForMultiline(t *testing.T) {
	d := &ast.Declaration{
		System: "t",
		Intent: ast.Intent{Primary: "line1\nline2\n"},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "primary: |") {
		t.Errorf("multiline string should use literal block scalar `|`:\n%s", out)
	}
}

func TestMarshal_PlainScalarForSingleLine(t *testing.T) {
	d := &ast.Declaration{
		System: "single-slug",
		Intent: ast.Intent{Primary: "one line"},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// "one line" should appear without `|` prefix.
	if strings.Contains(string(out), "primary: |") {
		t.Errorf("single-line string should NOT use `|`:\n%s", out)
	}
	if !strings.Contains(string(out), "primary: one line") {
		t.Errorf("expected plain scalar:\n%s", out)
	}
}

// A string like "one line\n" decodes from `primary: |\n  one line\n`,
// but its only newline is the trailing artifact. The canonicalizer
// must treat it as single-line so the round-trip emits as plain --
// otherwise every `|`-authored single-line string stays locked in
// block-scalar form forever. This is the regression that motivated
// the trim-trailing-newline rule in scalarString.
func TestMarshal_TrailingNewlineDoesNotForceLiteral(t *testing.T) {
	d := &ast.Declaration{
		System: "t",
		Intent: ast.Intent{Primary: "one line\n"},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "primary: |") {
		t.Errorf("single content line with trailing newline should NOT use `|`:\n%s", out)
	}
	if !strings.Contains(string(out), "primary: one line") {
		t.Errorf("expected plain scalar:\n%s", out)
	}
}

// Strings with internal newlines must still use the literal block
// scalar regardless of whether they end with a trailing newline.
func TestMarshal_InternalNewlineUsesLiteral(t *testing.T) {
	cases := map[string]string{
		"with trailing":    "line1\nline2\n",
		"without trailing": "line1\nline2",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			d := &ast.Declaration{
				System: "t",
				Intent: ast.Intent{Primary: body},
			}
			out, err := Marshal(d, Options{})
			if err != nil {
				t.Fatal(err)
			}
			// Either `|` (clip; preserves single trailing newline) or
			// `|-` (strip; no trailing newline) is acceptable. Both
			// are literal block forms; what we want to reject is the
			// plain-scalar emit that would smush the lines together.
			if !strings.Contains(string(out), "primary: |") {
				t.Errorf("multiline string must use literal `|` form:\n%s", out)
			}
		})
	}
}

func TestMarshal_EmptyMapsAsFlow(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      ast.Intent{Primary: "p"},
		Invariants:  nil, // nil and empty map both produce {}
		Assumptions: map[string]string{},
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "invariants: {}") {
		t.Errorf("empty invariants should render as `{}`:\n%s", s)
	}
	if !strings.Contains(s, "assumptions: {}") {
		t.Errorf("empty assumptions should render as `{}`:\n%s", s)
	}
}

func TestMarshal_OmitsOptionalEmptyBlocks(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      ast.Intent{Primary: "p"},
		Invariants:  map[string]string{},
		Assumptions: map[string]string{},
		// contracts and unconstrained omitted.
	}
	out, err := Marshal(d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "contracts:") {
		t.Errorf("empty optional `contracts:` should be omitted:\n%s", s)
	}
	if strings.Contains(s, "unconstrained:") {
		t.Errorf("empty optional `unconstrained:` should be omitted:\n%s", s)
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
		t.Errorf("output must end with a newline; got %q", out[len(out)-3:])
	}
	if len(out) >= 2 && out[len(out)-2] == '\n' {
		t.Errorf("output must NOT end with multiple newlines; got %q",
			out[len(out)-3:])
	}
}

func TestMarshal_StripCommentsOption(t *testing.T) {
	// Build a source node with a head comment on `system:`.
	src := &yaml.Node{Kind: yaml.DocumentNode}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	src.Content = []*yaml.Node{root}
	systemKey := &yaml.Node{
		Kind: yaml.ScalarNode, Value: "system",
		HeadComment: "# this is a head comment",
	}
	systemVal := &yaml.Node{Kind: yaml.ScalarNode, Value: "t"}
	root.Content = []*yaml.Node{systemKey, systemVal}

	d := &ast.Declaration{
		System:      "t",
		Intent:      ast.Intent{Primary: "p"},
		Invariants:  map[string]string{},
		Assumptions: map[string]string{},
	}

	preserved, err := Marshal(d, Options{StripComments: false, SourceNode: src})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(preserved, []byte("this is a head comment")) {
		t.Errorf("expected comment preserved when StripComments=false:\n%s", preserved)
	}

	stripped, err := Marshal(d, Options{StripComments: true, SourceNode: src})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(stripped, []byte("this is a head comment")) {
		t.Errorf("expected comment stripped when StripComments=true:\n%s", stripped)
	}
}

func TestMarshal_Idempotent(t *testing.T) {
	// Marshal twice and confirm byte equality.
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

func TestMarshal_Idempotent_AfterDecode(t *testing.T) {
	// The stronger property: Marshal(Decode(Marshal(d))) == Marshal(d).
	first, err := Marshal(minimalDecl(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	var d2 ast.Declaration
	if err := yaml.Unmarshal(first, &d2); err != nil {
		t.Fatalf("re-decode of canonical output failed: %v", err)
	}
	second, err := Marshal(&d2, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("canonical form not idempotent across decode:\n--- first ---\n%s\n--- second ---\n%s",
			first, second)
	}
}

func TestMarshal_NilDeclaration(t *testing.T) {
	if _, err := Marshal(nil, Options{}); err == nil {
		t.Fatal("expected error for nil declaration")
	}
}
