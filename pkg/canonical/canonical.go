// Package canonical produces a deterministic, byte-stable
// representation of a parsed `.dx` declaration.
//
// The canonical form has these properties:
//
//  1. Top-level keys appear in the SPEC §2 canonical order:
//     system, intent, invariants, assumptions, contracts, unconstrained.
//  2. Map entries inside invariants, assumptions, contracts, and
//     unconstrained are sorted alphabetically by key.
//  3. List values (currently only intent.secondary) preserve their
//     authored order -- list ordering is semantic in `.dx`.
//  4. Multi-line strings are emitted as literal block scalars (`|`),
//     never folded (`>`); single-line strings use plain or
//     double-quoted form per yaml.v3's defaults.
//  5. Output ends with exactly one trailing newline. No trailing
//     whitespace on any line.
//
// Two callers consume this package:
//
//   - `dx fmt` writes the canonical form back over the source,
//     preserving comments where the user authored them.
//   - `dx export` emits the canonical form (or a JSON projection
//     of the AST) for ingestion by another agent, stripping comments.
//
// The canonicalizer is deliberately AST-driven, not text-driven: we
// rebuild the YAML node graph from the decoded ast.Declaration rather
// than mutating the input nodes. This guarantees that any `.dx` input
// that decodes to the same AST produces byte-identical output, which
// is the property that makes `dx fmt` idempotent and makes
// `dx export` hashable.
package canonical

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/dewitt/dx/pkg/ast"
)

// Options controls canonicalizer behavior.
type Options struct {
	// StripComments drops all head/line/foot comments from the
	// emitted output. `dx export` sets this to true; `dx fmt`
	// sets it to false.
	StripComments bool

	// SourceNode, if non-nil, is the original *yaml.Node graph from
	// which d was decoded. When provided and StripComments is false,
	// the canonicalizer copies head comments from matching top-level
	// keys onto the rebuilt graph so `dx fmt` round-trips them.
	// Comments on entries inside invariants/assumptions/contracts/
	// unconstrained are NOT preserved across formatting in this
	// release -- doing so requires content-keyed identity, which is
	// brittle. (See "Known limitations" in fmt's skill section.)
	SourceNode *yaml.Node
}

// Marshal returns the canonical YAML representation of d.
//
// The returned byte slice ends with exactly one '\n' and contains no
// trailing whitespace on any line. Two ast.Declaration values that
// compare equal MUST produce byte-identical output; this is the
// property that lets `dx fmt` be idempotent.
func Marshal(d *ast.Declaration, opts Options) ([]byte, error) {
	if d == nil {
		return nil, fmt.Errorf("canonical.Marshal: nil declaration")
	}

	root := buildRoot(d, opts)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(root); err != nil {
		return nil, fmt.Errorf("canonical.Marshal: encode: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("canonical.Marshal: close: %w", err)
	}

	return scrubTrailingWhitespace(buf.Bytes()), nil
}

// buildRoot constructs a fresh document node containing the top-level
// mapping in SPEC §2 canonical order. Optional blocks are emitted only
// when they have content.
func buildRoot(d *ast.Declaration, opts Options) *yaml.Node {
	doc := &yaml.Node{Kind: yaml.DocumentNode}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	doc.Content = []*yaml.Node{root}

	headComments := topLevelHeadComments(opts.SourceNode, opts.StripComments)

	addPair := func(key string, value *yaml.Node) {
		k := scalarString(key, false)
		if hc, ok := headComments[key]; ok {
			k.HeadComment = hc
		}
		root.Content = append(root.Content, k, value)
	}

	// 1. system (required).
	if d.System != "" {
		addPair("system", scalarString(d.System, false))
	}

	// 2. intent (required). Build sub-mapping by hand to enforce
	// primary-then-secondary ordering.
	if d.Intent.Primary != "" || len(d.Intent.Secondary) > 0 {
		intent := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if d.Intent.Primary != "" {
			intent.Content = append(intent.Content,
				scalarString("primary", false),
				scalarString(d.Intent.Primary, true),
			)
		}
		if len(d.Intent.Secondary) > 0 {
			seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			for _, s := range d.Intent.Secondary {
				seq.Content = append(seq.Content, scalarString(s, true))
			}
			intent.Content = append(intent.Content,
				scalarString("secondary", false),
				seq,
			)
		}
		addPair("intent", intent)
	}

	// 3. invariants (required, may be empty map).
	addPair("invariants", sortedStringMap(d.Invariants))

	// 4. assumptions (required, may be empty map).
	addPair("assumptions", sortedStringMap(d.Assumptions))

	// 5. contracts (optional).
	if len(d.Contracts) > 0 {
		addPair("contracts", sortedContractMap(d.Contracts))
	}

	// 6. unconstrained (optional).
	if len(d.Unconstrained) > 0 {
		addPair("unconstrained", sortedStringMap(d.Unconstrained))
	}

	return doc
}

// sortedStringMap returns a mapping node with keys sorted
// alphabetically. Empty input produces a flow-style empty map ({}) so
// the SPEC §3 zero-state is preserved verbatim.
func sortedStringMap(m map[string]string) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if len(m) == 0 {
		n.Style = yaml.FlowStyle
		return n
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		n.Content = append(n.Content,
			scalarString(k, false),
			scalarString(m[k], true),
		)
	}
	return n
}

// sortedContractMap orders contracts by name, and within each
// contract emits given/when/then in that fixed order regardless of
// authored order. (The struct order in ast.Contract is incidental;
// the wire order is normative.)
func sortedContractMap(m map[string]ast.Contract) *yaml.Node {
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if len(m) == 0 {
		n.Style = yaml.FlowStyle
		return n
	}
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, name := range names {
		c := m[name]
		body := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		if c.Given != "" {
			body.Content = append(body.Content,
				scalarString("given", false),
				scalarString(c.Given, true),
			)
		}
		if c.When != "" {
			body.Content = append(body.Content,
				scalarString("when", false),
				scalarString(c.When, true),
			)
		}
		if c.Then != "" {
			body.Content = append(body.Content,
				scalarString("then", false),
				scalarString(c.Then, true),
			)
		}
		n.Content = append(n.Content, scalarString(name, false), body)
	}
	return n
}

// scalarString returns a scalar node carrying value.
//
// When preferLiteralIfMultiline is true and the value contains an
// *internal* newline, the node is emitted as a literal block scalar
// (`|`). A *trailing* newline alone does not count as multi-line:
// it is the artifact YAML produces when a literal block is decoded
// (e.g., `key: |\n  hello\n` decodes to "hello\n"), and round-
// tripping a single-line string through `|` would otherwise
// permanently lock it into block-scalar form even when the content
// is one line. We strip a single trailing newline before checking,
// then hand the (possibly trimmed) value to yaml.v3, which picks
// plain or double-quoted style automatically.
//
// When preferLiteralIfMultiline is false (key strings, slug values,
// etc.), the value is passed through unchanged and yaml.v3 picks
// the style.
func scalarString(value string, preferLiteralIfMultiline bool) *yaml.Node {
	if !preferLiteralIfMultiline {
		return &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	}

	// Treat at most one trailing newline as a YAML emit artifact, not
	// semantic content. Anything beyond that -- internal newlines or
	// multiple trailing newlines -- is preserved as authored.
	body := value
	if strings.HasSuffix(body, "\n") {
		body = body[:len(body)-1]
	}

	n := &yaml.Node{Kind: yaml.ScalarNode, Value: value}
	if strings.ContainsRune(body, '\n') {
		n.Style = yaml.LiteralStyle
		return n
	}

	// Single-line content (with at most a trailing newline). Emit the
	// content without the trailing newline so yaml.v3 picks plain (or
	// quoted) form, not `|`. The semantic round-trip remains clean
	// because a plain scalar decodes back to a string with no
	// trailing newline -- the artifact is gone in both directions.
	n.Value = body
	return n
}

// topLevelHeadComments builds a map from top-level key name to the
// head comment that preceded that key in the source. When stripping
// comments or when no source node is available, the result is an
// empty map.
func topLevelHeadComments(src *yaml.Node, strip bool) map[string]string {
	out := make(map[string]string)
	if strip || src == nil || len(src.Content) == 0 {
		return out
	}
	doc := src.Content[0]
	if doc.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i+1 < len(doc.Content); i += 2 {
		key := doc.Content[i]
		if hc := key.HeadComment; hc != "" {
			out[key.Value] = hc
		}
	}
	return out
}

// scrubTrailingWhitespace removes spaces and tabs immediately before
// any line terminator and ensures the output ends with exactly one
// newline. yaml.v3's encoder is generally well-behaved, but literal
// block scalars whose body has trailing spaces can leak them through;
// this pass is defense in depth.
func scrubTrailingWhitespace(b []byte) []byte {
	// Trim trailing whitespace per line.
	var out bytes.Buffer
	out.Grow(len(b))
	lines := bytes.Split(b, []byte("\n"))
	for i, line := range lines {
		// Strip trailing spaces and tabs.
		end := len(line)
		for end > 0 && (line[end-1] == ' ' || line[end-1] == '\t') {
			end--
		}
		out.Write(line[:end])
		if i < len(lines)-1 {
			out.WriteByte('\n')
		}
	}
	// Normalize trailing newlines: collapse any run of blank lines at
	// the end to exactly one '\n'.
	trimmed := bytes.TrimRight(out.Bytes(), "\n")
	result := make([]byte, 0, len(trimmed)+1)
	result = append(result, trimmed...)
	result = append(result, '\n')
	return result
}
