# Agent Instruction Protocol: Working with `dx`

This document defines the behavioral constraints for all AI agents
(Archaeologists, Architects, Implementers, and Judges) contributing to
this repository. It is the **shortest** authoritative source: the
canonical operational playbooks live under [`skills/`](skills/) and
elaborate every rule below in role-specific terms.

## 0. Where to Start

Before reading the rest of this file, agents should load
[`skills/dx-orchestrator/SKILL.md`](skills/dx-orchestrator/SKILL.md).
It encodes the prime directive, the `HANDOFF: <from> → <to>: <reason>`
audit-trail format, and the routing table that decides which
role-skill to load for a given task. The role-skills (one each for
the four agents above) build on the rules below.

The behavioral rules in this document follow from the project's
philosophical position — that the `.dx` file is the *idea* of the
system and the imperative code is a derived witness. The full
positioning, including the prior art (Z, TLA+, OWL, Eiffel, the
denotational-semantics tradition) and what's specifically new in
the LLM-as-implementer paradigm, is in
[SPEC.md §1 (Philosophy)](SPEC.md#1-philosophy).
Worth reading once; everything below operationalizes it.

## 1. The Primacy of the Declaration
The `.dx` file is the source of truth. You must never generate
imperative code that violates a defined invariant in the `.dx` file.
If an invariant is technically impossible to satisfy, you must propose
a mutation to the `.dx` file rather than "fixing it in code."

This is the single rule whose violation defeats `dx`. See the
[`implementer`](skills/implementer/SKILL.md) skill for the operational
form: the implementer is forbidden from touching `intent`,
`invariants`, `contracts`, or `unconstrained`; only the
[`architect`](skills/architect/SKILL.md) may modify those blocks.

## 2. Explicit Assumption Logging
When implementation requires a choice not specified in the `intent`
or `invariants`, you **must not** choose silently.
1. Add a new entry to the `assumptions` block in the `.dx` file.
2. Document the heuristic leap **and why it was made**.
3. Only proceed with implementation once the assumption is recorded.

The implementer is the only role permitted to *append* to
`assumptions:` during code generation. The architect *promotes*,
*demotes*, or *rejects* assumptions as a separate operation.

## 3. Verification Loop
Before declaring a task "complete":
1. Execute `dx lint` on all modified `.dx` files. Exit code 0 is
   required. Lint enforces SPEC §2 (no anchors/aliases, no folded
   scalars, no custom tags, scalar leaves under
   `invariants`/`assumptions`/`unconstrained`) and SPEC §3 (required
   keys present).
2. Generate or run the implementation; build/test it in its host
   language.
3. Compare the implementation behavior against the `contracts:`
   block. v0.1.0 has no `dx verify` (deferred per SPEC §4); the
   [`judge`](skills/judge/SKILL.md) skill is the v0.1.0 contract
   executor — an agent walks each contract by hand or via tool-use.
4. If a contract fails, treat the failure as a **semantic bug**, not
   a flaky test. The judge classifies it as either an
   implementation bug or a spec gap and routes to the appropriate
   role.

## 4. Pruning and Parsimony
As an Architect, your goal is the minimum viable constraint set.
Avoid over-specifying. If the user intent can be achieved without a
specific invariant, move that constraint to the `unconstrained:`
block (with a description) or omit it entirely. Over-specification
is a defect — it forecloses future implementations for no benefit.

## 5. Communication with Humans
When discussing changes, use `dx diff` to explain semantic
shifts:

```
dx diff <before>.dx <after>.dx
```

The output is a stable, machine-parseable ledger of operations
(`[ADDED]`, `[REMOVED]`, `[MUTATED]`, `[PROMOTED]`, `[DEMOTED]`,
`[RENAMED]`) ordered by SPEC §2 canonical block order. Paste it into
your handoff or summary.

Do not summarize code changes; summarize changes to the **intent**
and **invariants**. A text diff over YAML obscures the architectural
"why" behind the "how"; the semantic ledger is built for that "why".

## 6. After Merging a `.dx` File
When a `.dx` file is touched on multiple branches and merged, the
architect MUST:

1. Run `dx lint` on the merge result. A textual three-way merge
   can produce structurally invalid YAML.
2. Run `dx diff <merge-base> <merge-result>` to surface every
   semantic operation introduced by the merge. Even a clean text
   merge can hide a semantic conflict (e.g., one branch demoted an
   invariant while the other tightened it).
3. Reconcile any conflict in the **spec**, not the implementation.
   Per §1 the `.dx` file leads.

This is the v0.1.0 substitute for a structural merge tool (SPEC §5);
a future revision may introduce `dx merge`. See
[`skills/dx-toolchain/SKILL.md`](skills/dx-toolchain/SKILL.md)
§6a for the full ritual.
