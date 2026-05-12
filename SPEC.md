# Specification: The `.dx` Language (v0.1.0)

## 1. Physical Format
Files must be valid YAML 1.2. The canonical file extension is `.dx`.

## 2. Structural Constraints
To maintain a deterministic Abstract Syntax Tree (AST) and prevent semantic drift during AI processing, the following restrictions apply:
*   **No Anchors/Aliases:** The use of `&` (anchors) and `*` (aliases) is strictly forbidden.
*   **No Complex Tags:** Custom YAML tags (e.g., `!!binary`, `!!set`) are not supported.
*   **Literal Scalars Only:** All multiline strings must use the literal block scalar (`|`). The folded scalar (`>`) is prohibited due to ambiguous whitespace handling in diverse LLM tokenizers.
*   **Root Key Ordering:** While YAML is unordered, agents should prefer the order: `system`, `intent`, `invariants`, `assumptions`, `contracts`, `unconstrained`.

## 3. Schema Definitions

### `system` (Required)
A unique identifier for the declaration.
- Type: String (Slug format)

### `intent` (Required)
The high-level semantic purpose of the implementation.
- `primary`: The core objective.
- `secondary`: (Optional) Supporting objectives or non-functional goals.

### `invariants` (Required)
Non-negotiable constraints that the implementation must satisfy. 
- Map of `id: string`.
- Keys should be prefixed by category (e.g., `sec_`, `perf_`, `iface_`).

### `assumptions` (Required)
Heuristics or design choices made by the agent that require human validation.
- Map of `id: string`.
- Empty maps are allowed but the key must exist to signal a "zero-assumption" state.

### `contracts` (Optional)
Verifiable state-transition rules for black-box testing.
- Map of named contract blocks.
- Fields: `given` (initial state), `when` (execution triggers), `then` (expected outcome/side-effect).

### `unconstrained` (Optional)
Explicitly declared degrees of freedom.
- Map of `category: description`.

## 4. Verification Model

`declare` v0.1.0 ships **without** a built-in contract executor.

Verification of an implementation against the `contracts:` block is performed by an agent operating under the `judge` skill (see `skills/judge/SKILL.md`). The judge interprets each contract's `given` / `when` / `then` clauses as prose, sets up the precondition, runs the implementation, and evaluates the observable outcome. Pass/fail classification (implementation bug vs. spec gap vs. intent mismatch) is the judge's responsibility.

This deliberately defers a `declare verify` command to a future revision. The genesis design discussion (see `docs/origins/`) considered baking a contract harness into the CLI; we chose human/agent-driven verification for v0.1.0 because (a) it ships immediately and (b) it keeps the CLI strictly LLM-free per ARCHITECTURE.md §4. A `declare verify` command remains a candidate for v0.2.

Until then, the verification loop in AGENTS.md §3 remains the single source of truth: `declare lint` is mechanical; everything downstream of it is the judge's job.

## 5. Concurrent-Edit Conflict Resolution

`.dx` files are version-controlled like source code. v0.1.0 deliberately does **not** define a structural merge algorithm; concurrent edits resolve through whatever VCS the project uses (typically git's three-way merge).

After any merge, the architect MUST:

1. Run `declare lint` on the merge result. A textual merge can produce structurally invalid YAML (e.g., duplicated keys, indentation breaks); lint catches these immediately.
2. Run `declare diff <merge-base> <merge-result>` to surface every semantic operation introduced by the merge. A clean text-merge can still hide a semantic conflict (e.g., one branch demoted an invariant to `unconstrained:` while the other branch tightened it).
3. Reconcile any semantic conflict in the spec, not in the implementation. Per AGENTS.md §1, the `.dx` file leads.

A future revision may introduce a CRDT-style structural merge (`declare merge --base ours --theirs`) that operates over the AST directly and surfaces semantic conflicts as first-class operations. v0.1.0 does not.

## 6. Reserved Field Names (Future Compatibility)

The following field names are **reserved** within `invariants:`, `assumptions:`, `contracts:`, and `unconstrained:` map values. v0.1.0 does not require them, but a future revision may attach normative semantics to each. Tooling MUST NOT use them for unrelated purposes.

- `rule` — the constraint or assertion text (the body of a v0.1.0 leaf).
- `reason` — free-form prose explaining *why* the entry exists.
- `author` — the agent or human responsible for the most recent mutation (e.g., `agent-architect@cloudcode/2026-05-12`).
- `since` — the spec version or change identifier in which the entry first appeared.

In v0.1.0, `invariants:` / `assumptions:` / `unconstrained:` leaves are scalar strings. The reserved-field set anticipates a v0.2 transition to a structured shape:

```yaml
# Forward-compatible v0.2 sketch -- NOT valid v0.1.0:
invariants:
  perf_cache_ttl:
    rule: |
      Cache TTL must be strictly 600 seconds.
    reason: |
      Upstream API documentation forbids polling faster than 10 minutes.
    author: agent-architect@cloudcode
    since: v0.1.0
```

The genesis design discussion proposed this audit-trail shape and explicitly deferred it to keep v0.1.0 minimal. Reserving the names now lets a future revision adopt the structured form without colliding with field names already in use.

## 7. Versioning

This document describes v0.1.0 of the `.dx` language. Future revisions will be released as `v0.MAJOR.MINOR`:

- **Patch** (`v0.1.x`): clarifications, additional reserved names, additional linter checks that reject already-questionable input. No new required fields.
- **Minor** (`v0.x.0`): new optional blocks, structured forms of existing leaves (gated by the reserved-field discipline in §6), new CLI commands.
- **Major** (`v1.0.0`): commitment to long-term backward compatibility.

v0.1.0 does not include a top-level spec-version declaration. A future revision will introduce one (likely a top-level `dx_spec:` key); until then, `.dx` files have no in-band version marker and are assumed to target the current released spec.
