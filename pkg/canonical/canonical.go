// Package canonical produces a deterministic, byte-stable
// CommonMark representation of a parsed declaration (SPEC §4.5).
//
// The canonical form has these properties:
//
//  1. The `#` system heading appears on the first line of the
//     document.
//  2. `##` block headings appear in the SPEC §4.5 canonical order:
//     Intent, Invariants, Assumptions, Contracts, Unconstrained.
//  3. Within each block, `###` key headings appear in ascending
//     lexicographic order by slug. The heading body itself
//     (free-form prose) is preserved verbatim.
//  4. Intent is emitted as a single paragraph when its body has one
//     item, and as an unordered list when its body has more than
//     one item.
//  5. Within each Contracts entry, sub-fields appear in the order
//     **Given:**, **When:**, **Then:** regardless of authored
//     order.
//  6. Empty REQUIRED blocks (Invariants, Assumptions) are emitted
//     as a heading with no children, preserving the semantically-
//     meaningful empty form (SPEC §4.3.4).
//  7. Optional blocks (Contracts, Unconstrained) are omitted when
//     empty.
//  8. Each structural heading is preceded by exactly one blank line.
//  9. Output ends with exactly one trailing newline. No trailing
//     whitespace on any line.
//
// Two callers consume this package:
//
//   - `dx fmt` writes the canonical form back over the source.
//   - `dx export` emits the canonical form (or a JSON projection
//     of the AST) for ingestion by another agent.
//
// The canonicalizer is AST-driven, not text-driven: two
// ast.Declaration values that compare equal MUST produce
// byte-identical output.
package canonical

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/dewitt/dx/pkg/ast"
)

// Options controls canonicalizer behavior. Reserved for future
// extensibility; currently there are no per-call knobs (the v0.2.0
// canonical form is fully determined by the AST).
type Options struct {
	// StripComments has no effect in v0.2.0 because the AST does
	// not carry leaf comments. Retained for API stability with the
	// v0.1.0 toolchain shape.
	StripComments bool
}

// Marshal returns the canonical CommonMark representation of d.
//
// The returned byte slice ends with exactly one '\n' and contains
// no trailing whitespace on any line. Two ast.Declaration values
// that compare equal MUST produce byte-identical output; this is
// the property that lets `dx fmt` be idempotent.
func Marshal(d *ast.Declaration, opts Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("canonical.Marshal: nil declaration")
	}
	var buf bytes.Buffer

	// `# system` heading on line 1.
	buf.WriteString("# ")
	buf.WriteString(d.System)
	buf.WriteByte('\n')

	// `## Intent` (REQUIRED).
	writeBlockHeader(&buf, "Intent")
	writeIntent(&buf, d.Intent)

	// `## Invariants` (REQUIRED, may be empty).
	writeBlockHeader(&buf, "Invariants")
	writeKeyedBlock(&buf, d.Invariants)

	// `## Assumptions` (REQUIRED, may be empty).
	writeBlockHeader(&buf, "Assumptions")
	writeKeyedBlock(&buf, d.Assumptions)

	// `## Contracts` (OPTIONAL).
	if len(d.Contracts) > 0 {
		writeBlockHeader(&buf, "Contracts")
		writeContractsBlock(&buf, d.Contracts)
	}

	// `## Unconstrained` (OPTIONAL).
	if len(d.Unconstrained) > 0 {
		writeBlockHeader(&buf, "Unconstrained")
		writeKeyedBlock(&buf, d.Unconstrained)
	}

	return scrubTrailingWhitespace(buf.Bytes()), nil
}

// writeBlockHeader writes "\n## <name>\n" -- a blank line plus the
// level-2 heading.
func writeBlockHeader(buf *bytes.Buffer, name string) {
	buf.WriteString("\n## ")
	buf.WriteString(name)
	buf.WriteByte('\n')
}

// writeIntent emits the `## Intent` body. A single-element intent
// is emitted as one paragraph; a multi-element intent is emitted
// as an unordered list (in the slice's order, which is the
// architect's priority order per SPEC §4.3.2).
//
// An empty intent (zero items) emits nothing additional, leaving
// the heading as the only line. The lint pass catches the missing
// intent body.
func writeIntent(buf *bytes.Buffer, intent []string) {
	if len(intent) == 0 {
		return
	}
	if len(intent) == 1 {
		buf.WriteByte('\n')
		buf.WriteString(strings.TrimSpace(intent[0]))
		buf.WriteByte('\n')
		return
	}
	buf.WriteByte('\n')
	for _, item := range intent {
		buf.WriteString("- ")
		buf.WriteString(strings.TrimSpace(item))
		buf.WriteByte('\n')
	}
}

// writeKeyedBlock emits a block whose entries are simple
// heading-plus-body pairs (Invariants, Assumptions, Unconstrained).
// Entries are sorted by slug (the map key) for stable diff output;
// each entry's verbatim heading is written under the ### marker.
func writeKeyedBlock(buf *bytes.Buffer, m map[string]ast.Entry) {
	if len(m) == 0 {
		// SPEC §4.3.4: an empty REQUIRED block is encoded as the
		// heading with no `###` children. The heading is already
		// written; nothing more to emit.
		return
	}
	slugs := make([]string, 0, len(m))
	for k := range m {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		entry := m[slug]
		buf.WriteString("\n### ")
		buf.WriteString(entry.Heading)
		buf.WriteByte('\n')
		body := strings.TrimSpace(entry.Body)
		if body != "" {
			buf.WriteByte('\n')
			buf.WriteString(body)
			buf.WriteByte('\n')
		}
	}
}

// writeContractsBlock emits the Contracts block. Each contract is a
// `###` section containing the three sub-fields in fixed order:
// **Given:**, **When:**, **Then:**.
func writeContractsBlock(buf *bytes.Buffer, m map[string]ast.Contract) {
	slugs := make([]string, 0, len(m))
	for k := range m {
		slugs = append(slugs, k)
	}
	sort.Strings(slugs)
	for _, slug := range slugs {
		c := m[slug]
		buf.WriteString("\n### ")
		buf.WriteString(c.Heading)
		buf.WriteByte('\n')
		writeContractSubfield(buf, "Given", c.Given)
		writeContractSubfield(buf, "When", c.When)
		writeContractSubfield(buf, "Then", c.Then)
	}
}

// writeContractSubfield emits "\n**<Label>:** <body>\n" for
// single-paragraph bodies and a multi-paragraph form when the body
// contains a blank line.
func writeContractSubfield(buf *bytes.Buffer, label, body string) {
	body = strings.TrimSpace(body)
	buf.WriteString("\n**")
	buf.WriteString(label)
	buf.WriteString(":**")
	if body == "" {
		buf.WriteByte('\n')
		return
	}
	paragraphs := splitParagraphs(body)
	if len(paragraphs) == 0 {
		buf.WriteByte('\n')
		return
	}
	// First paragraph: inline with the label.
	buf.WriteByte(' ')
	buf.WriteString(paragraphs[0])
	buf.WriteByte('\n')
	// Subsequent paragraphs.
	for _, p := range paragraphs[1:] {
		buf.WriteByte('\n')
		buf.WriteString(p)
		buf.WriteByte('\n')
	}
}

// splitParagraphs splits s into paragraph strings by blank-line
// separators. Each returned paragraph has no leading or trailing
// newline. (Mirrors the helper in pkg/lint but kept local to avoid
// a cyclic import.)
func splitParagraphs(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	var paras []string
	var buf strings.Builder
	flush := func() {
		if buf.Len() > 0 {
			paras = append(paras, buf.String())
			buf.Reset()
		}
	}
	for _, line := range lines {
		trim := strings.TrimRight(line, " \t\r")
		if trim == "" {
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

// scrubTrailingWhitespace removes trailing spaces and tabs from
// every line and ensures the output ends with exactly one newline.
func scrubTrailingWhitespace(b []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(b))
	lines := bytes.Split(b, []byte("\n"))
	for i, line := range lines {
		end := len(line)
		for end > 0 && (line[end-1] == ' ' || line[end-1] == '\t') {
			end--
		}
		out.Write(line[:end])
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	trimmed := bytes.TrimRight(out.Bytes(), "\n")
	result := make([]byte, 0, len(trimmed)+1)
	result = append(result, trimmed...)
	result = append(result, '\n')
	return result
}
