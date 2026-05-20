package lint

// Sub-field and intent-body parsers.
//
// parseIntent reads the body of a `## Intent` block: either a
// single CommonMark paragraph (one intent) or an unordered list
// (multiple intents in priority order). The two forms are mutually
// exclusive; mixing them is a structural error.
//
// parseContract reads the body of a `### <name>` contract section
// looking for the paragraph-leading bold keywords `**Given**`,
// `**When**`, `**Then**` defined in SPEC §4.3.5.

import (
	"fmt"
	"strings"

	"github.com/dewitt/dx/pkg/ast"
)

// contractLabels are the recognized paragraph-leading bold markers
// inside a `### <name>` contract section, in canonical order.
var contractLabels = []string{"Given", "When", "Then"}

// parseIntent reads the body of a `## Intent` block and returns the
// list of intents in priority order. A single-paragraph body
// returns a one-element slice; an unordered-list body returns one
// element per list item. Mixing a paragraph and a list under the
// same heading is a structural error.
//
// An empty body returns a nil slice with no issues; the caller is
// responsible for flagging that as "missing intent body" if the
// Intent block is REQUIRED to be non-empty.
func parseIntent(path string, line int, body string) ([]string, []Issue) {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, nil
	}

	paragraphs := splitParagraphs(body)
	if len(paragraphs) == 0 {
		return nil, nil
	}

	// Classify each paragraph as either a list (first non-blank
	// line begins with `- `, `* `, or `+ `) or prose.
	var listParas, proseParas []string
	for _, p := range paragraphs {
		if isUnorderedListParagraph(p) {
			listParas = append(listParas, p)
		} else {
			proseParas = append(proseParas, p)
		}
	}

	// Mixed forms: emit an issue and fall back to treating the
	// whole body as a single intent so the caller still has
	// something to work with.
	if len(listParas) > 0 && len(proseParas) > 0 {
		return []string{body}, []Issue{{
			Path:    path,
			Line:    line,
			Message: "`## Intent` body mixes a paragraph and a list; choose one form (SPEC §4.3.2)",
		}}
	}

	if len(listParas) > 0 {
		// All paragraphs are list paragraphs. Flatten them into
		// items. CommonMark renders adjacent list paragraphs as
		// one logical list, so we treat them uniformly.
		var items []string
		var issues []Issue
		for _, p := range listParas {
			pItems, pIssues := parseUnorderedList(path, line, p)
			items = append(items, pItems...)
			issues = append(issues, pIssues...)
		}
		return items, issues
	}

	// All paragraphs are prose. Multiple prose paragraphs are
	// joined as a single intent (separated by a blank line in
	// canonical form); this preserves the architect's option to
	// write a longer intent across paragraphs.
	return []string{strings.Join(proseParas, "\n\n")}, nil
}

// isUnorderedListParagraph reports whether p's first non-blank
// line begins with a CommonMark unordered-list marker.
func isUnorderedListParagraph(p string) bool {
	lines := strings.Split(p, "\n")
	for _, l := range lines {
		trim := strings.TrimLeft(l, " \t")
		if trim == "" {
			continue
		}
		if len(trim) >= 2 && (trim[0] == '-' || trim[0] == '*' || trim[0] == '+') && trim[1] == ' ' {
			return true
		}
		return false
	}
	return false
}

// parseContract extracts the **Given**, **When**, **Then**
// sub-fields from the body of a `### <name>` contract section.
//
// All three sub-fields MUST be present per SPEC §4.3.5; missing or
// duplicate sub-fields are flagged as Issues. The sub-field bodies
// are trimmed of surrounding whitespace but otherwise preserved
// verbatim (multi-paragraph contract clauses are supported).
//
// The returned Contract has Heading unset; the caller (applyKey)
// fills it in.
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
					"contract `%s` is missing `**%s**` sub-field (SPEC §4.3.5)",
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
						"%s contains `**%s**` more than once (SPEC §4.3)",
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
			flush()
			currentLabel = label
			currentBuf.WriteString(rest)
		} else if currentLabel == "" {
			if strings.TrimSpace(para) != "" {
				issues = append(issues, Issue{
					Path: path,
					Line: line,
					Message: fmt.Sprintf(
						"%s contains content before any recognized `**Given**` / `**When**` / `**Then**` paragraph (SPEC §4.3)",
						context,
					),
				})
			}
		} else {
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
// no-colon bold-keyword marker `**<Label>** ` -- two asterisks, a
// single identifier-shaped word, two more asterisks, and a literal
// space. If so, it returns the label, the rest of the paragraph
// with the marker (including the trailing space) consumed, and
// true. Otherwise it returns "", "", false.
//
// The trailing-space requirement distinguishes this form from
// inline-bold prose like `**important**:`, `**bold-mid-sentence**`,
// or `**foo**.` -- those have no space after the closing `**`, so
// they are correctly left as leaf content.
//
// This parser is also conservative on the label itself: only ASCII
// letters are permitted (no spaces, digits, punctuation). The set
// of *recognized* labels is applied separately by the caller, so a
// well-formed `**Note** ...` paragraph that happens to start a
// leaf would be parsed here as label="Note" and then rejected as
// non-canonical by splitSubfields, which treats it as continuation
// prose -- preserving the original `**Note**` in the leaf text.
func stripLeadingBoldLabel(p string) (string, string, bool) {
	s := strings.TrimLeft(p, " \t")
	if !strings.HasPrefix(s, "**") {
		return "", "", false
	}
	body := s[2:]
	// Find the closing `**`. The label is everything up to it.
	end := strings.Index(body, "**")
	if end < 0 {
		return "", "", false
	}
	label := body[:end]
	if label == "" {
		return "", "", false
	}
	if !isLabelWord(label) {
		return "", "", false
	}
	rest := body[end+2:]
	// Require at least one space between the closing `**` and the
	// body. Without the space, this is inline-bold prose, not a
	// label.
	if !strings.HasPrefix(rest, " ") && !strings.HasPrefix(rest, "\t") {
		return "", "", false
	}
	rest = strings.TrimLeft(rest, " \t")
	return label, rest, true
}

// isLabelWord reports whether s is a single identifier-shaped
// label (one or more ASCII letters). The recognized vocabulary is
// applied separately by the caller; this function only checks that
// the candidate could be a label.
func isLabelWord(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	return true
}

// parseUnorderedList extracts the items of a CommonMark unordered
// list (lines beginning with `-`, `*`, or `+` followed by a space).
// The function does NOT recursively interpret nested CommonMark; it
// treats the body of each list item as a verbatim string with the
// bullet marker removed.
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
					Message: "list body must begin with an unordered CommonMark list item (`- `, `* `, or `+ `) (SPEC §4.3.2)",
				})
				return nil, issues
			}
			continue
		}
		// Continuation line of the current item; preserve the
		// original (possibly indented) prefix verbatim.
		current.WriteByte('\n')
		current.WriteString(l)
	}
	flush()

	return items, issues
}
