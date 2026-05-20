// Package export emits a parsed declaration in agent-optimized
// formats.
//
// Two formats ship today:
//
//   - FormatMarkdown (default) — canonical CommonMark per
//     pkg/canonical. The form a fresh agent should consume:
//     byte-stable for the same AST so two agents can agree on
//     hashes, idempotent under repeated export.
//
//   - FormatJSON — compact one-line JSON projection of the AST.
//     The form to feed to non-LLM consumers (other tools, sub-agents
//     that prefer structured input). Map iteration order is
//     stabilized via canonical key sorting so output is also
//     byte-stable.
//
// The canonical CommonMark form is what `dx fmt` writes back over
// the source as well; export and fmt produce the same bytes for the
// same AST.
package export

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/dewitt/dx/pkg/ast"
	"github.com/dewitt/dx/pkg/canonical"
)

// Format names a target serialization for `dx export`.
type Format string

const (
	// FormatMarkdown emits canonical CommonMark.
	FormatMarkdown Format = "markdown"
	// FormatJSON emits a compact JSON projection of the AST.
	FormatJSON Format = "json"
)

// Write serializes d in the requested format and writes the result
// to w. The output ends with a single newline regardless of format.
func Write(w io.Writer, d *ast.Declaration, format Format) error {
	if d == nil {
		return fmt.Errorf("export: nil declaration")
	}

	switch format {
	case "", FormatMarkdown:
		out, err := canonical.Marshal(d, canonical.Options{
			StripComments: true,
		})
		if err != nil {
			return err
		}
		_, err = w.Write(out)
		return err

	case FormatJSON:
		// We marshal a stable, ordered projection by hand rather
		// than relying on encoding/json's map iteration (which is
		// nondeterministic across runs of the same process).
		payload := projectForJSON(d)
		// We bypass json.Marshal in favor of an Encoder configured
		// with SetEscapeHTML(false): the export consumer is an
		// agent or tool, never an HTML page, and the default
		// `<` -> `\u003c` escaping is just noise that bloats
		// tokens. Encoder.Encode appends a trailing newline for us.
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(payload); err != nil {
			return fmt.Errorf("export: json marshal: %w", err)
		}
		_, err := w.Write(buf.Bytes())
		return err

	default:
		return fmt.Errorf("export: unknown format %q (want one of: markdown, json)", format)
	}
}

// entryJSON is the JSON projection of an ast.Entry. Each entry
// carries both its derived slug (the map key) and its original
// heading text (the "heading" field), so a downstream consumer can
// present whichever it prefers.
type entryJSON struct {
	Heading string `json:"heading,omitempty"`
	Body    string `json:"body,omitempty"`
}

type contractJSON struct {
	Heading string `json:"heading,omitempty"`
	Given   string `json:"given,omitempty"`
	When    string `json:"when,omitempty"`
	Then    string `json:"then,omitempty"`
}

// projectForJSON builds a deterministic JSON-ready projection of d.
//
// The shape mirrors the declaration as closely as JSON allows:
//
//	{
//	  "system": "...",
//	  "intent": [ "first intent", "second intent", ... ],
//	  "invariants": {
//	    "<slug>": { "heading": "<prose>", "body": "<prose>" }, ...
//	  },
//	  "assumptions": { ... same shape ... },
//	  "contracts": {
//	    "<slug>": { "heading": "...", "given": "...", "when": "...", "then": "..." },
//	    ...
//	  },
//	  "unconstrained": { ... same shape as invariants ... }
//	}
//
// Map values are written via map[string]<struct> which encoding/json
// sorts by key alphabetically. We rely on that behavior (Go's
// encoding/json does sort map keys), which is stable since Go 1.12.
func projectForJSON(d *ast.Declaration) any {
	type rootJSON struct {
		System        string                  `json:"system,omitempty"`
		Intent        []string                `json:"intent,omitempty"`
		Invariants    map[string]entryJSON    `json:"invariants"`
		Assumptions   map[string]entryJSON    `json:"assumptions"`
		Contracts     map[string]contractJSON `json:"contracts,omitempty"`
		Unconstrained map[string]entryJSON    `json:"unconstrained,omitempty"`
	}

	root := rootJSON{
		System:      d.System,
		Intent:      d.Intent,
		Invariants:  projectEntries(d.Invariants),
		Assumptions: projectEntries(d.Assumptions),
	}

	if len(d.Contracts) > 0 {
		root.Contracts = make(map[string]contractJSON, len(d.Contracts))
		names := make([]string, 0, len(d.Contracts))
		for k := range d.Contracts {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, n := range names {
			c := d.Contracts[n]
			root.Contracts[n] = contractJSON{
				Heading: c.Heading,
				Given:   c.Given,
				When:    c.When,
				Then:    c.Then,
			}
		}
	}

	if len(d.Unconstrained) > 0 {
		root.Unconstrained = projectEntries(d.Unconstrained)
	}

	return root
}

// projectEntries converts an ast.Entry map into the JSON
// projection form. The result is always a non-nil map (possibly
// empty) so the REQUIRED `invariants` and `assumptions` keys
// always render as `{}` rather than `null` -- preserving the
// SPEC §4.3 distinction between "empty" and "absent."
func projectEntries(m map[string]ast.Entry) map[string]entryJSON {
	out := make(map[string]entryJSON, len(m))
	for k, v := range m {
		out[k] = entryJSON{Heading: v.Heading, Body: v.Body}
	}
	return out
}
