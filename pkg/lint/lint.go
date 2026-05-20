// Package lint validates declaration files against the structural
// rules in SPECIFICATION.md (v0.2.0 CommonMark serialization).
//
// The lint pipeline runs three passes, in order:
//
//  1. Scan the source for ATX headings using goldmark (see parse.go).
//     Top-level headings inside code blocks, block quotes, or list
//     items are not counted as structural; this prevents false
//     positives from `#` characters that appear inside leaf content.
//  2. Build the AST by mapping the structural heading sequence onto
//     the components defined in SPEC §3 (see structure.go). The
//     contents between structural boundaries are captured as opaque
//     leaf text per SPEC §4.2.
//  3. Verify required blocks are present per SPEC §4.2 / §4.3.
//
// All passes are non-fatal at the function boundary: problems are
// surfaced as Issues so callers can render them uniformly.
package lint

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dewitt/dx/pkg/ast"
)

// Issue is a single linter finding tied to a source location when
// available.
type Issue struct {
	Path    string // file path
	Line    int    // 1-based; 0 when unknown
	Column  int    // 1-based; 0 when unknown
	Message string
}

func (i Issue) String() string {
	if i.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", i.Path, i.Line, i.Column, i.Message)
	}
	return fmt.Sprintf("%s: %s", i.Path, i.Message)
}

// Result aggregates the outcome of linting a single file.
type Result struct {
	Path        string
	Declaration *ast.Declaration // nil if structural parsing failed
	Issues      []Issue
}

// OK reports whether the file produced zero issues.
func (r *Result) OK() bool { return len(r.Issues) == 0 }

// LintFile reads the named file and returns a Result describing all
// issues detected. A non-nil error is returned only for I/O failures;
// structural problems are reported as Issues.
//
// Most callers should prefer LintSource, which also accepts a
// `<rev>:<path>` git revision spec.
func LintFile(path string) (*Result, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return Lint(path, data), nil
}

// LintSource reads the named source -- either a filesystem path or a
// `<rev>:<path>` git revision spec, mirroring `git show` syntax --
// and returns a Result describing all issues detected.
//
// The revision form requires the working directory to be inside a
// git checkout; resolution shells out to `git show <rev>:<path>`
// and surfaces git's diagnostic output verbatim on failure.
func LintSource(source string) (*Result, error) {
	data, displayPath, err := readSource(source)
	if err != nil {
		return nil, err
	}
	return Lint(displayPath, data), nil
}

// Lint decodes data as a declaration and returns the diagnostic
// Result. It never returns an error: all problems are surfaced as
// Issues so callers can present them uniformly.
func Lint(path string, data []byte) *Result {
	res := &Result{Path: path}

	if len(data) == 0 {
		res.Issues = append(res.Issues, Issue{
			Path:    path,
			Message: "empty file: a declaration requires at minimum `# <system>` plus `## Intent`, `## Invariants`, and `## Assumptions` (SPEC §4.3)",
		})
		return res
	}

	// Pass 1: scan for structural headings.
	doc := parseSource(data)

	// Pass 2: build the AST and surface structural issues.
	decl, issues := buildDeclaration(path, doc)
	res.Issues = append(res.Issues, issues...)
	if decl != nil {
		res.Declaration = decl
	}

	// Pass 3: verify required blocks are present.
	if decl != nil {
		res.Issues = append(res.Issues, validateRequired(path, decl)...)
	}

	return res
}

// validateRequired enforces the REQUIRED markers in SPEC §4.3.
//
// SPEC §4.3 mandates a non-empty `#` system heading, a non-empty
// `## Intent` body, an `## Invariants` heading, and an
// `## Assumptions` heading. The Invariants and Assumptions blocks
// MAY have zero `###` children (per §4.3.3 and §4.3.4) but the
// `##` heading itself MUST be present; that distinction is
// recorded in BlocksPresent.
func validateRequired(path string, d *ast.Declaration) []Issue {
	var issues []Issue

	if strings.TrimSpace(d.System) == "" {
		issues = append(issues, Issue{
			Path:    path,
			Line:    posLine(d, "system"),
			Message: "missing required `#` system heading (SPEC §4.3.1)",
		})
	}
	if !d.BlocksPresent["Intent"] {
		issues = append(issues, Issue{
			Path:    path,
			Message: "missing required `## Intent` block (SPEC §4.3.2)",
		})
	} else if intentIsEmpty(d.Intent) {
		issues = append(issues, Issue{
			Path:    path,
			Line:    posLine(d, "intent"),
			Message: "missing intent body under `## Intent`: write either a single paragraph or an unordered list (SPEC §4.3.2)",
		})
	}
	if !d.BlocksPresent["Invariants"] {
		issues = append(issues, Issue{
			Path:    path,
			Message: "missing required `## Invariants` block (SPEC §4.3.3)",
		})
	}
	if !d.BlocksPresent["Assumptions"] {
		issues = append(issues, Issue{
			Path:    path,
			Message: "missing required `## Assumptions` block (SPEC §4.3.4)",
		})
	}

	return issues
}

// intentIsEmpty reports whether the intent body is effectively
// empty: zero items, or items that are all whitespace.
func intentIsEmpty(intent []string) bool {
	for _, item := range intent {
		if strings.TrimSpace(item) != "" {
			return false
		}
	}
	return true
}

// posLine returns the 1-based source line recorded for key, or 0 if
// no position was recorded.
func posLine(d *ast.Declaration, key string) int {
	if d.Positions == nil {
		return 0
	}
	return d.Positions[key].Line
}
