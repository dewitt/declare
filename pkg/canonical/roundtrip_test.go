package canonical_test

// Cross-package property tests for the canonicalizer. These live in
// the _test package so they can import pkg/lint, which would
// otherwise create an import cycle with pkg/canonical (canonical
// does not import lint at runtime).
//
// The properties asserted here are the load-bearing contracts for
// `dx fmt`:
//
//   - Round-trip soundness: lint(canonical(d)) decodes to a
//     Declaration equal to d; in particular, the output is always
//     lintable.
//   - Idempotency: canonical(canonical(d)) is byte-identical to
//     canonical(d).
//
// Together, these mean an architect can run `dx fmt -w` on any
// linted declaration and the file will (a) still lint, (b) keep
// the same observable AST, and (c) be a fixed point under further
// formatting.

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/dewitt/dx/pkg/canonical"
	"github.com/dewitt/dx/pkg/lint"
)

const exampleHello = `# hello-world

## Intent

**Primary:** Greet a user by name on standard output.

**Secondary:**

- Be friendly.
- Exit cleanly.

## Invariants

### iface_stdout

Writes a single UTF-8 line to stdout terminated by ` + "`" + `\n` + "`" + `.

### perf_startup_ms

Cold-start latency must remain under 50ms on commodity hardware.

## Assumptions

### greeting.format

The greeting is "Hello, <name>!".

## Contracts

### greets_named_user

**Given:** The argument vector contains exactly one non-empty name.

**When:** The binary is invoked.

**Then:** stdout contains ` + "`" + `Hello, <name>!\n` + "`" + ` and the exit code is 0.

## Unconstrained

### language

Any language with a stable POSIX runtime is acceptable.
`

func TestRoundTrip_LintsCleanly(t *testing.T) {
	res := lint.Lint("input.md", []byte(exampleHello))
	if !res.OK() {
		t.Fatalf("input did not lint: %v", res.Issues)
	}
	out, err := canonical.Marshal(res.Declaration, canonical.Options{})
	if err != nil {
		t.Fatal(err)
	}
	res2 := lint.Lint("formatted.md", out)
	if !res2.OK() {
		t.Fatalf("canonical output did not lint:\n--- output ---\n%s\n--- issues ---\n%v",
			out, res2.Issues)
	}
}

func TestRoundTrip_PreservesAST(t *testing.T) {
	res := lint.Lint("input.md", []byte(exampleHello))
	if !res.OK() {
		t.Fatalf("input did not lint: %v", res.Issues)
	}
	out, err := canonical.Marshal(res.Declaration, canonical.Options{})
	if err != nil {
		t.Fatal(err)
	}
	res2 := lint.Lint("formatted.md", out)
	if !res2.OK() {
		t.Fatal(res2.Issues)
	}

	a, b := res.Declaration, res2.Declaration
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

func TestRoundTrip_Idempotent(t *testing.T) {
	res := lint.Lint("input.md", []byte(exampleHello))
	if !res.OK() {
		t.Fatalf("input did not lint: %v", res.Issues)
	}
	first, err := canonical.Marshal(res.Declaration, canonical.Options{})
	if err != nil {
		t.Fatal(err)
	}
	res2 := lint.Lint("first.md", first)
	if !res2.OK() {
		t.Fatal(res2.Issues)
	}
	second, err := canonical.Marshal(res2.Declaration, canonical.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("canonicalize(canonicalize(x)) != canonicalize(x):\n--- first ---\n%s\n--- second ---\n%s",
			first, second)
	}
}

func TestRoundTrip_BundledExamples(t *testing.T) {
	// Apply the same three properties to every bundled declaration.
	cases := []string{
		"../../examples/hello.md",
		"../../examples/weather_cli/system.md",
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
			out, err := canonical.Marshal(res.Declaration, canonical.Options{})
			if err != nil {
				t.Fatal(err)
			}
			res2 := lint.Lint(path+".formatted", out)
			if !res2.OK() {
				t.Fatalf("canonical output did not lint:\n--- output ---\n%s\n--- issues ---\n%v",
					out, res2.Issues)
			}
			out2, err := canonical.Marshal(res2.Declaration, canonical.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(out, out2) {
				t.Errorf("not idempotent for %s", path)
			}
		})
	}
}
