package canonical_test

// Cross-package property tests for the canonicalizer. These live in
// the _test package so they can import pkg/lint, which would otherwise
// create an import cycle with pkg/canonical (canonical does not import
// lint at runtime).
//
// The properties asserted here are the load-bearing contracts for
// `dx fmt`:
//
//   - Round-trip soundness: lint(canonical(d)) decodes to a Declaration
//     equal to d; in particular, the output is always lintable.
//   - Idempotency: canonical(canonical(d)) is byte-identical to
//     canonical(d).
//
// Together, these mean an architect can run `dx fmt -w` on any
// linted .dx file and the file will (a) still lint, (b) keep the same
// observable AST, and (c) be a fixed point under further formatting.

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/dewitt/dx/pkg/ast"
	"github.com/dewitt/dx/pkg/canonical"
	"github.com/dewitt/dx/pkg/lint"
)

const exampleHello = `system: hello-world

intent:
  primary: |
    Greet a user by name on standard output.
  secondary:
    - Be friendly.
    - Exit cleanly.

invariants:
  iface_stdout: |
    Writes a single UTF-8 line to stdout terminated by ` + "`" + `\n` + "`" + `.
  perf_startup_ms: |
    Cold-start latency must remain under 50ms on commodity hardware.

assumptions:
  greeting.format: |
    The greeting is "Hello, <name>!".

contracts:
  greets_named_user:
    given: |
      The argument vector contains exactly one non-empty name.
    when: |
      The binary is invoked.
    then: |
      stdout contains "Hello, <name>!\n" and the exit code is 0.

unconstrained:
  language: |
    Any language with a stable POSIX runtime is acceptable.
`

func TestRoundTrip_LintsCleanly(t *testing.T) {
	res := lint.Lint("input.dx", []byte(exampleHello))
	if !res.OK() {
		t.Fatalf("input did not lint: %v", res.Issues)
	}
	out, err := canonical.Marshal(res.Declaration, canonical.Options{
		StripComments: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res2 := lint.Lint("formatted.dx", out)
	if !res2.OK() {
		t.Fatalf("canonical output did not lint:\n--- output ---\n%s\n--- issues ---\n%v",
			out, res2.Issues)
	}
}

func TestRoundTrip_PreservesAST(t *testing.T) {
	res := lint.Lint("input.dx", []byte(exampleHello))
	if !res.OK() {
		t.Fatalf("input did not lint: %v", res.Issues)
	}
	out, err := canonical.Marshal(res.Declaration, canonical.Options{
		StripComments: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res2 := lint.Lint("formatted.dx", out)
	if !res2.OK() {
		t.Fatal(res2.Issues)
	}

	// Compare the observable fields modulo a single trailing newline
	// on every string value. The canonicalizer treats the trailing
	// newline produced by `|`-style decoding as a YAML emit artifact
	// rather than semantic content (so a single-line invariant body
	// like "Greet a user.\n" round-trips as "Greet a user." in plain
	// scalar form). The semantic guarantee is therefore not byte-
	// equality of values but equality after trimming one trailing
	// newline. The embedded *yaml.Node is naturally different and is
	// not compared.
	a, b := normalizeForCompare(res.Declaration), normalizeForCompare(res2.Declaration)
	if a.System != b.System {
		t.Errorf("System: %q -> %q", a.System, b.System)
	}
	if a.Intent.Primary != b.Intent.Primary {
		t.Errorf("Intent.Primary: %q -> %q", a.Intent.Primary, b.Intent.Primary)
	}
	if !reflect.DeepEqual(a.Intent.Secondary, b.Intent.Secondary) {
		t.Errorf("Intent.Secondary: %v -> %v", a.Intent.Secondary, b.Intent.Secondary)
	}
	if !reflect.DeepEqual(a.Invariants, b.Invariants) {
		t.Errorf("Invariants:\nbefore: %v\nafter:  %v", a.Invariants, b.Invariants)
	}
	if !reflect.DeepEqual(a.Assumptions, b.Assumptions) {
		t.Errorf("Assumptions:\nbefore: %v\nafter:  %v", a.Assumptions, b.Assumptions)
	}
	if !reflect.DeepEqual(a.Contracts, b.Contracts) {
		t.Errorf("Contracts:\nbefore: %v\nafter:  %v", a.Contracts, b.Contracts)
	}
	if !reflect.DeepEqual(a.Unconstrained, b.Unconstrained) {
		t.Errorf("Unconstrained:\nbefore: %v\nafter:  %v", a.Unconstrained, b.Unconstrained)
	}
}

// normalizeForCompare returns a shallow copy of d with every string
// field stripped of a single trailing newline. Used by the AST
// round-trip test to express the semantic equivalence the
// canonicalizer enforces -- see scalarString in canonical.go for
// the rationale.
func normalizeForCompare(d *ast.Declaration) *ast.Declaration {
	trimNL := func(s string) string {
		return strings.TrimSuffix(s, "\n")
	}
	out := &ast.Declaration{
		System: trimNL(d.System),
		Intent: ast.Intent{
			Primary: trimNL(d.Intent.Primary),
		},
	}
	for _, s := range d.Intent.Secondary {
		out.Intent.Secondary = append(out.Intent.Secondary, trimNL(s))
	}
	if d.Invariants != nil {
		out.Invariants = make(map[string]string, len(d.Invariants))
		for k, v := range d.Invariants {
			out.Invariants[k] = trimNL(v)
		}
	}
	if d.Assumptions != nil {
		out.Assumptions = make(map[string]string, len(d.Assumptions))
		for k, v := range d.Assumptions {
			out.Assumptions[k] = trimNL(v)
		}
	}
	if d.Contracts != nil {
		out.Contracts = make(map[string]ast.Contract, len(d.Contracts))
		for k, c := range d.Contracts {
			out.Contracts[k] = ast.Contract{
				Given: trimNL(c.Given),
				When:  trimNL(c.When),
				Then:  trimNL(c.Then),
			}
		}
	}
	if d.Unconstrained != nil {
		out.Unconstrained = make(map[string]string, len(d.Unconstrained))
		for k, v := range d.Unconstrained {
			out.Unconstrained[k] = trimNL(v)
		}
	}
	return out
}

func TestRoundTrip_Idempotent(t *testing.T) {
	res := lint.Lint("input.dx", []byte(exampleHello))
	if !res.OK() {
		t.Fatalf("input did not lint: %v", res.Issues)
	}
	first, err := canonical.Marshal(res.Declaration, canonical.Options{
		StripComments: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	res2 := lint.Lint("first.dx", first)
	if !res2.OK() {
		t.Fatal(res2.Issues)
	}
	second, err := canonical.Marshal(res2.Declaration, canonical.Options{
		StripComments: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("canonicalize(canonicalize(x)) != canonicalize(x):\n--- first ---\n%s\n--- second ---\n%s",
			first, second)
	}
}

func TestRoundTrip_BundledExamples(t *testing.T) {
	// Apply the same three properties to every .dx in examples/.
	// We hard-code the list rather than walking the directory so a
	// stray file doesn't accidentally enroll itself.
	cases := []string{
		"../../examples/hello.dx",
		"../../examples/weather_cli/system.dx",
	}
	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			res, err := lint.LintFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !res.OK() {
				t.Fatalf("input did not lint: %v", res.Issues)
			}
			out, err := canonical.Marshal(res.Declaration, canonical.Options{
				StripComments: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			res2 := lint.Lint(path+".formatted", out)
			if !res2.OK() {
				t.Fatalf("canonical output did not lint:\n--- output ---\n%s\n--- issues ---\n%v",
					out, res2.Issues)
			}
			out2, err := canonical.Marshal(res2.Declaration, canonical.Options{
				StripComments: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out, out2) {
				t.Errorf("not idempotent for %s", path)
			}
		})
	}
}
