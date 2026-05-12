// Package lint validates `.dx` files against the structural rules in SPEC.md.
//
// The lint pipeline runs four passes, in order:
//
//  1. Parse the file as YAML 1.2 (gopkg.in/yaml.v3) into a node graph.
//  2. Walk the node graph to enforce SPEC §2 physical rules
//     (no anchors/aliases, literal scalars only, no custom tags).
//     See physical.go.
//  3. Strict-decode the node graph into the AST (unknown fields rejected).
//  4. Verify required top-level keys are present per SPEC §3.
//
// Pass 2 runs before pass 3 because anchors/aliases would otherwise be
// silently followed by the decoder, producing surprising downstream
// errors. All passes are non-fatal at the function boundary -- problems
// surface as Issues so callers can render them uniformly.
package lint

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dewitt/declare/pkg/ast"
)

// Issue is a single linter finding tied to a source location when available.
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
	Declaration *ast.Declaration // nil if decoding failed
	Issues      []Issue
}

// OK reports whether the file produced zero issues.
func (r *Result) OK() bool { return len(r.Issues) == 0 }

// LintFile reads the named file and returns a Result describing all issues
// detected. A non-nil error is returned only for I/O failures; structural
// problems are reported as Issues.
//
// Most callers should prefer LintSource, which also accepts a
// `<rev>:<path>` git revision spec. LintFile is retained for the
// strict file-path case and for backward compatibility.
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
// git checkout; resolution shells out to `git show <rev>:<path>` and
// surfaces git's diagnostic output verbatim on failure.
//
// As with LintFile, a non-nil error is returned only for I/O or
// resolution failures; structural problems are reported as Issues.
func LintSource(source string) (*Result, error) {
	data, displayPath, err := readSource(source)
	if err != nil {
		return nil, err
	}
	return Lint(displayPath, data), nil
}

// Lint decodes data as a `.dx` declaration and returns the diagnostic Result.
// It never returns an error -- all problems are surfaced as Issues so callers
// can present them uniformly.
func Lint(path string, data []byte) *Result {
	res := &Result{Path: path}

	// Pass 1: parse to a node graph so subsequent passes can inspect
	// physical features the strict decoder would erase (see SPEC §2).
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		res.Issues = append(res.Issues, issueFromYAMLErr(path, err))
		return res
	}

	// Pass 2: SPEC §2 physical rules. Run before strict-decode because
	// anchors/aliases would otherwise be silently dereferenced and
	// folded scalars would be invisibly normalized into Go strings.
	res.Issues = append(res.Issues, validatePhysical(path, &root)...)

	// Pass 2b: leaf-type rules for invariants/assumptions/unconstrained.
	// These run before strict-decode so the agent gets line-tagged
	// diagnostics instead of yaml.v3's positionless type errors.
	res.Issues = append(res.Issues, validateLeafTypes(path, &root)...)

	// Pass 3: strict decode into the AST.
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)

	var decl ast.Declaration
	if err := decoder.Decode(&decl); err != nil {
		// io.EOF means the file was empty.
		if errors.Is(err, io.EOF) {
			res.Issues = append(res.Issues, Issue{
				Path:    path,
				Message: "empty file: a `.dx` declaration requires at least `system`, `intent`, `invariants`, and `assumptions`",
			})
			return res
		}
		res.Issues = append(res.Issues, issueFromYAMLErr(path, err))
		return res
	}
	decl.Node = &root
	res.Declaration = &decl

	// Pass 4: structural validation of required blocks (SPEC §3).
	res.Issues = append(res.Issues, validateRequired(path, &decl)...)

	return res
}

// validateRequired enforces the "Required" markers in SPEC §3.
func validateRequired(path string, d *ast.Declaration) []Issue {
	var issues []Issue

	if strings.TrimSpace(d.System) == "" {
		issues = append(issues, Issue{Path: path, Message: "missing required key `system` (SPEC §3)"})
	}
	if strings.TrimSpace(d.Intent.Primary) == "" {
		issues = append(issues, Issue{Path: path, Message: "missing required key `intent.primary` (SPEC §3)"})
	}
	// `invariants` and `assumptions` must be present as keys, even when empty
	// (SPEC §3 explicitly calls out a "zero-assumption" state). The strict
	// decoder will have populated these as nil maps if absent; we cannot
	// distinguish absent-from-empty without consulting the node graph.
	if d.Invariants == nil && !hasTopLevelKey(d.Node, "invariants") {
		issues = append(issues, Issue{Path: path, Message: "missing required key `invariants` (SPEC §3)"})
	}
	if d.Assumptions == nil && !hasTopLevelKey(d.Node, "assumptions") {
		issues = append(issues, Issue{Path: path, Message: "missing required key `assumptions` (SPEC §3)"})
	}

	return issues
}

// hasTopLevelKey reports whether the document's root mapping contains the
// given key. It tolerates malformed graphs by returning false.
func hasTopLevelKey(root *yaml.Node, key string) bool {
	if root == nil || len(root.Content) == 0 {
		return false
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		if doc.Content[i].Value == key {
			return true
		}
	}
	return false
}

// issueFromYAMLErr converts a yaml.v3 error -- which often embeds line numbers
// in its message -- into an Issue. yaml.v3 exposes a TypeError with per-field
// messages; we flatten it into individual Issues for nicer reporting.
func issueFromYAMLErr(path string, err error) Issue {
	var te *yaml.TypeError
	if errors.As(err, &te) && len(te.Errors) > 0 {
		// Join all type-errors into one Issue for now; future revisions
		// can split them out once we plumb per-error positions.
		return Issue{Path: path, Message: strings.Join(te.Errors, "; ")}
	}
	return Issue{Path: path, Message: err.Error()}
}
