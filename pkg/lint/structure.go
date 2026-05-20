package lint

// Structural-pass walker for SPEC §4.2 / §4.3 (v0.2.0 CommonMark
// serialization).
//
// buildDeclaration consumes the heading sequence produced by
// parseSource and folds it into the AST defined in pkg/ast. The
// function also reports structural issues:
//
//   - Multiple `#` system headings, or a `#` heading not at the top.
//   - `##` block headings outside the canonical vocabulary.
//   - Duplicate `##` block headings.
//   - `###` key headings outside any `##` block.
//   - `###` key headings inside `## Intent` (Intent uses a fixed
//     body shape, not free-form `###` keys per SPEC §4.3.2).
//   - `###` headings whose slug-reduced form collides with another
//     in the same block.
//   - Contracts missing one or more of `**Given**`, `**When**`,
//     `**Then**`.
//   - Contracts that contain a sub-field more than once.
//
// Issues are reported with 1-based source line numbers so the
// agent and the editor can navigate to them directly.

import (
	"fmt"
	"strings"

	"github.com/dewitt/dx/pkg/ast"
)

// canonicalBlocks names the SPEC §4.2 closed vocabulary of `##`
// block headings, in canonical order.
var canonicalBlocks = []string{
	"Intent",
	"Invariants",
	"Assumptions",
	"Contracts",
	"Unconstrained",
}

// canonicalBlockSet is the lookup form.
var canonicalBlockSet = func() map[string]bool {
	m := make(map[string]bool, len(canonicalBlocks))
	for _, b := range canonicalBlocks {
		m[b] = true
	}
	return m
}()

// buildDeclaration walks doc.headings and produces an AST plus a
// list of structural issues. The AST is always returned (never nil)
// so that downstream lint passes can run; missing data is left as
// the zero value of each field.
func buildDeclaration(path string, doc *document) (*ast.Declaration, []Issue) {
	decl := &ast.Declaration{
		Invariants:    map[string]ast.Entry{},
		Assumptions:   map[string]ast.Entry{},
		Contracts:     map[string]ast.Contract{},
		Unconstrained: map[string]ast.Entry{},
		Positions:     map[string]ast.Position{},
		BlocksPresent: map[string]bool{},
	}
	var issues []Issue

	if len(doc.headings) == 0 {
		issues = append(issues, Issue{
			Path:    path,
			Message: "no headings found: a declaration begins with `# <system>` (SPEC §4.2)",
		})
		return decl, issues
	}

	currentBlock := ""
	systemSeen := false

	for i, h := range doc.headings {
		switch h.level {
		case 1:
			// `#` system heading.
			if systemSeen {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: "duplicate `#` system heading: a declaration MUST contain exactly one (SPEC §4.2)",
				})
				continue
			}
			if i != 0 {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: "`#` system heading MUST be the first heading in the document (SPEC §4.2)",
				})
			}
			systemSeen = true
			decl.System = h.text
			decl.Positions["system"] = ast.Position{Line: h.line}
			if strings.TrimSpace(h.text) == "" {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: "`#` system heading body is empty: provide a system name (SPEC §4.3.1)",
				})
			}

		case 2:
			// `##` block heading.
			if !canonicalBlockSet[h.text] {
				issues = append(issues, Issue{
					Path: path,
					Line: h.line,
					Message: fmt.Sprintf(
						"`## %s` is not a canonical block heading; expected one of %s (SPEC §4.2)",
						h.text, strings.Join(canonicalBlocks, ", "),
					),
				})
				// Treat as unknown so subsequent `###` keys are
				// flagged as "outside a recognized block".
				currentBlock = ""
				continue
			}
			if decl.BlocksPresent[h.text] {
				issues = append(issues, Issue{
					Path: path,
					Line: h.line,
					Message: fmt.Sprintf(
						"duplicate `## %s` block: each block MAY appear at most once (SPEC §4.2)",
						h.text,
					),
				})
				continue
			}
			decl.BlocksPresent[h.text] = true
			currentBlock = h.text
			decl.Positions[strings.ToLower(h.text)] = ast.Position{Line: h.line}

			// Handle Intent specially: capture the body as either a
			// single paragraph or an unordered list. There are no
			// `###` keys under Intent.
			if h.text == "Intent" {
				leaf := doc.leafBetween(i)
				items, intentIssues := parseIntent(path, h.line, leaf)
				issues = append(issues, intentIssues...)
				decl.Intent = items
			}

		case 3:
			// `###` key heading.
			if currentBlock == "" {
				issues = append(issues, Issue{
					Path: path,
					Line: h.line,
					Message: fmt.Sprintf(
						"`### %s` appears outside any recognized `##` block (SPEC §4.2)",
						h.text,
					),
				})
				continue
			}
			if currentBlock == "Intent" {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: "`## Intent` does not use `###` keys; write the intent as a single paragraph or as an unordered list (SPEC §4.3.2)",
				})
				continue
			}
			heading := h.text
			if strings.TrimSpace(heading) == "" {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: "`###` key heading body is empty: provide a name (SPEC §4.2)",
				})
				continue
			}
			slug := ast.Slug(heading)
			if slug == "" {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: fmt.Sprintf("`### %s` reduces to an empty slug (no ASCII alphanumerics): provide a name with at least one letter or digit (SPEC §4.2)", heading),
				})
				continue
			}
			leaf := doc.leafBetween(i)
			issues = append(issues, applyKey(path, decl, currentBlock, slug, heading, leaf, h.line)...)

		case 4, 5, 6:
			// Heading levels 4+ are reserved for leaf content
			// (SPEC §4.2). When they appear as top-level blocks
			// they sit inside the preceding leaf's byte range
			// and leafBetween captures them verbatim. Nothing
			// for the structural pass to do.
		}
	}

	return decl, issues
}

// applyKey records a `###` key (by its derived slug) under the
// current block. It enforces per-block constraints (e.g., contracts
// must contain Given/When/Then) and reports slug collisions.
func applyKey(path string, decl *ast.Declaration, block, slug, heading, body string, line int) []Issue {
	var issues []Issue

	switch block {
	case "Invariants":
		if existing, dup := decl.Invariants[slug]; dup {
			return []Issue{{
				Path: path,
				Line: line,
				Message: fmt.Sprintf(
					"`### %s` collides with `### %s` (both reduce to slug `%s`) (SPEC §4.2)",
					heading, existing.Heading, slug,
				),
			}}
		}
		decl.Invariants[slug] = ast.Entry{Heading: heading, Body: body}
		decl.Positions["invariants."+slug] = ast.Position{Line: line}

	case "Assumptions":
		if existing, dup := decl.Assumptions[slug]; dup {
			return []Issue{{
				Path: path,
				Line: line,
				Message: fmt.Sprintf(
					"`### %s` collides with `### %s` (both reduce to slug `%s`) (SPEC §4.2)",
					heading, existing.Heading, slug,
				),
			}}
		}
		decl.Assumptions[slug] = ast.Entry{Heading: heading, Body: body}
		decl.Positions["assumptions."+slug] = ast.Position{Line: line}

	case "Contracts":
		if existing, dup := decl.Contracts[slug]; dup {
			return []Issue{{
				Path: path,
				Line: line,
				Message: fmt.Sprintf(
					"`### %s` collides with `### %s` (both reduce to slug `%s`) (SPEC §4.2)",
					heading, existing.Heading, slug,
				),
			}}
		}
		c, contractIssues := parseContract(path, line, heading, body)
		issues = append(issues, contractIssues...)
		c.Heading = heading
		decl.Contracts[slug] = c
		decl.Positions["contracts."+slug] = ast.Position{Line: line}

	case "Unconstrained":
		if existing, dup := decl.Unconstrained[slug]; dup {
			return []Issue{{
				Path: path,
				Line: line,
				Message: fmt.Sprintf(
					"`### %s` collides with `### %s` (both reduce to slug `%s`) (SPEC §4.2)",
					heading, existing.Heading, slug,
				),
			}}
		}
		decl.Unconstrained[slug] = ast.Entry{Heading: heading, Body: body}
		decl.Positions["unconstrained."+slug] = ast.Position{Line: line}
	}

	return issues
}
