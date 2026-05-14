package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dewitt/dx/pkg/ast"
)

func sampleDecl() *ast.Declaration {
	return &ast.Declaration{
		System: "t",
		Intent: ast.Intent{
			Primary:   "do the thing",
			Secondary: []string{"keep it small", "be friendly"},
		},
		Invariants: map[string]string{
			"perf_x":  "p",
			"iface_a": "a",
		},
		Assumptions: map[string]string{},
		Contracts: map[string]ast.Contract{
			"c1": {Given: "g", When: "w", Then: "t"},
		},
		Unconstrained: map[string]string{
			"language": "any",
		},
	}
}

func TestWrite_DefaultIsYAML(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sampleDecl(), ""); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "system:") {
		t.Errorf("default format should produce YAML; got:\n%s", out)
	}
}

func TestWrite_YAMLStripsComments(t *testing.T) {
	// canonical.Marshal handles comment stripping; export only sets
	// the option. Rather than reproduce the canonical-side test,
	// confirm the wiring: export with default options must NOT have
	// preserved any input comment, because export never receives a
	// SourceNode.
	var buf bytes.Buffer
	if err := Write(&buf, sampleDecl(), FormatYAML); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "#") {
		t.Errorf("yaml export must not contain comments; got:\n%s", buf.String())
	}
}

func TestWrite_JSONIsValid(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sampleDecl(), FormatJSON); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("json output not valid: %v\noutput: %s", err, buf.String())
	}
	if got["system"] != "t" {
		t.Errorf("system: got %v, want %q", got["system"], "t")
	}
}

func TestWrite_JSONHasTrailingNewline(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, sampleDecl(), FormatJSON); err != nil {
		t.Fatal(err)
	}
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Errorf("json output must end with newline; got %q", out)
	}
}

func TestWrite_JSONDoesNotEscapeAngleBrackets(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      ast.Intent{Primary: "Hello, <name>!"},
		Invariants:  map[string]string{},
		Assumptions: map[string]string{},
	}
	var buf bytes.Buffer
	if err := Write(&buf, d, FormatJSON); err != nil {
		t.Fatal(err)
	}
	// The Go default would emit \u003cname\u003e; we explicitly
	// disabled HTML escaping so the bracket appears literally.
	if strings.Contains(buf.String(), "\\u003c") {
		t.Errorf("json output should not HTML-escape `<`/`>`; got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "<name>") {
		t.Errorf("expected `<name>` literal in output; got:\n%s", buf.String())
	}
}

func TestWrite_JSONIsDeterministic(t *testing.T) {
	// Same input must produce byte-identical output across runs --
	// this is the property that lets two agents agree on hashes.
	var a, b bytes.Buffer
	if err := Write(&a, sampleDecl(), FormatJSON); err != nil {
		t.Fatal(err)
	}
	if err := Write(&b, sampleDecl(), FormatJSON); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatalf("json output not deterministic:\n--- a ---\n%s\n--- b ---\n%s",
			a.String(), b.String())
	}
}

func TestWrite_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, sampleDecl(), Format("xml"))
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("error should name the unknown format; got %v", err)
	}
}

func TestWrite_NilDeclaration(t *testing.T) {
	var buf bytes.Buffer
	if err := Write(&buf, nil, FormatYAML); err == nil {
		t.Fatal("expected error for nil declaration")
	}
}

func TestWrite_JSONOmitsEmptyOptionals(t *testing.T) {
	d := &ast.Declaration{
		System:      "t",
		Intent:      ast.Intent{Primary: "p"},
		Invariants:  map[string]string{},
		Assumptions: map[string]string{},
	}
	var buf bytes.Buffer
	if err := Write(&buf, d, FormatJSON); err != nil {
		t.Fatal(err)
	}
	s := buf.String()
	if strings.Contains(s, "\"contracts\"") {
		t.Errorf("empty contracts should be omitted; got:\n%s", s)
	}
	if strings.Contains(s, "\"unconstrained\"") {
		t.Errorf("empty unconstrained should be omitted; got:\n%s", s)
	}
	// invariants and assumptions are required and must appear as
	// `{}` (semantic distinction per SPEC §3).
	if !strings.Contains(s, "\"invariants\":{}") {
		t.Errorf("required invariants should appear as {}; got:\n%s", s)
	}
	if !strings.Contains(s, "\"assumptions\":{}") {
		t.Errorf("required assumptions should appear as {}; got:\n%s", s)
	}
}
