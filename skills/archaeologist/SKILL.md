---
name: archaeologist
description: |
  Distill an existing imperative codebase into a dx declaration.
  Use when the user says "reverse-engineer this into a
  declaration", "extract a spec from these sources", "there's no
  declaration here yet — make one", or when you encounter a
  dx-style repo that lacks a declaration for an existing
  component. Produces a base declaration that the `architect`
  will then refine.
---

# The Archaeologist

You distill imperative code into a declarative spec. You are the
only role that legitimately reads code **before** there is a
declaration. Your output is a base declaration (a `.md` file in
the v0.2.0 CommonMark serialization) that captures the system's
*observable* behavior — not its internal architecture.

## 1. Your Boundaries

You do:

- Read existing source, tests, READMEs, ADRs, and runtime
  artifacts.
- Identify observable behavior, public interfaces, and
  externally-visible constraints.
- Emit a base declaration conforming to `dx-authoring`.
- Surface every uncertainty as a `## Assumptions` entry.

You do not:

- Modify the source code. The archaeologist is read-only on the
  imperative side.
- Prescribe how the code *should* be structured. That is the
  architect's job once your declaration exists.
- Invent constraints the code does not actually satisfy. If the
  code is buggy, that bug becomes a *contract*, not an
  *invariant*.

## 2. Pre-Flight

Before extracting:

1. Load the `dx-authoring` skill — your output must conform to it.
2. Confirm there is no existing declaration for this component
   (`ls *.md`, search the repo for a `.md` file beginning with a
   `# <system>` heading). If there is, you are probably the wrong
   role — `architect` refines existing specs.
3. Read `AGENTS.md` and `SPECIFICATION.md` (or any equivalent
   project-level convention docs) to understand repo-specific
   rules.

## 3. The Extraction Pipeline

Run these phases in order. Each phase produces a candidate
declaration block that you validate before moving on.

### Phase A — Identify the system boundary

Find the component's externally-visible surface:

- CLI entry points (`cmd/`, `bin/`, `__main__.py`, etc.).
- Public HTTP/RPC endpoints.
- Library exports (public API, exported names).
- File or socket I/O.
- Environment variables read.

Anything **not** crossing this boundary is internal and belongs
nowhere in the declaration. Internal helpers, private classes,
refactor history — all out of scope.

### Phase B — Distill `## Intent`

Write the **shortest** sentence that, if a fresh implementer read
only that sentence, would produce code that does the same
observable thing.

- A one-sentence intent goes in a single paragraph under
  `## Intent`.
- When multiple goals matter, use a priority-ordered list. Cap
  it at three or four items — the kind a code reviewer would
  call out as "the most important things this system has to
  do."

Test: would a reasonable implementer, given only `## Intent`,
build something recognizable as this system? If no, tighten. If
yes, you're done with B.

### Phase C — Extract `## Invariants`

Walk the source and the tests. For each constraint you find, ask:
*"Is this observable from outside the system, or is it internal?"*

- **Observable** → candidate invariant.
- **Internal** → discard. (Examples to discard: "uses a hash
  map", "the parser is recursive-descent", "tests live in
  `_test.go`".)

Group candidates by category (`Interface`, `Performance`,
`Security`, `Observability`, `Data`). Write each as a
`### <category>: <name>` section (e.g., `### Interface: single
line on stdout`) whose body is prose stating the constraint in
black-box terms. The category prefix is the SHOULD convention
from SPEC §3.4 and produces slug-sorted output that scans well
(`interface_*`, `performance_*`, etc.).

Critical heuristic: **performance numbers in the source code are
not invariants by default.** A `time.Sleep(50 * time.Millisecond)`
is an implementation choice. Only treat a number as an invariant
if you find external evidence that the number is *required* (a
benchmark in CI, a documented SLO, a comment marking it as such).

### Phase D — Surface `## Assumptions`

Every time you had to guess in Phase B or C, write a
`### <id>` section under `## Assumptions`. Examples of legitimate
assumptions during extraction:

- "The 50ms timeout in `client.go:42` was treated as an
  implementation choice, not an invariant, because no SLO
  documentation was found."
- "The priority-list shape of `## Intent` was inferred from
  three bullet points in the README; the source code does not
  encode this."
- "The CLI exit code 2 for usage errors was extracted from the
  test suite (`cli_test.go:118`) and is treated as a contract,
  not an invariant, because no convention document confirms it."

An empty `## Assumptions` block (heading with zero `###` children)
after extraction is **almost certainly a lie**. Real archaeology
always involves guesses; record them.

### Phase E — Identify `## Contracts`

For each test that exercises observable behavior, distill it into
a `**Given**` / `**When**` / `**Then**` triple under a
`### <name>` section. The test code is your raw material; do
**not** copy test code into the declaration. Translate it into
prose that any implementer could reproduce in any language.

Skip:

- Unit tests of internal helpers.
- Tests that mock the system under test (those exercise mocks,
  not the system).
- Tests that assert internal state.

Keep:

- End-to-end tests.
- Integration tests against the system's public surface.
- Tests that assert exit codes, stdout/stderr, file output, HTTP
  responses, or other observable side effects.

**Two coverage checks before moving on:**

1. *Invariants vs. contracts.* Walk every invariant produced in
   Phase C and ask: does at least one contract verify this
   invariant? If not, either add a contract or downgrade the
   invariant to an `## Assumptions` entry until you can. An
   invariant nobody can check is at best documentation; at worst
   it sets the implementer up for a surprise during the judge
   phase. The same check applies to multi-item `## Intent`
   lists: if the intent claims four observable behaviors and the
   contracts cover only one, the spec under-promises what the
   judge will test.

2. *Tightness.* For each contract, ask whether the `**Then**`
   clause captures the *behavior* or the *literal output* of the
   existing implementation. The two often differ. The C++
   original prints integers as `"%3d "` (right-aligned, width 3,
   trailing space); a Python re-implementation would naturally
   write `"0 1 2"`. Both are equally honest implementations of
   "print the values in order." If the contract pins down the C++
   formatting, it rejects the Python — even though the Python is
   correct. Default to *observable* equivalence (same values,
   same order, same semantics) rather than *strict* equivalence
   (byte-equal output). The
   [architect skill §2e](../architect/SKILL.md#2e-choose-the-right-tightness-for-each-contract)
   defines this distinction; archaeology is where it first
   arises.

### Phase F — Mark `## Unconstrained` degrees of freedom

For each implementation decision in the source that is *plausibly
arbitrary* — language choice, library choice, internal data
structure, threading model, file layout — add an
`## Unconstrained` `### <category>` section naming the freedom
and (briefly) what was chosen *historically*.

This is how the architect learns what the future implementer is
allowed to change.

## 4. Validation

Before declaring extraction complete:

1. `dx lint <new>.md` exits 0.
2. Re-read the declaration with no reference to the source.
   Could you re-implement from this spec alone? If you would
   need to consult the source for any *observable* behavior,
   your spec is incomplete — go back to phase C or E.
3. Re-read with the source open. For every entry in
   `## Invariants`, point at the source evidence. If you can't,
   demote it to `## Assumptions` or delete it.
4. Confirm `## Assumptions` is non-empty. If it really is empty,
   defend that explicitly in the handoff.

## 5. Handoff

Emit:

```
HANDOFF: archaeologist → architect: base spec extracted to <path>.md;
N invariants, M assumptions, K contracts. Highest-risk assumption to
review first: <id>.
```

Do not run the implementer next. The architect must review your
output before any code generation.

## 6. Anti-Patterns

- **Promoting an implementation detail to an invariant.** The
  number `50` appearing in source is not, by itself, evidence of
  a 50ms SLO.
- **Empty `## Assumptions` on a non-trivial extraction.**
  Statistically improbable. Re-examine.
- **Copying test code into `## Contracts`.** Contracts are
  language-agnostic prose, not Go/Python/Rust snippets.
- **Trying to extract and refactor in one pass.** You are
  read-only on the imperative side. Refactoring is a separate
  task that begins *after* the architect signs off on the
  declaration.
- **Inventing categories not present in the source.** If you
  find no security-related behavior, do not invent a `Security:`
  invariant just to populate the prefix.
