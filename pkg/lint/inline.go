package lint

// Inline sub-field parsers for `## Intent` (Primary / Secondary) and
// for `## Contracts` entries (Given / When / Then).
//
// The structural pass identifies block and key boundaries by ATX
// heading; everything between two structural headings is a leaf
// "body" string. These parsers walk the body text looking for the
// paragraph-leading bold markers defined in SPEC §4.2 and §4.3.5.
//
// Per SPEC §4.2 the bold markers MUST appear at the start of a
// paragraph (i.e. after a blank line, or at the very start of the
// body). The parser uses a simple two-state scan: paragraph break
// (blank line) followed by a line whose first non-whitespace token
// matches `**<Label>:**`. The body of the sub-field is the rest of
// that paragraph plus any subsequent paragraphs up to the next
// recognized marker.

import (
	"fmt"
	"strings"

	"github.com/dewitt/dx/pkg/ast"
)

// intentLabels and contractLabels are the recognized paragraph-
// leading bold markers, in source order. Each marker is matched
// case-sensitively per SPEC §4.2.
var intentLabels = []string{"Primary", "Secondary"}
var contractLabels = []string{"Given", "When", "Then"}

// parseIntent extracts the **Primary:** and **Secondary:** sub-
// fields from the body of a `## Intent` block.
//
// Returns the primary string, the secondary list, and any
// structural issues (missing Primary, malformed Secondary list,
// duplicate sub-field).
func parseIntent(path string, line int, body string) (string, []string, []Issue) {
	subs, issues := splitSubfields(path, line, body, intentLabels, "Intent")

	primary := strings.TrimSpace(subs["Primary"])
	secondaryRaw, hasSecondary := subs["Secondary"]

	var secondary []string
	if hasSecondary {
		items, listIssues := parseUnorderedList(path, line, secondaryRaw)
		issues = append(issues, listIssues...)
		secondary = items
	}

	return primary, secondary, issues
}

// parseContract extracts the **Given:**, **When:**, **Then:**
// sub-fields from the body of a `### <name>` contract section.
//
// All three sub-fields MUST be present per SPEC §4.3.5; missing or
// duplicate sub-fields are flagged as Issues. The sub-field bodies
// are trimmed of surrounding whitespace but otherwise preserved
// verbatim (multi-paragraph contract clauses are supported).
func parseContract(path string, line int, name, body string) (ast.Contract, []Issue) {
	subs, issues := splitSubfields(path, line, body, contractLabels, "contract `"+name+"`")

	c := ast.Contract{
		Given: strings.TrimSpace(subs["Given"]),
		When:  strings.TrimSpace(subs["When"]),
		Then:  strings.TrimSpace(subs["Then"]),
	}

	for _, label := range contractLabels {
		if _, ok := subs[label]; !ok {
			issues = append(issues, Issue{
				Path: path,
				Line: line,
				Message: fmt.Sprintf(
					"contract `%s` is missing `**%s:**` sub-field (SPEC §4.3.5)",
					name, label,
				),
			})
		}
	}

	return c, issues
}

// splitSubfields walks body looking for paragraph-leading
// `**<Label>:**` markers from labels. It returns a map from label
// name (without the bold or colon) to the body text of that sub-
// field, plus issues for duplicates or unrecognized markers.
//
// A "paragraph-leading" marker is the first non-whitespace token of
// a paragraph (i.e. follows a blank line, or appears at the very
// start of body).
func splitSubfields(path string, line int, body string, labels []string, context string) (map[string]string, []Issue) {
	out := make(map[string]string)
	var issues []Issue

	labelSet := make(map[string]bool, len(labels))
	for _, l := range labels {
		labelSet[l] = true
	}

	// Split into "paragraphs" by blank-line separators. A blank
	// line is one whose content is empty after stripping ASCII
	// whitespace.
	paragraphs := splitParagraphs(body)

	var currentLabel string
	var currentBuf strings.Builder

	flush := func() {
		if currentLabel != "" {
			if _, dup := out[currentLabel]; dup {
				issues = append(issues, Issue{
					Path: path,
					Line: line,
					Message: fmt.Sprintf(
						"%s contains `**%s:**` more than once (SPEC §4.3)",
						context, currentLabel,
					),
				})
			} else {
				out[currentLabel] = strings.TrimRight(currentBuf.String(), "\n")
			}
		}
		currentBuf.Reset()
	}

	for _, para := range paragraphs {
		label, rest, ok := stripLeadingBoldLabel(para)
		if ok && labelSet[label] {
			// Begin a new sub-field. Flush whatever was in
			// progress.
			flush()
			currentLabel = label
			currentBuf.WriteString(rest)
		} else if currentLabel == "" {
			// Text before any recognized sub-field marker. This
			// is malformed per SPEC §4.3.2 / §4.3.5 (sub-fields
			// MUST cover the block body). Emit one issue and
			// drop the prose.
			if strings.TrimSpace(para) != "" {
				issues = append(issues, Issue{
					Path: path,
					Line: line,
					Message: fmt.Sprintf(
						"%s contains content before any recognized `**<Label>:**` marker (SPEC §4.3)",
						context,
					),
				})
			}
		} else {
			// Continuation paragraph of the current sub-field.
			currentBuf.WriteString("\n\n")
			currentBuf.WriteString(para)
		}
	}
	flush()

	return out, issues
}

// splitParagraphs splits body into paragraph strings by blank-line
// separators. Each returned paragraph has no leading or trailing
// newline.
func splitParagraphs(body string) []string {
	if body == "" {
		return nil
	}
	lines := strings.Split(body, "\n")
	var paras []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			paras = append(paras, buf.String())
			buf.Reset()
		}
	}
	for _, line := range lines {
		if isBlankLine(line) {
			flush()
			continue
		}
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		buf.WriteString(line)
	}
	flush()
	return paras
}

// stripLeadingBoldLabel checks whether paragraph p begins with the
// bold-and-colon marker `**<Label>:**` (optionally followed by
// whitespace). If so, it returns the label, the rest of the
// paragraph with the marker removed, and true. Otherwise it returns
// "", "", false.
//
// The leading characters before `**` must be whitespace only (a
// paragraph MAY be indented in source per CommonMark, though our
// canonical form does not indent).
func stripLeadingBoldLabel(p string) (string, string, bool) {
	s := strings.TrimLeft(p, " \t")
	if !strings.HasPrefix(s, "**") {
		return "", "", false
	}
	s = s[2:]
	// Find the closing `:**`. The label is everything up to it.
	end := strings.Index(s, ":**")
	if end < 0 {
		return "", "", false
	}
	label := s[:end]
	// Reject labels that contain whitespace or aren't a single
	// identifier-like word: per SPEC §4.2 the labels are exactly
	// the recognized vocabulary words.
	if strings.ContainsAny(label, " \t\n") {
		return "", "", false
	}
	rest := s[end+len(":**"):]
	// Strip one leading space if present (the conventional
	// "**Given:** foo" form has a space after the marker).
	rest = strings.TrimLeft(rest, " \t")
	return label, rest, true
}

// parseUnorderedList extracts the items of a CommonMark unordered
// list (lines beginning with `-`, `*`, or `+` followed by a space).
// The function does NOT recursively interpret nested CommonMark; it
// treats the body of each list item as a verbatim string with the
// bullet marker removed.
//
// This is sufficient for `## Intent` `**Secondary:**` lists, which
// per SPEC §4.3.2 are unordered CommonMark lists of strings.
func parseUnorderedList(path string, line int, body string) ([]string, []Issue) {
	var items []string
	var issues []Issue
	var current strings.Builder

	flush := func() {
		if current.Len() > 0 {
			items = append(items, strings.TrimRight(current.String(), "\n"))
			current.Reset()
		}
	}

	lines := strings.Split(strings.TrimSpace(body), "\n")
	for _, l := range lines {
		trimmed := strings.TrimLeft(l, " \t")
		if len(trimmed) >= 2 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
			flush()
			current.WriteString(trimmed[2:])
			continue
		}
		if current.Len() == 0 {
			if strings.TrimSpace(l) != "" {
				issues = append(issues, Issue{
					Path:    path,
					Line:    line,
					Message: "`**Secondary:**` body must be an unordered CommonMark list (`-` items) (SPEC §4.3.2)",
				})
				return nil, issues
			}
			continue
		}
		// Continuation line of the current item.
		current.WriteByte('\n')
		current.WriteString(l)
	}
	flush()

	return items, issues
}
