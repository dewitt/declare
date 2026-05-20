package lint

// Markdown parser/loader for v0.2.0 declarations (SPEC §4).
//
// The parser uses goldmark to identify the *positions* of ATX headings
// (so we don't mistake a `#` inside a fenced code block for a heading)
// and otherwise treats the byte range between consecutive structural
// headings as opaque leaf content per SPEC §4.2 ("Leaf layer:
// Opaque CommonMark").
//
// The output of this pass is a Document: a sequence of headings (with
// level, body text, source line, and source byte range) plus the
// source bytes themselves. Higher-level passes in lint.go consume the
// Document to build the AST and report structural issues.

import (
	"bytes"
	"strings"

	gmast "github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"

	"github.com/yuin/goldmark"
)

// headingNode records one ATX heading found in the source.
type headingNode struct {
	level    int    // 1..6
	text     string // trimmed heading body
	line     int    // 1-based source line
	startOff int    // byte offset of '#' character
	endOff   int    // byte offset one past the final newline of the heading line
}

// document is the intermediate representation produced by the parser.
type document struct {
	source   []byte
	headings []headingNode
}

// parseSource runs goldmark on data and returns a document describing
// the ATX headings it found. The returned document is later consumed
// to build the AST and to extract leaf content.
//
// Parsing never fails: invalid CommonMark is still some CommonMark.
// Structural problems are surfaced by higher-level passes as Issues.
func parseSource(data []byte) *document {
	src := data
	reader := gmtext.NewReader(src)
	md := goldmark.New()
	root := md.Parser().Parse(reader)

	doc := &document{source: src}

	// Walk only the top-level block children of the document.
	// CommonMark guarantees that an ATX heading is a top-level
	// block (or a child of a block-quote / list-item, neither of
	// which are part of our structural grammar). Restricting to
	// the document's direct children means we automatically ignore
	// '#' lines that appear inside fenced code blocks, indented
	// code blocks, block quotes, or list items.
	for child := root.FirstChild(); child != nil; child = child.NextSibling() {
		h, ok := child.(*gmast.Heading)
		if !ok {
			continue
		}
		hn := newHeadingNode(src, h)
		doc.headings = append(doc.headings, hn)
	}

	return doc
}

// newHeadingNode extracts the heading body text and source position
// from a goldmark Heading node.
//
// goldmark's heading.Lines() contains the segments of the heading's
// inline content; the leading '#' characters and the single mandatory
// space after them are *not* part of those segments. We use the first
// segment's start to locate the source line and the segment range to
// extract the body text verbatim.
func newHeadingNode(src []byte, h *gmast.Heading) headingNode {
	lines := h.Lines()
	if lines.Len() == 0 {
		// Empty heading body (e.g. `## `). Locate the heading line
		// by scanning back for the '#' that introduced it. goldmark
		// guarantees that a Heading with no Lines was still
		// recognized as an ATX heading; the source position of
		// the '#' is one we synthesize by walking from offset 0.
		// In practice this branch is rare; we fall back to a
		// best-effort line number.
		return headingNode{
			level: h.Level,
			text:  "",
			line:  1,
		}
	}

	first := lines.At(0)
	// Body extraction: take the bytes of the (single) content
	// segment and trim ASCII whitespace. ATX heading bodies are
	// always single-line per CommonMark §4.2.
	body := strings.TrimSpace(string(src[first.Start:first.Stop]))

	// Source line: count newlines from offset 0 up to the start
	// of the heading-line. The body's segment starts after `#`s
	// and the mandatory space, so we walk back to find the
	// beginning of the line.
	lineStart := first.Start
	for lineStart > 0 && src[lineStart-1] != '\n' {
		lineStart--
	}
	line := 1 + bytes.Count(src[:lineStart], []byte("\n"))

	// End offset: one past the next newline (or end-of-source).
	endOff := first.Stop
	for endOff < len(src) && src[endOff] != '\n' {
		endOff++
	}
	if endOff < len(src) {
		endOff++ // include the newline
	}

	return headingNode{
		level:    h.Level,
		text:     body,
		line:     line,
		startOff: lineStart,
		endOff:   endOff,
	}
}

// leafBetween returns the source bytes between the end of heading
// headings[i] and the start of heading headings[i+1] (or end of
// source if i is the last heading). Leading and trailing blank lines
// are trimmed.
//
// The returned string is the verbatim leaf content per SPEC §4.2;
// it is not further interpreted as CommonMark by the loader.
func (d *document) leafBetween(i int) string {
	if i < 0 || i >= len(d.headings) {
		return ""
	}
	start := d.headings[i].endOff
	end := len(d.source)
	if i+1 < len(d.headings) {
		end = d.headings[i+1].startOff
	}
	return trimBlankLines(string(d.source[start:end]))
}

// trimBlankLines removes leading and trailing blank lines from s but
// preserves indentation and internal blank lines verbatim. A line is
// "blank" if it contains only spaces and tabs (CommonMark's definition).
func trimBlankLines(s string) string {
	// Trim leading blank lines.
	for {
		nl := strings.IndexByte(s, '\n')
		if nl < 0 {
			break
		}
		if !isBlankLine(s[:nl]) {
			break
		}
		s = s[nl+1:]
	}
	// Trim trailing blank lines. The final "line" may have no
	// trailing newline.
	for len(s) > 0 {
		// Find the last newline; the content after it is the
		// final line.
		lastNL := strings.LastIndexByte(s, '\n')
		if lastNL < 0 {
			// Single line; if blank, drop it entirely.
			if isBlankLine(s) {
				s = ""
			}
			break
		}
		final := s[lastNL+1:]
		if !isBlankLine(final) {
			// Last line is non-blank. Strip any trailing newline.
			if strings.HasSuffix(s, "\n") {
				s = s[:len(s)-1]
			}
			// Now also remove blank lines that sit between the
			// final non-blank line and the newline we just removed.
			for {
				lastNL = strings.LastIndexByte(s, '\n')
				if lastNL < 0 {
					break
				}
				if !isBlankLine(s[lastNL+1:]) {
					break
				}
				s = s[:lastNL]
			}
			break
		}
		// Trim the blank final line plus its preceding newline.
		s = s[:lastNL]
	}
	return s
}

func isBlankLine(line string) bool {
	for i := 0; i < len(line); i++ {
		c := line[i]
		if c != ' ' && c != '\t' && c != '\r' {
			return false
		}
	}
	return true
}
