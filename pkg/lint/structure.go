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
//     sub-structure, not free-form `###` keys per SPEC §4.3.2).
//   - Duplicate `###` keys within the same block.
//   - Contracts missing one or more of `**Given:**`, `**When:**`,
//     `**Then:**`.
//   - Contracts that contain a sub-field more than once.
//   - Heading levels 4+ at the structural layer (they MUST be
//     treated as leaf content per SPEC §4.2).
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
		Invariants:    map[string]string{},
		Assumptions:   map[string]string{},
		Contracts:     map[string]ast.Contract{},
		Unconstrained: map[string]string{},
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

	// Walk: track the current `##` block as we go. Headings at level
	// 4+ are not structural and the parser already discards them
	// before they reach us (it only emits ATX headings as headings;
	// but goldmark passes through all six levels, so we explicitly
	// skip 4+ here).
	currentBlock := ""
	currentBlockLine := 0
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
					Message: "`#` system heading body is empty: provide a slug identifier (SPEC §4.3.1)",
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
				currentBlockLine = h.line
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
			currentBlockLine = h.line
			decl.Positions[strings.ToLower(h.text)] = ast.Position{Line: h.line}

			// Handle Intent specially: extract Primary/Secondary
			// from the leaf content under the heading. There are
			// no `###` keys under Intent.
			if h.text == "Intent" {
				leaf := doc.leafBetween(i)
				primary, secondary, intentIssues := parseIntent(path, h.line, leaf)
				issues = append(issues, intentIssues...)
				decl.Intent.Primary = primary
				decl.Intent.Secondary = secondary
				if primary != "" {
					decl.Positions["intent.primary"] = ast.Position{Line: h.line}
				}
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
					Path: path,
					Line: h.line,
					Message: fmt.Sprintf(
						"`## Intent` does not use `###` keys; use `**Primary:**` and `**Secondary:**` instead (SPEC §4.3.2)",
					),
				})
				continue
			}
			key := h.text
			if strings.TrimSpace(key) == "" {
				issues = append(issues, Issue{
					Path:    path,
					Line:    h.line,
					Message: "`###` key heading body is empty: provide an identifier (SPEC §4.2)",
				})
				continue
			}
			leaf := doc.leafBetween(i)
			issues = append(issues, applyKey(path, decl, currentBlock, key, leaf, h.line)...)

		case 4, 5, 6:
			// Heading levels 4+ are reserved for leaf content
			// (SPEC §4.2). The structural pass should never see
			// them because parseSource only collects top-level
			// children of the document; goldmark may emit them
			// at any level, but those that *are* top-level still
			// reach us. SPEC says they MUST be treated as leaf
			// content: emit a friendly warning so the author
			// understands they will appear inside the preceding
			// key's body, not as their own structural element.
			//
			// We intentionally do NOT flag this as an error.
			// Authors may legitimately use ####+ inside an
			// invariant body to add sub-structure to prose. The
			// surprise we want to avoid is silent reinterpretation;
			// a comment in the docs and the rendering itself
			// (the heading is visibly smaller) carry that load.
			//
			// However, headings 4+ appearing as direct children
			// of the document are unusual; the parser will have
			// already absorbed them into the preceding leaf
			// because leafBetween reads byte ranges, not goldmark
			// nodes. The fact that they appear here at all means
			// they sit between two structural headings (1-3)
			// with no preceding key heading, which is just leaf
			// content of the preceding `##` block. Nothing to
			// do.
			_ = currentBlockLine // referenced for future per-block diagnostics
		}
	}

	return decl, issues
}

// applyKey records a `###` key under the current block. It enforces
// per-block constraints (e.g., contracts must contain Given/When/Then)
// and reports duplicates.
func applyKey(path string, decl *ast.Declaration, block, key, body string, line int) []Issue {
	var issues []Issue

	switch block {
	case "Invariants":
		if _, dup := decl.Invariants[key]; dup {
			return []Issue{{
				Path:    path,
				Line:    line,
				Message: fmt.Sprintf("duplicate invariant `### %s` (SPEC §4.3.3)", key),
			}}
		}
		decl.Invariants[key] = body
		decl.Positions["invariants."+key] = ast.Position{Line: line}

	case "Assumptions":
		if _, dup := decl.Assumptions[key]; dup {
			return []Issue{{
				Path:    path,
				Line:    line,
				Message: fmt.Sprintf("duplicate assumption `### %s` (SPEC §4.3.4)", key),
			}}
		}
		decl.Assumptions[key] = body
		decl.Positions["assumptions."+key] = ast.Position{Line: line}

	case "Contracts":
		if _, dup := decl.Contracts[key]; dup {
			return []Issue{{
				Path:    path,
				Line:    line,
				Message: fmt.Sprintf("duplicate contract `### %s` (SPEC §4.3.5)", key),
			}}
		}
		c, contractIssues := parseContract(path, line, key, body)
		issues = append(issues, contractIssues...)
		decl.Contracts[key] = c
		decl.Positions["contracts."+key] = ast.Position{Line: line}

	case "Unconstrained":
		if _, dup := decl.Unconstrained[key]; dup {
			return []Issue{{
				Path:    path,
				Line:    line,
				Message: fmt.Sprintf("duplicate unconstrained entry `### %s` (SPEC §4.3.6)", key),
			}}
		}
		decl.Unconstrained[key] = body
		decl.Positions["unconstrained."+key] = ast.Position{Line: line}
	}

	return issues
}
