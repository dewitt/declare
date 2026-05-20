// Package ast defines the in-memory representation of a dx declaration.
//
// The AST mirrors the schema described in SPECIFICATION.md (v0.2.0). It
// is intentionally shallow: the declaration file is the source of
// truth, and the AST is a transparent projection of it.
//
// The v0.2.0 serialization is CommonMark (per SPEC §4). The AST is
// serialization-agnostic: the decoded value types (Declaration,
// Intent, Contract, the three keyed maps) are the same shape that a
// future alternative serialization would produce, and the conceptual
// model in SPEC §3 stays constant across serializations.
//
// Source positions are retained alongside the decoded values so that
// downstream tooling (lint, fmt, diff) can produce line-tagged
// diagnostics. The position structure does not embed any
// CommonMark-library type; the AST has no library dependencies.
package ast

// Declaration is the root of a parsed declaration file.
//
// Field ordering follows SPEC §4.5 ("Canonical Form") so that
// re-emitted canonical forms preserve the recommended agent
// ergonomics.
type Declaration struct {
	System        string              // SPEC §4.3.1
	Intent        []string            // SPEC §4.3.2 (single sentence = 1 element; list = N elements)
	Invariants    map[string]Entry    // SPEC §4.3.3, keyed by slug
	Assumptions   map[string]Entry    // SPEC §4.3.4, keyed by slug
	Contracts     map[string]Contract // SPEC §4.3.5, keyed by slug
	Unconstrained map[string]Entry    // SPEC §4.3.6, keyed by slug

	// Positions records the source line for every structural
	// element the lint pass identified. The map keys are dotted
	// paths like "system", "intent",
	// "invariants.<slug>", "contracts.<slug>.given". Lines are
	// 1-based; 0 means unknown. Populated by the loader (pkg/lint)
	// and consumed by downstream tooling for diagnostics.
	//
	// Positions is nil for declarations constructed in memory
	// without going through the loader.
	Positions map[string]Position

	// BlocksPresent records whether each top-level block heading
	// was observed in the source. Keys are block names exactly as
	// they appear in SPEC §4.2 ("Intent", "Invariants",
	// "Assumptions", "Contracts", "Unconstrained"). This
	// distinction is required to honor SPEC §4.3.4: an
	// ## Assumptions heading with zero ### children encodes
	// "intentionally empty", whereas omitting the heading entirely
	// is a structural error.
	//
	// nil for declarations constructed in memory.
	BlocksPresent map[string]bool
}

// Entry is one keyed entry under ## Invariants, ## Assumptions, or
// ## Unconstrained. The Heading field preserves the original ###
// heading body verbatim (for canonical round-tripping); the slug
// (used as the map key) is derived from Heading per SPEC §4.2.
type Entry struct {
	Heading string // verbatim ### heading body (free-form prose)
	Body    string // verbatim leaf content beneath the heading
}

// Contract is a single black-box verification rule
// (SPEC §4.3.5). All three sub-fields MUST be present in a
// well-formed declaration.
//
// Heading preserves the original ### heading body verbatim; the slug
// derived from it is used as the map key.
type Contract struct {
	Heading string
	Given   string
	When    string
	Then    string
}

// Position records a 1-based source line. A zero Line indicates the
// position is unknown.
type Position struct {
	Line int
}
