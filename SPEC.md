# Specification: The `.dx` Language (v0.1.0)

This document is the normative reference for the `.dx` language. The
intellectual position behind it — why a separate declarative artifact,
and why now — is in [ARCHITECTURE.md §1 (Philosophy)](ARCHITECTURE.md#1-philosophy).
The rules below operationalize that position; reading them in
isolation is fine, but the philosophy explains *why* they take the
shape they do.

## 1. Physical Format

Files MUST be valid YAML 1.2 (subject to the structural constraints
in §2). The canonical file extension is `.dx`.

YAML was chosen as the substrate after considering JSON, TOML, HCL,
and a custom DSL. The decision rests on four properties that matter
specifically for the LLM-mediated authoring context:

- **Universal editor support.** Every modern editor highlights YAML
  out of the box. No plugin, no language-server install, no setup
  cost for a human reviewer of any background.
- **Multi-line ergonomics.** The literal block scalar (`|`)
  preserves human-authored bytes line-by-line, which matters when
  a contract's `then:` clause references observable output
  verbatim. JSON has no native multi-line story; TOML's is awkward;
  HCL's is fine but locks adoption to the HashiCorp ecosystem.
- **Comment support.** YAML allows `#` comments. This is essential
  for human authoring and review. JSON's lack of comments alone
  rules it out for a spec language meant to be read by both humans
  and machines.
- **Deterministic AST.** YAML 1.2 is well-specified and produces
  stable parse trees across implementations *when the strict-subset
  rules in §2 are applied*. Without those rules YAML is famously
  unpredictable; the §2 constraints exist precisely to recover the
  determinism that the broader YAML spec sacrifices.

A custom DSL was rejected because tree-sitter / syntax-highlighter
investment is a real cost, and no candidate DSL we considered
offered enough advantage over a strict YAML subset to justify it.

## 2. Structural Constraints

To maintain a deterministic Abstract Syntax Tree (AST) and prevent
semantic drift during agent processing, the following restrictions
apply. All MUST and MUST NOT clauses below are enforced by
`declare lint` and produce a non-zero exit on violation.

*   **No Anchors / Aliases.** A `.dx` file MUST NOT use YAML
    anchors (`&name`) or aliases (`*name`). They introduce hidden
    state that breaks an agent's local reasoning over the document.
*   **No Custom Tags.** A `.dx` file MUST NOT use explicit YAML
    tags outside the implicit core schema (`!!str`, `!!int`,
    `!!float`, `!!bool`, `!!null`, `!!seq`, `!!map`, `!!timestamp`).
    `!!binary`, `!!set`, and any user-defined `!foo` tags are
    rejected.
*   **Literal Scalars Only.** Multi-line strings MUST use the
    literal block scalar (`|`). The folded scalar (`>`) is rejected
    because it collapses newlines into spaces in ways that vary
    subtly across YAML libraries and LLM tokenizers — the resulting
    decoded value is no longer reliably the bytes the human wrote.
*   **Scalar Leaves.** Map values inside `invariants`,
    `assumptions`, and `unconstrained` MUST be scalar strings, not
    nested mappings or sequences. (See §6 for the v0.2 reserved
    field set, which anticipates relaxing this rule to allow a
    structured leaf shape.)
*   **Root Key Ordering.** A `.dx` file SHOULD list its top-level
    keys in this order: `system`, `intent`, `invariants`,
    `assumptions`, `contracts`, `unconstrained`. The `declare fmt`
    command enforces this ordering automatically, sorts entries
    within `invariants` / `assumptions` / `contracts` /
    `unconstrained` alphabetically, and produces a byte-stable
    canonical form. `declare export` produces the same form with
    comments stripped. A spec that violates the SHOULD will lint
    cleanly but is not in canonical form.

## 3. Schema Definitions

### `system` (Required)

A unique identifier for the declaration. Acts as the namespace for
this `.dx` file; a multi-spec project distinguishes its specs by
this name.

- Type: String (slug format; conventionally kebab-case, no leading
  digit).

### `intent` (Required)

The high-level semantic purpose of the system. Operationalizes the
"the `.dx` file is the *idea* of the system, written down" position
in [ARCHITECTURE.md §1](ARCHITECTURE.md#1-philosophy): a fresh
implementer reading only `intent` should understand what the system
is *for*, even if they cannot yet build it.

- `primary` (required) — the core objective, in one sentence. A
  string scalar.
- `secondary` (optional) — supporting goals or non-functional
  objectives, ordered by author intent (the order is preserved by
  `declare fmt`). A list of string scalars.

### `invariants` (Required)

Non-negotiable observable constraints that the implementation must
satisfy. Operationalizes positions 1 (the `.dx` file is primary)
and 4 (verification is observational) in
[ARCHITECTURE.md §1](ARCHITECTURE.md#1-philosophy): each invariant
is a proposition about the system's externally-visible behavior
that all valid implementations must honor. Invariants describe
*what is true*, not *how to compute it* — a well-formed invariant
never names a language, library, framework, or internal data
structure.

- Map of `id: string`. Empty map (`{}`) is allowed; the key MUST
  exist even when there are no invariants.
- Keys SHOULD carry a category prefix (`iface_`, `perf_`, `sec_`,
  `obs_`, `data_`, `ux_`, or a project-defined prefix used
  consistently within a single file).
- The body is a string scalar describing the constraint in
  black-box terms.

### `assumptions` (Required)

Heuristic choices an agent made because `intent` and `invariants`
did not uniquely determine the answer. Operationalizes position 3
(heuristic leaps as first-class artifacts) in
[ARCHITECTURE.md §1](ARCHITECTURE.md#1-philosophy): the entire
purpose of the block is to convert silent invention into auditable,
promotable workflow state. Each entry is a *what was decided* paired
with a *why it was the most defensible choice given the ambiguity*.
The architect later promotes a ratified assumption to `invariants`,
demotes one to `unconstrained`, or rejects it (rewriting the spec
so the assumption becomes unnecessary).

- Map of `id: string`.
- Empty map (`{}`) is allowed and meaningful: it asserts "the
  agent made no unrecorded heuristic choices." The key MUST exist
  even in this state to distinguish "intentionally empty" from
  "forgot to record."
- The body is a string scalar describing the assumption.

### `contracts` (Optional)

Black-box verification rules in given/when/then form. Operationalizes
position 4 (verification is observational) in
[ARCHITECTURE.md §1](ARCHITECTURE.md#1-philosophy): a contract is
a recipe an outside observer can run to confirm an invariant holds.
Every clause MUST reference observable state (stdout, stderr, exit
code, file system state, HTTP response, log output, …) — never
internal program state.

- Map of `name: contract`.
- Each `contract` is a mapping with three string-scalar fields:
  - `given` — the precondition under which the contract applies.
  - `when` — the triggering action or event.
  - `then` — the observable outcome that MUST hold.
- All three fields are conventionally present; a contract that
  cannot express any one of them in observable terms is a signal
  that the underlying invariant is not testable as a black box and
  may need rephrasing.

### `unconstrained` (Optional)

Explicitly declared degrees of freedom. Operationalizes position 2
(implementations are plural and replaceable) in
[ARCHITECTURE.md §1](ARCHITECTURE.md#1-philosophy): each entry says
"this aspect of the system is *not* constrained by the spec; the
implementer may choose freely." Without this block, every
unspecified aspect is ambiguously either an oversight or an
intentional non-constraint; this block disambiguates.

- Map of `category: description`. Both are string scalars.
- Common categories include `language`, `internal_data_structures`,
  `cache_format`, `output_phrasing`, `concurrency_model` — anything
  the spec wants to leave open.
- Use this block aggressively. Over-specification is a defect: if
  the spec did not intend to constrain something, it MUST be either
  in this block or absent altogether.

## 4. Verification Model

`declare` v0.1.0 ships **without** a built-in contract executor.

Verification of an implementation against the `contracts:` block is performed by an agent operating under the `judge` skill (see `skills/judge/SKILL.md`). The judge interprets each contract's `given` / `when` / `then` clauses as prose, sets up the precondition, runs the implementation, and evaluates the observable outcome. Pass/fail classification (implementation bug vs. spec gap vs. intent mismatch) is the judge's responsibility.

This deliberately defers a `declare verify` command to a future revision. The genesis design discussion (see `docs/origins/`) considered baking a contract harness into the CLI; we chose human/agent-driven verification for v0.1.0 because (a) it ships immediately and (b) it keeps the CLI strictly LLM-free per ARCHITECTURE.md §5. A `declare verify` command remains a candidate for v0.2.

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
    rule: Cache TTL must be strictly 600 seconds.
    reason: Upstream API documentation forbids polling faster than 10 minutes.
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
