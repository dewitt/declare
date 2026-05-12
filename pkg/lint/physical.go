package lint

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Physical-rule walkers for SPEC §2.
//
// SPEC §2 forbids several YAML features that are valid YAML 1.2 but
// degrade the determinism of the AST or the parsing reliability of LLM
// tokenizers. The strict decoder we run in lint.go cannot catch them --
// they are properties of the source representation, not of the decoded
// values -- so we walk the retained *yaml.Node graph here.
//
// Each helper appends Issues with positional information so an editor
// or CI can surface the exact offending line and column.

// validatePhysical walks root and returns one Issue per SPEC §2 violation.
// It is safe to call before strict-decoding succeeds; in fact, anchors
// and aliases must be flagged here because the decoder would silently
// follow them and the structural error messages would be confusing.
func validatePhysical(path string, root *yaml.Node) []Issue {
	if root == nil {
		return nil
	}
	v := &physicalVisitor{path: path}
	v.walk(root)
	return v.issues
}

type physicalVisitor struct {
	path   string
	issues []Issue
}

func (v *physicalVisitor) walk(n *yaml.Node) {
	if n == nil {
		return
	}

	switch n.Kind {
	case yaml.AliasNode:
		// SPEC §2: "The use of `&` (anchors) and `*` (aliases) is
		// strictly forbidden."
		v.add(n, "alias node forbidden by SPEC §2 (no anchors/aliases)")
		// Do not recurse into n.Alias: the target lives elsewhere in
		// the tree and will be visited there if it is reachable.
		return

	case yaml.ScalarNode:
		v.checkAnchor(n)
		v.checkTag(n)
		v.checkScalarStyle(n)
		return

	case yaml.MappingNode, yaml.SequenceNode, yaml.DocumentNode:
		v.checkAnchor(n)
		v.checkContainerTag(n)
		for _, c := range n.Content {
			v.walk(c)
		}
		return
	}
}

func (v *physicalVisitor) checkAnchor(n *yaml.Node) {
	if n.Anchor != "" {
		// SPEC §2: anchors are forbidden because they introduce
		// hidden state that breaks the LLM's local reasoning over the
		// document.
		v.add(n, fmt.Sprintf(
			"anchor `&%s` forbidden by SPEC §2 (no anchors/aliases)",
			n.Anchor,
		))
	}
}

// checkTag rejects any explicit, non-default tag on a scalar.
//
// yaml.v3 always populates Tag with the *resolved* tag after decoding,
// even when the source had no tag at all. We therefore allow the set of
// implicit tags YAML 1.2 produces during default resolution and reject
// everything else.
func (v *physicalVisitor) checkTag(n *yaml.Node) {
	if isAllowedScalarTag(n.Tag) {
		return
	}
	v.add(n, fmt.Sprintf(
		"explicit YAML tag %q forbidden by SPEC §2 (no custom tags)",
		n.Tag,
	))
}

// checkContainerTag mirrors checkTag for mapping/sequence nodes. We
// allow the implicit container tags (`!!map`, `!!seq`) that yaml.v3
// resolves during decoding; anything else is a custom tag.
func (v *physicalVisitor) checkContainerTag(n *yaml.Node) {
	if isAllowedContainerTag(n.Tag) {
		return
	}
	v.add(n, fmt.Sprintf(
		"explicit YAML tag %q forbidden by SPEC §2 (no custom tags)",
		n.Tag,
	))
}

// isAllowedScalarTag mirrors the set of implicit tags YAML 1.2 produces
// during default scalar resolution. SPEC §2 forbids any *custom* tag --
// e.g., `!!binary`, `!!set`, or any user-defined `!foo` -- but allows
// the implicit core-schema tags that yaml.v3 will synthesize even when
// the source had no tag at all.
func isAllowedScalarTag(tag string) bool {
	switch tag {
	case "",
		"!!str",
		"!!int",
		"!!float",
		"!!bool",
		"!!null",
		"!!timestamp":
		return true
	}
	return false
}

func isAllowedContainerTag(tag string) bool {
	switch tag {
	case "", "!!map", "!!seq":
		return true
	}
	return false
}

// checkScalarStyle rejects folded block scalars (`>`).
//
// SPEC §2: "All multiline strings must use the literal block scalar (|).
// The folded scalar (>) is prohibited due to ambiguous whitespace
// handling in diverse LLM tokenizers."
func (v *physicalVisitor) checkScalarStyle(n *yaml.Node) {
	if n.Style&yaml.FoldedStyle != 0 {
		v.add(n, "folded block scalar `>` forbidden by SPEC §2 (use literal `|`)")
	}
}

func (v *physicalVisitor) add(n *yaml.Node, msg string) {
	v.issues = append(v.issues, Issue{
		Path:    v.path,
		Line:    n.Line,
		Column:  n.Column,
		Message: msg,
	})
}

// validateLeafTypes enforces SPEC §3 leaf-type constraints that the
// strict Go decoder cannot express directly: in particular,
// `invariants:`, `assumptions:`, and `unconstrained:` must map IDs to
// scalar strings, not to nested mappings or sequences.
//
// The strict decoder will reject most of these as type errors when the
// AST declares `map[string]string`, but its error messages are noisy
// and lack source positions. Surfacing them here gives the agent a
// pointed, line-tagged diagnostic.
func validateLeafTypes(path string, root *yaml.Node) []Issue {
	if root == nil || len(root.Content) == 0 {
		return nil
	}
	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return nil
	}
	var issues []Issue
	for i := 0; i+1 < len(doc.Content); i += 2 {
		key := doc.Content[i].Value
		val := doc.Content[i+1]
		switch key {
		case "invariants", "assumptions", "unconstrained":
			issues = append(issues, validateScalarStringMap(path, key, val)...)
		}
	}
	return issues
}

func validateScalarStringMap(path, blockName string, n *yaml.Node) []Issue {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	var issues []Issue
	for i := 0; i+1 < len(n.Content); i += 2 {
		valueNode := n.Content[i+1]
		if valueNode.Kind != yaml.ScalarNode {
			issues = append(issues, Issue{
				Path:   path,
				Line:   valueNode.Line,
				Column: valueNode.Column,
				Message: fmt.Sprintf(
					"`%s.%s` must be a scalar string per SPEC §3; got a %s",
					blockName, n.Content[i].Value, kindName(valueNode.Kind),
				),
			})
		}
	}
	return issues
}

func kindName(k yaml.Kind) string {
	switch k {
	case yaml.DocumentNode:
		return "document"
	case yaml.MappingNode:
		return "mapping"
	case yaml.SequenceNode:
		return "sequence"
	case yaml.ScalarNode:
		return "scalar"
	case yaml.AliasNode:
		return "alias"
	}
	return fmt.Sprintf("unknown kind %d", k)
}
