---
name: dx-orchestrator
description: |
  Use when working in a dx-managed repository (any directory
  containing one or more dx declaration files, or whose `AGENTS.md`
  references the dx paradigm). Routes the agent to the correct
  role-skill (archaeologist, architect, implementer, judge),
  enforces the prompt-first workflow, and prevents silent semantic
  drift between human intent and generated code. Read this before
  answering any task that involves reading, writing, or executing
  code governed by a declaration.
---

# dx Orchestrator

`dx` is a declarative specification language for the agentic AI
era. The declaration (a CommonMark file with canonical extension
`.md`) is the **source of truth** for intent and constraints; the
imperative code is a derived artifact. Your first job in any dx
repo is to figure out which role you are playing and load the
corresponding skill.

This skill does not do work itself. It routes.

## 1. The Prime Directive

> **The declaration is the source of truth.** Never write
> imperative code that violates a defined invariant. If an
> invariant is technically impossible to satisfy, propose a
> mutation to the declaration rather than "fixing it in code."

Violating this rule is the single failure mode `dx` exists to
prevent. If you find yourself about to special-case the code to
make it work, stop and re-read the declaration.

## 2. Role Routing

Pick exactly one role per task. Load its skill and follow it strictly.

| Trigger phrase / situation                                                 | Load this skill        |
| -------------------------------------------------------------------------- | ---------------------- |
| "Reverse-engineer this codebase into a declaration."                       | `archaeologist`        |
| "There's no declaration here yet — distill one from the existing source."  | `archaeologist`        |
| "Write/refine the declaration." / "Add an invariant." / "Tighten intent."  | `architect`            |
| "Promote this assumption." / "Move X from assumptions to invariants."      | `architect`            |
| "Generate the implementation from `system.md`."                            | `implementer`          |
| "Make the code conform to the spec." / "Fix this code to satisfy X."       | `implementer`          |
| "Verify the implementation against the contracts."                         | `judge`                |
| "Run the contracts. Tell me which ones fail and why."                      | `judge`                |
| Pure CLI usage (`dx lint`, `fmt`, `diff`, `export`)                        | `dx-toolchain`         |
| Reconciling a declaration merge (lint then diff against merge base)        | `dx-toolchain` §6a + `architect` |
| Spec questions ("How do I encode an empty block?", "What goes in `## Intent`?") | `dx-authoring`    |

If the task spans multiple roles (common), execute them **sequentially**:
architect → implementer → judge. Never have one role silently doing another's
job — for example, the implementer must not add or remove invariants; that is
strictly the architect's job.

## 3. Universal Pre-Flight Checks

Before any role-specific work:

1. **Locate the declarations.** Look for `.md` files that begin
   with a `# <system>` heading followed by `## Intent`. At the
   repo root and any obvious subdirectories. If there are none
   and the user is asking you to write code, stop and ask whether
   to invoke the `archaeologist` first.
2. **Validate them.** Run `dx lint <file>.md` on every
   declaration you intend to read or modify. Lint errors mean the
   file is structurally untrustworthy; fix them (as architect)
   before doing anything else.
3. **Check for an `AGENTS.md`.** It encodes repo-specific
   conventions that override generic skill guidance.

## 4. The Universal Invariants (apply to every role)

These come from `AGENTS.md` and apply regardless of which
role-skill you load. Re-read them before each non-trivial action.

### 4a. Explicit Assumption Logging (AGENTS.md §2)

When implementation requires a choice not specified in
`## Intent` or `## Invariants`, you **must not** choose silently.

1. Add an entry to the `## Assumptions` block of the declaration
   as a new `### <id>` section.
2. Document the heuristic leap **and why it was made**.
3. Only then proceed.

This is the single mechanism by which `dx` converts silent LLM
hallucinations into auditable, promotable workflow state.
Skipping it defeats the entire system.

### 4b. The Verification Loop (AGENTS.md §3)

Before declaring any task complete:

1. `dx lint` every modified declaration. Must exit 0.
2. Generate or run the implementation.
3. Compare implementation behavior against the `## Contracts`
   block.
4. Treat any contract failure as a **semantic bug**, not a flaky
   test.

### 4c. Pruning and Parsimony (AGENTS.md §4)

The architect's goal is the **minimum viable constraint set**.
If a requirement can be met without an explicit invariant, that
requirement belongs in `## Unconstrained`, not `## Invariants`.
Over-specification is a defect.

### 4d. Semantic Communication (AGENTS.md §5)

When summarizing changes for a human, summarize changes to
**intent and invariants**, not to lines of code. Use
`dx diff <before>.md <after>.md` rather than text diffs.

## 5. Handoff Protocol

When transitioning from one role to another within the same task, write a
single-line handoff in the conversation:

```
HANDOFF: <from-role> → <to-role>: <one-sentence reason>
```

Examples:

```
HANDOFF: architect → implementer: invariants are stable; generate Go code under cmd/.
HANDOFF: implementer → judge: implementation compiles and lints; run contracts.
HANDOFF: judge → architect: contract `greets_a_named_user` failed because the spec is
ambiguous about trailing whitespace — needs a new invariant or `## Unconstrained` entry.
```

This makes the loop legible to humans reviewing the transcript.

## 6. When to Stop and Ask

Escalate to the human (do not proceed silently) when:

- Two invariants directly contradict each other.
- A contract failure cannot be cleanly classified as either an implementation
  bug or a spec gap.
- You would need to add **more than three** new assumptions to make progress;
  that signals the spec is too sparse and the architect should be invoked
  explicitly by the human.
- The user requests imperative behavior that would violate an existing
  invariant. Cite the invariant by ID and ask whether to mutate the spec.

## 7. Anti-Patterns (Do Not Do)

- **"Fix it in code."** Never. Mutate the declaration or escalate.
- **Silent assumption.** Never ship code that embeds a heuristic
  choice not recorded in `## Assumptions`.
- **Role bleed.** The implementer never edits `## Invariants`.
  The architect never writes the implementation. The judge never
  modifies either.
- **Skipping `dx lint`.** Always run it on every declaration you
  touch.
- **Over-specifying.** If the human didn't constrain it, it
  belongs in `## Unconstrained`, not in a fabricated invariant.
