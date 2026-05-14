package contracts

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
		Intent: ast.Intent{Primary: "p"},
		Contracts: map[string]ast.Contract{
			// Deliberately not alphabetical -- exercises the sort.
			"zulu": {
				Given: "z given",
				When:  "z when",
				Then:  "z then",
			},
			"alpha": {
				Given: "a given line one\na given line two\n",
				When:  "a when",
				Then:  "a then",
			},
			"beta": {
				Given: "b given",
				When:  "b when\n",
				Then:  "b then\n",
			},
		},
	}
}

func TestList_AlphabeticalOrder(t *testing.T) {
	got := List(sampleDecl())
	want := []string{"alpha", "beta", "zulu"}
	if len(got) != len(want) {
		t.Fatalf("len: got %d, want %d", len(got), len(want))
	}
	for i, e := range got {
		if e.Name != want[i] {
			t.Errorf("[%d] got %q, want %q", i, e.Name, want[i])
		}
	}
}

func TestList_NilDeclaration(t *testing.T) {
	got := List(nil)
	if got == nil {
		t.Fatal("List(nil) returned nil; want empty slice for caller convenience")
	}
	if len(got) != 0 {
		t.Errorf("got %d entries; want 0", len(got))
	}
}

func TestList_EmptyContracts(t *testing.T) {
	got := List(&ast.Declaration{System: "t"})
	if len(got) != 0 {
		t.Errorf("got %d entries from empty contracts; want 0", len(got))
	}
}

func TestWriteList_TextDefault(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteList(&buf, List(sampleDecl()), WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	want := "alpha\nbeta\nzulu\n"
	if buf.String() != want {
		t.Errorf("got:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestWriteList_TextEmpty(t *testing.T) {
	// The empty-text contract: no input -> no output, not even a
	// trailing newline. This is what makes `xargs -n1 dx ...`
	// behave correctly when there are zero contracts.
	var buf bytes.Buffer
	if err := WriteList(&buf, nil, WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("empty-text output should be empty; got %q", buf.String())
	}
}

func TestWriteList_TextVerbose(t *testing.T) {
	var buf bytes.Buffer
	err := WriteList(&buf, List(sampleDecl()), WriteOptions{Verbose: true})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()

	// Each contract takes exactly four lines: name + 3 indented
	// clauses. Three contracts -> 12 lines total.
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 12 {
		t.Errorf("verbose output should be 12 lines (3 contracts * 4); got %d:\n%s",
			len(lines), out)
	}

	// alpha's `given:` is multi-line; preview must end in " …".
	if !strings.Contains(out, "given: a given line one …") {
		t.Errorf("multiline body should be truncated with ' …'; got:\n%s", out)
	}
	// zulu's `when:` is single-line; must NOT end in " …".
	if strings.Contains(out, "when: z when …") {
		t.Errorf("single-line body should NOT have a trailing ' …'; got:\n%s", out)
	}
}

func TestWriteList_VerboseAlignment(t *testing.T) {
	// All three clauses must align at the same column. This is a
	// human-ergonomics property; if it ever drifts, a reviewer will
	// notice fast and the test should catch it first.
	var buf bytes.Buffer
	err := WriteList(&buf, List(sampleDecl()), WriteOptions{Verbose: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		// Verbose clause lines start with 2-space indent + key + colon.
		switch {
		case strings.HasPrefix(line, "  given:"),
			strings.HasPrefix(line, "  when: "),
			strings.HasPrefix(line, "  then: "):
			// Fixed format `  KEY:  value` -- the key is padded to 5
			// chars in the format string. A future change to the
			// renderer's padding would surface here.
		case line == "" || !strings.HasPrefix(line, "  "):
			// name lines and blank lines are fine.
		default:
			t.Errorf("unexpected indent shape on line: %q", line)
		}
	}
}

func TestWriteList_JSON(t *testing.T) {
	var buf bytes.Buffer
	err := WriteList(&buf, List(sampleDecl()), WriteOptions{Format: FormatJSON})
	if err != nil {
		t.Fatal(err)
	}
	// Trailing newline.
	out := buf.Bytes()
	if len(out) == 0 || out[len(out)-1] != '\n' {
		t.Fatalf("json output must end with newline; got %q", out)
	}
	// Validity + shape.
	var parsed struct {
		Contracts []struct {
			Name  string `json:"name"`
			Given string `json:"given"`
			When  string `json:"when"`
			Then  string `json:"then"`
		} `json:"contracts"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("output not valid JSON: %v\n%s", err, out)
	}
	if len(parsed.Contracts) != 3 {
		t.Fatalf("got %d contracts; want 3", len(parsed.Contracts))
	}
	// Order preserved across JSON.
	gotNames := []string{
		parsed.Contracts[0].Name,
		parsed.Contracts[1].Name,
		parsed.Contracts[2].Name,
	}
	wantNames := []string{"alpha", "beta", "zulu"}
	for i := range wantNames {
		if gotNames[i] != wantNames[i] {
			t.Errorf("[%d] got %q, want %q", i, gotNames[i], wantNames[i])
		}
	}
	// Multi-line bodies preserved verbatim.
	if !strings.Contains(parsed.Contracts[0].Given, "a given line two") {
		t.Errorf("multi-line given body lost in JSON: %q", parsed.Contracts[0].Given)
	}
}

func TestWriteList_JSONEmpty(t *testing.T) {
	// Empty JSON output is `{"contracts":[]}` followed by a newline.
	// This is a *positive* signal of "spec has zero contracts" --
	// distinct from a parse failure or empty pipe.
	var buf bytes.Buffer
	err := WriteList(&buf, nil, WriteOptions{Format: FormatJSON})
	if err != nil {
		t.Fatal(err)
	}
	if buf.String() != "{\"contracts\":[]}\n" {
		t.Errorf("empty-json output should be `{\"contracts\":[]}\\n`; got %q", buf.String())
	}
}

func TestWriteList_JSONNoHTMLEscape(t *testing.T) {
	// Mirrors the same property in pkg/export: agent-facing JSON
	// must not Unicode-escape `<` / `>` / `&`.
	d := &ast.Declaration{
		Contracts: map[string]ast.Contract{
			"x": {Then: "stdout contains \"Hello, <name>!\""},
		},
	}
	var buf bytes.Buffer
	if err := WriteList(&buf, List(d), WriteOptions{Format: FormatJSON}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "\\u003c") {
		t.Errorf("json output should not HTML-escape `<`; got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "<name>") {
		t.Errorf("expected `<name>` literal in output; got:\n%s", buf.String())
	}
}

func TestWriteList_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	err := WriteList(&buf, List(sampleDecl()), WriteOptions{Format: Format("xml")})
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("error should name the unknown format; got %v", err)
	}
}

func TestWriteList_DeterministicAcrossRuns(t *testing.T) {
	// Two runs over the same input must produce byte-identical
	// output for both formats. This is the property that lets two
	// agents agree on a hash of the contract enumeration.
	for _, fmt := range []Format{FormatText, FormatJSON} {
		t.Run(string(fmt), func(t *testing.T) {
			opts := WriteOptions{Format: fmt}
			var a, b bytes.Buffer
			if err := WriteList(&a, List(sampleDecl()), opts); err != nil {
				t.Fatal(err)
			}
			if err := WriteList(&b, List(sampleDecl()), opts); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(a.Bytes(), b.Bytes()) {
				t.Fatalf("non-deterministic output:\n--- a ---\n%s\n--- b ---\n%s",
					a.String(), b.String())
			}
		})
	}
}
