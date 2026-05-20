# AGENTS

Behavioral protocol for any agent (human or AI) modifying this
repository. Five rules. Operational details live in
[`WORKFLOW.md`](WORKFLOW.md) and the [`skills/`](skills/) directory;
the language itself is defined in [`SPECIFICATION.md`](SPECIFICATION.md).

## 0. Load the orchestrator skill first

Before doing anything else, an AI agent operating in this repo
should load
[`skills/dx-orchestrator/SKILL.md`](skills/dx-orchestrator/SKILL.md).
It encodes the role-routing table, the handoff format, and the
prime directive in a form that an agent runtime can act on
directly. The rules below assume that has happened.

A human reading this file may skip to rule 1.

## 1. The declaration is the source of truth

Never write imperative code that violates a defined invariant in
the declaration. If an invariant is technically impossible to
satisfy, propose a mutation to the declaration. Do not weaken the
code to make a passing fit.

This is the rule whose violation defeats the entire system. The
write privileges that operationalize it are in
[`WORKFLOW.md` § "Write privileges"](WORKFLOW.md#write-privileges-by-block):
the implementer is forbidden from touching `## Intent`,
`## Invariants`, `## Contracts`, or `## Unconstrained`; only the
architect may modify those blocks.

## 2. Log heuristic choices before acting on them

When implementation requires a choice not specified in
`## Intent` or `## Invariants`:

1. Add an entry to the `## Assumptions` block of the declaration.
2. Document both the choice that was made and the reason it was
   the most defensible choice given the ambiguity.
3. Only then write the code that depends on the choice.

Silent invention is the failure mode this rule exists to prevent.
Any party (human or agent) producing or modifying an
implementation is bound by it; see
[`SPECIFICATION.md` §3.5](SPECIFICATION.md#35-assumptions).

The implementer is the only role permitted to *append* to
`## Assumptions` during code generation. The architect *promotes*,
*demotes*, or *rejects* assumptions later, as a separate
operation.

## 3. Run the verification loop before declaring a task complete

Lint the spec, build the implementation, walk every contract,
classify any failure. The detailed loop and failure-classification
rules are in [`WORKFLOW.md` § "The verification loop"](WORKFLOW.md#the-verification-loop).
Skipping any step is the most common way bugs ship.

## 4. Minimum viable constraint set

When acting as architect, the goal is the smallest spec that
captures the intent. If a requirement can be met without an
explicit invariant, leave it in `## Unconstrained` (with a
description) or omit it entirely. Over-specification forecloses
future implementations for no benefit.

## 5. Communicate spec changes via `dx diff`, not text diffs

When summarizing a declaration change for a human or another
agent, run `dx diff <before>.md <after>.md` and paste the output.
The result is a stable ledger of operations (`[ADDED]`,
`[REMOVED]`, `[MUTATED]`, `[PROMOTED]`, `[DEMOTED]`, `[RENAMED]`)
ordered by canonical block order; it surfaces what changed at the
level of intent and constraints, not at the level of bytes.

Do not summarize code changes when a declaration change is what
matters. If both changed, summarize the spec change first; the
code change is downstream.

## 6. After merging a declaration

Run the post-merge ritual from
[`WORKFLOW.md` § "The post-merge ritual"](WORKFLOW.md#the-post-merge-ritual):
lint the merge result, diff it against the merge base, reconcile
any semantic conflict in the spec rather than the implementation.
A clean text-merge can hide a semantic conflict.
