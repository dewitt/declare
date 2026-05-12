// Package contracts provides operations over the `contracts:` block
// of a parsed `.dx` declaration.
//
// Today this is a thin enumeration layer for `declare contracts list`:
// it turns the unordered map[string]ast.Contract on the AST into a
// stable, alphabetically-ordered slice of (name, contract) pairs and
// renders that slice in either a shell-friendly text form or a
// structured JSON form.
//
// As `declare verify` lands in v0.2 this package is the natural home
// for a contract execution plan, parameter substitution, and result
// reporting -- but those concerns are deliberately out of scope here.
package contracts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/dewitt/declare/pkg/ast"
)

// Entry is one (name, contract) pair drawn from a Declaration's
// contracts: block. Slices of Entry are always sorted alphabetically
// by Name -- that's the determinism callers rely on for shell scripting
// and for hash-based agreement between agents.
type Entry struct {
	Name     string
	Contract ast.Contract
}

// List returns every contract in d as a sorted slice of Entry. A
// declaration with no contracts: block (or with the block present
// but empty) returns an empty slice, never nil-with-distinction.
func List(d *ast.Declaration) []Entry {
	if d == nil || len(d.Contracts) == 0 {
		return []Entry{}
	}
	names := make([]string, 0, len(d.Contracts))
	for n := range d.Contracts {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Entry, 0, len(names))
	for _, n := range names {
		out = append(out, Entry{Name: n, Contract: d.Contracts[n]})
	}
	return out
}

// Format selects an output serialization for `declare contracts list`.
type Format string

const (
	// FormatText emits one contract per line. With Verbose=true,
	// each contract is followed by indented `given:` / `when:` /
	// `then:` clauses (truncated to one line with an ellipsis if
	// the body spans multiple lines).
	FormatText Format = "text"
	// FormatJSON emits a single JSON object: {"contracts":[...]} with
	// each entry carrying name + given + when + then.
	FormatJSON Format = "json"
)

// WriteOptions controls how WriteList renders entries.
type WriteOptions struct {
	Format  Format
	Verbose bool // affects FormatText only; JSON always includes full bodies
}

// WriteList serializes entries according to opts and writes the
// result to w.
//
// Trailing-newline conventions differ per format:
//
//   - FormatText writes one line per entry; empty input writes
//     nothing at all (no trailing newline) so that piping into
//     `wc -l` or `xargs` produces zero rather than one.
//   - FormatJSON always writes a single object {"contracts":[...]}
//     followed by one newline, even when the array is empty. Empty
//     JSON output ({"contracts":[]}\n) is a positive signal that
//     the spec has zero contracts, distinct from a parse failure.
func WriteList(w io.Writer, entries []Entry, opts WriteOptions) error {
	switch opts.Format {
	case "", FormatText:
		return writeText(w, entries, opts.Verbose)
	case FormatJSON:
		return writeJSON(w, entries)
	default:
		return fmt.Errorf("contracts: unknown format %q (want one of: text, json)", opts.Format)
	}
}

func writeText(w io.Writer, entries []Entry, verbose bool) error {
	var buf bytes.Buffer
	for _, e := range entries {
		buf.WriteString(e.Name)
		buf.WriteByte('\n')
		if verbose {
			writeClause(&buf, "given", e.Contract.Given)
			writeClause(&buf, "when", e.Contract.When)
			writeClause(&buf, "then", e.Contract.Then)
		}
	}
	_, err := w.Write(buf.Bytes())
	return err
}

// writeClause emits one indented `key: value` line under a verbose
// entry. Multi-line bodies are truncated to the first non-empty line
// with a trailing " …" so the per-contract block stays exactly four
// lines tall (name + three clauses). Use FormatJSON for full bodies.
func writeClause(buf *bytes.Buffer, key, body string) {
	preview := firstNonEmptyLine(body)
	truncated := false
	for _, ln := range strings.Split(body, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || ln == preview {
			continue
		}
		truncated = true
		break
	}
	fmt.Fprintf(buf, "  %-5s %s", key+":", preview)
	if truncated {
		buf.WriteString(" …")
	}
	buf.WriteByte('\n')
}

func firstNonEmptyLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" {
			return ln
		}
	}
	return ""
}

func writeJSON(w io.Writer, entries []Entry) error {
	type jsonContract struct {
		Name  string `json:"name"`
		Given string `json:"given,omitempty"`
		When  string `json:"when,omitempty"`
		Then  string `json:"then,omitempty"`
	}
	type payload struct {
		Contracts []jsonContract `json:"contracts"`
	}
	p := payload{Contracts: make([]jsonContract, 0, len(entries))}
	for _, e := range entries {
		p.Contracts = append(p.Contracts, jsonContract{
			Name:  e.Name,
			Given: e.Contract.Given,
			When:  e.Contract.When,
			Then:  e.Contract.Then,
		})
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(p); err != nil {
		return fmt.Errorf("contracts: json marshal: %w", err)
	}
	_, err := w.Write(buf.Bytes())
	return err
}
