// Package diff produces a semantic ledger of operations between two
// declarations.
//
// A textual diff over the raw CommonMark is structurally hostile to
// review -- reordering blocks, reflowing prose, or moving an HTML
// comment all explode into noisy red/green even when the spec did
// not change. SPECIFICATION.md §3.9 (Spec Evolution) defines a
// *semantic delta* over the schema as the right unit of change
// reporting:
//
//	[ADDED]    invariants.perf_p99_ms
//	[REMOVED]  unconstrained.storage_backend
//	[MUTATED]  intent.primary
//	[PROMOTED] assumptions.cli_default -> invariants.iface_cli_default
//	[DEMOTED]  invariants.x -> unconstrained.x
//	[RENAMED]  invariants.iface_legacy_a -> invariants.iface_legacy_b
//
// The point of this command is the architect's audit trail (AGENTS.md
// §5): humans should see *what changed about the intent and
// invariants*, not what changed in the bytes.
package diff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dewitt/dx/pkg/ast"
)

// Op enumerates the kinds of semantic edits diff can report.
type Op string

const (
	OpAdded    Op = "ADDED"
	OpRemoved  Op = "REMOVED"
	OpMutated  Op = "MUTATED"
	OpPromoted Op = "PROMOTED"
	OpDemoted  Op = "DEMOTED"
	OpRenamed  Op = "RENAMED"
)

// Change is a single semantic operation in the ledger.
//
// For ADDED / REMOVED / MUTATED, only Path is used; ToPath is empty.
// For PROMOTED / DEMOTED / RENAMED, Path is the source and ToPath is
// the destination; the ID has effectively moved between two locations
// in the schema.
type Change struct {
	Op       Op
	Path     string
	ToPath   string // populated only for PROMOTED, DEMOTED, RENAMED
	OldValue string // populated for MUTATED; empty otherwise
	NewValue string // populated for MUTATED; empty otherwise
}

// String returns the canonical single-line representation used by
// `dx diff`. The format is intentionally machine-parseable: an
// agent can read it back via simple string operations.
func (c Change) String() string {
	switch c.Op {
	case OpPromoted, OpDemoted, OpRenamed:
		return fmt.Sprintf("[%s] %s -> %s", c.Op, c.Path, c.ToPath)
	default:
		return fmt.Sprintf("[%s] %s", c.Op, c.Path)
	}
}

// Diff returns the ordered ledger of changes from oldDecl to newDecl.
// Both arguments must be non-nil; pass an empty *ast.Declaration if a
// side is genuinely missing.
//
// The return order is stable: changes are grouped first by the schema
// block they affect (`system`, `intent`, `invariants`, `assumptions`,
// `contracts`, `unconstrained`) in SPEC §4.5 canonical order, and then
// alphabetically by path within each block. This determinism matters
// for diffs-of-diffs in code review.
func Diff(oldDecl, newDecl *ast.Declaration) []Change {
	if oldDecl == nil {
		oldDecl = &ast.Declaration{}
	}
	if newDecl == nil {
		newDecl = &ast.Declaration{}
	}

	// Flatten both sides into a single map[path]value. We then compute
	// a structural delta against the path space, with a post-pass to
	// recognize content-preserving moves (PROMOTED/DEMOTED/RENAMED).
	oldFlat := flatten(oldDecl)
	newFlat := flatten(newDecl)

	var changes []Change
	added := make(map[string]string)
	removed := make(map[string]string)

	// Phase 1: emit MUTATED for paths present on both sides; collect
	// added/removed for the move-detection phase.
	for path, oldVal := range oldFlat {
		newVal, ok := newFlat[path]
		if !ok {
			removed[path] = oldVal
			continue
		}
		if oldVal != newVal {
			changes = append(changes, Change{
				Op:       OpMutated,
				Path:     path,
				OldValue: oldVal,
				NewValue: newVal,
			})
		}
	}
	for path, newVal := range newFlat {
		if _, ok := oldFlat[path]; !ok {
			added[path] = newVal
		}
	}

	// Phase 2: detect content-preserving moves. For each removed entry,
	// look for an added entry with the same value; emit PROMOTED /
	// DEMOTED / RENAMED depending on whether (and how) the block prefix
	// changed.
	//
	// We iterate removed keys in sorted order so the move-detection is
	// deterministic when multiple paths share the same value.
	for _, oldPath := range sortedKeys(removed) {
		oldVal := removed[oldPath]
		matchPath := findMatchingValue(added, oldVal)
		if matchPath == "" {
			continue
		}
		op := classifyMove(oldPath, matchPath)
		changes = append(changes, Change{
			Op:     op,
			Path:   oldPath,
			ToPath: matchPath,
		})
		delete(removed, oldPath)
		delete(added, matchPath)
	}

	// Phase 3: emit the surviving REMOVED and ADDED entries.
	for _, p := range sortedKeys(removed) {
		changes = append(changes, Change{Op: OpRemoved, Path: p})
	}
	for _, p := range sortedKeys(added) {
		changes = append(changes, Change{Op: OpAdded, Path: p})
	}

	sort.SliceStable(changes, func(i, j int) bool {
		bi, bj := blockOrder(changes[i].Path), blockOrder(changes[j].Path)
		if bi != bj {
			return bi < bj
		}
		return changes[i].Path < changes[j].Path
	})

	return changes
}

// flatten projects a Declaration into a map of dotted paths to scalar
// values. Only fields the diff cares about are included; comment
// metadata, raw nodes, and so on are ignored by design.
//
// Path conventions:
//
//	system
//	intent.primary
//	intent.secondary[0], intent.secondary[1], ...
//	invariants.<id>
//	assumptions.<id>
//	contracts.<name>.given
//	contracts.<name>.when
//	contracts.<name>.then
//	unconstrained.<category>
//
// Numeric secondary indices look like array indices, but they are
// stable identifiers chosen here so that reordering a list shows up as
// an ADDED+REMOVED pair (which is honest: a list reorder *is* a
// semantic change at the position level). A future revision may switch
// to content-keyed identifiers.
func flatten(d *ast.Declaration) map[string]string {
	out := make(map[string]string)

	if d.System != "" {
		out["system"] = d.System
	}
	if d.Intent.Primary != "" {
		out["intent.primary"] = d.Intent.Primary
	}
	for i, s := range d.Intent.Secondary {
		out[fmt.Sprintf("intent.secondary[%d]", i)] = s
	}
	for k, v := range d.Invariants {
		out["invariants."+k] = v
	}
	for k, v := range d.Assumptions {
		out["assumptions."+k] = v
	}
	for name, c := range d.Contracts {
		if c.Given != "" {
			out["contracts."+name+".given"] = c.Given
		}
		if c.When != "" {
			out["contracts."+name+".when"] = c.When
		}
		if c.Then != "" {
			out["contracts."+name+".then"] = c.Then
		}
	}
	for k, v := range d.Unconstrained {
		out["unconstrained."+k] = v
	}
	return out
}

// findMatchingValue returns the first path in m whose value equals
// target, in sorted-key order. Returns "" if no match exists.
func findMatchingValue(m map[string]string, target string) string {
	for _, k := range sortedKeys(m) {
		if m[k] == target {
			return k
		}
	}
	return ""
}

// classifyMove decides whether a same-content move from oldPath to
// newPath is a PROMOTED, DEMOTED, or RENAMED operation.
//
// "Promotion" follows the architect's workflow vocabulary: an entry
// becomes more *committed* as it moves rightward through this chain:
//
//	unconstrained -> assumptions -> invariants
//
// A move toward `invariants` is a promotion; a move away is a demotion.
// Anything within the same top-level block is a rename.
func classifyMove(oldPath, newPath string) Op {
	oldBlock := topBlock(oldPath)
	newBlock := topBlock(newPath)
	if oldBlock == newBlock {
		return OpRenamed
	}
	if commitmentRank(newBlock) > commitmentRank(oldBlock) {
		return OpPromoted
	}
	if commitmentRank(newBlock) < commitmentRank(oldBlock) {
		return OpDemoted
	}
	// Same rank, different block (e.g., contracts <-> intent): not a
	// promotion in the architect sense, treat as a rename.
	return OpRenamed
}

// commitmentRank scores how "committed" an entry in the named block
// is. Higher numbers mean more permanently load-bearing.
func commitmentRank(block string) int {
	switch block {
	case "unconstrained":
		return 0
	case "assumptions":
		return 1
	case "contracts", "intent":
		// Intent and contracts are first-class but not on the
		// constraint axis; rank them with assumptions so a move from
		// them to invariants still reads as a promotion.
		return 1
	case "invariants":
		return 2
	case "system":
		return 3
	}
	return -1
}

// topBlock returns the first dotted segment of a path.
func topBlock(path string) string {
	if i := strings.IndexByte(path, '.'); i >= 0 {
		return path[:i]
	}
	return path
}

// blockOrder gives the SPEC §4.5 canonical sort order for top-level
// blocks; unknown blocks sort last.
func blockOrder(path string) int {
	switch topBlock(path) {
	case "system":
		return 0
	case "intent":
		return 1
	case "invariants":
		return 2
	case "assumptions":
		return 3
	case "contracts":
		return 4
	case "unconstrained":
		return 5
	}
	return 99
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
