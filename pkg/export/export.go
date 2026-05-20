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

// projectForJSON builds a deterministic JSON-ready projection of d.
// We use ordered slices for the top-level structure (so block order
// is stable) and sorted map projections for invariants/assumptions/
// contracts/unconstrained (so key order is stable). Empty optional
// blocks are omitted.
//
// The shape mirrors the .dx file as closely as JSON allows:
//
//	{
//	  "system": "...",
//	  "intent": { "primary": "...", "secondary": [...] },
//	  "invariants": { "id": "body", ... },
//	  "assumptions": { "id": "body", ... },
//	  "contracts": { "name": { "given": "...", "when": "...", "then": "..." } },
//	  "unconstrained": { "category": "description", ... }
//	}
//
// Map values are written via map[string]string / map[string]contract
// which in encoding/json sort keys alphabetically. We rely on that
// behavior (Go's encoding/json does sort map keys), which is stable
// since Go 1.12.
func projectForJSON(d *ast.Declaration) any {
	type intentJSON struct {
		Primary   string   `json:"primary,omitempty"`
		Secondary []string `json:"secondary,omitempty"`
	}
	type contractJSON struct {
		Given string `json:"given,omitempty"`
		When  string `json:"when,omitempty"`
		Then  string `json:"then,omitempty"`
	}

	// Use an ordered slice of [key, value] pairs at the top level so
	// the SPEC §4.2 block order is preserved. encoding/json doesn't
	// natively support ordered objects; the standard workaround is
	// json.RawMessage assembly, but for a fixed six-key schema a
	// custom marshaler is overkill. We rely on the fact that
	// encoding/json marshals struct fields in declaration order.

	type rootJSON struct {
		System        string                  `json:"system,omitempty"`
		Intent        *intentJSON             `json:"intent,omitempty"`
		Invariants    map[string]string       `json:"invariants"`
		Assumptions   map[string]string       `json:"assumptions"`
		Contracts     map[string]contractJSON `json:"contracts,omitempty"`
		Unconstrained map[string]string       `json:"unconstrained,omitempty"`
	}

	root := rootJSON{
		System:      d.System,
		Invariants:  ensureMap(d.Invariants),
		Assumptions: ensureMap(d.Assumptions),
	}

	if d.Intent.Primary != "" || len(d.Intent.Secondary) > 0 {
		root.Intent = &intentJSON{
			Primary:   d.Intent.Primary,
			Secondary: d.Intent.Secondary,
		}
	}

	if len(d.Contracts) > 0 {
		root.Contracts = make(map[string]contractJSON, len(d.Contracts))
		// Sorting here is belt-and-suspenders; encoding/json sorts
		// map keys, but doing it explicitly documents the intent.
		names := make([]string, 0, len(d.Contracts))
		for k := range d.Contracts {
			names = append(names, k)
		}
		sort.Strings(names)
		for _, n := range names {
			c := d.Contracts[n]
			root.Contracts[n] = contractJSON{
				Given: c.Given,
				When:  c.When,
				Then:  c.Then,
			}
		}
	}

	if len(d.Unconstrained) > 0 {
		root.Unconstrained = ensureMap(d.Unconstrained)
	}

	return root
}

// ensureMap returns m if non-nil, or an empty map. We materialize
// invariants and assumptions as `{}` rather than `null` in JSON
// because SPEC §4.3 treats the empty form as semantically distinct
// from "absent."
func ensureMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}
