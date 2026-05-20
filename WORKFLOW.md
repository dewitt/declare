# The dx Workflow

This document describes the recommended multi-agent workflow for
operating the dx language. It is prescriptive — it tells you what
to do — but it is not normative. The dx language [SPECIFICATION.md](SPECIFICATION.md)
permits other operationalizations: a solo human authoring a
declaration and implementing it themselves, a single agent acting in all
roles across one session, a fully automated pipeline that never
involves a human reviewer. This workflow is the shape we have
field-tested and recommend; it is the shape the [`skills/`](skills/)
directory enforces; it is the shape used by every journey under
[`docs/journeys/`](docs/journeys/).

If you intend to use dx without an AI agent at all, this document
is informational. The behavioral disciplines below — separating
spec edits from code edits, recording heuristic choices before
acting on them, treating contracts as the conformance gate — apply
to human authors too, but the multi-role structure exists primarily
to keep AI-mediated work auditable.

Declarations are CommonMark files (canonical extension `.md`) per
[`SPECIFICATION.md` §4](SPECIFICATION.md#4-serialization-v020).

## How this document relates to the others

| Document | Defines | Scope |
|---|---|---|
| [`SPECIFICATION.md`](SPECIFICATION.md) | The dx language | Universal; replaceable toolchains and workflows. |
| [`README.md`](README.md) | The reference toolchain | One toolchain implementation; the `dx` binary. |
| `WORKFLOW.md` (this doc) | The recommended workflow | One operationalization; the multi-agent loop. |
| [`skills/`](skills/) | Per-role enforcement | Implements this workflow as agent-loadable skills. |
| [`docs/journeys/`](docs/journeys/) | End-to-end use cases | Concrete walkthroughs that follow this workflow. |

The spec is the source of truth for the language. This document is
the source of truth for the workflow. The skills implement the
workflow in a form an agent runtime can load and follow. The
journeys show the workflow applied to specific tasks.

## The four roles

The workflow names four roles. A role is a *position in the
workflow*, not necessarily a separate agent or a separate human:
one agent may play multiple roles in different turns, and a solo
human author plays all four. What matters is that the work
performed under each role respects that role's boundary on the
declaration.

### Archaeologist

Distills an existing imperative implementation into a
declaration. Operates only when the system already exists in
code form; greenfield projects skip this role and start with the
architect.

The archaeologist reads the source code (and any existing tests,
documentation, or runtime artifacts), identifies the system's
*observable* behavior, and produces a base declaration capturing
that behavior. The archaeologist is read-only on the imperative
code: it does not modify the existing implementation and does not
prescribe how a future implementation should be structured.

The archaeologist's job is to surface *every* heuristic leap as
an `## Assumptions` entry. A first-pass extraction with an empty
`## Assumptions` block is almost always a lie; real archaeology
involves real guesses, and recording them is half the value.

### Architect

Owns the declaration. Authors and refines `## Intent`,
`## Invariants`, `## Contracts`, and `## Unconstrained` blocks;
reviews and resolves `## Assumptions` over time.

The architect is the *only* role with write access to the
spec-defining blocks (`## Intent`, `## Invariants`,
`## Contracts`, `## Unconstrained`). Other roles read these;
only the architect modifies them. The architect's discipline is
the *minimum-viable-constraint set*: every invariant the
architect adds is a constraint on every future implementation,
so the architect prunes aggressively.

The architect resolves `## Assumptions` entries over time into
one of three outcomes:

- **Promote** to `## Invariants` — the choice is ratified as a
  binding constraint.
- **Demote** to `## Unconstrained` — the choice is confirmed as
  not constraining any future implementation.
- **Reject** — the spec is modified so the assumption becomes
  unnecessary.

### Implementer

Produces or modifies imperative code from the declaration. Reads
the spec and may *append* to its `## Assumptions` block when
forced to make a heuristic choice during implementation, but
**must not** modify any other block. If an invariant is
impossible to satisfy, the implementer escalates to the
architect rather than weakening the code's conformance.

The implementer's discipline is *no silent invention*. When the
spec is ambiguous, the implementer logs the choice in
`## Assumptions` *before* writing the code that depends on it.
After the fact is too late: the architect's later review depends
on the assumption being a deliberate, recorded decision rather
than a reverse-engineered explanation.

### Judge

Determines whether a given implementation satisfies the
declaration's `## Contracts` block. Walks every contract,
classifies any failure, and routes findings to the appropriate
role. The judge writes nothing — not the code, not the spec —
and produces only a verdict.

### Write privileges, by block

The four roles have non-overlapping write authority on the
declaration. The boundary is the load-bearing property that
prevents role bleed and keeps the audit trail honest.

| Block | Archaeologist | Architect | Implementer | Judge |
|---|---|---|---|---|
| `# <system>` | W (initial only) | W | — | — |
| `## Intent` | W (initial only) | W | — | — |
| `## Invariants` | W (initial only) | W | — | — |
| `## Assumptions` | W | W | W (append-only) | — |
| `## Contracts` | W (initial only) | W | — | — |
| `## Unconstrained` | W (initial only) | W | — | — |

The archaeologist writes the entire file from scratch (it had to
exist before any other role could operate on it), so it has full
write authority during extraction. After that, only the architect
modifies the spec-defining blocks. The implementer can only
append to `## Assumptions`, recording new heuristic choices
forced by the implementation work. The judge writes nothing.

The boundary is enforced socially, not mechanically. The dx
toolchain does not (yet) refuse a commit that violates it.
Reviewers — human or agent — are responsible for noticing when
an implementer's commit modifies an invariant, and treating that
as a procedural violation regardless of whether the change makes
the implementation conform.

## The handoff protocol

When work transitions between roles, the transition is announced
explicitly. The conventional form is a single line of the form:

```
HANDOFF: <from-role> → <to-role>: <one-sentence reason>
```

Examples:

```
HANDOFF: archaeologist → architect: base spec extracted; 7 invariants, 3 assumptions logged. Highest-risk assumption: "Default output format".

HANDOFF: architect → implementer: invariants stable; new "Performance: p99 latency" invariant added with a matching contract. Existing impl already satisfies it; please re-run contracts to confirm.

HANDOFF: implementer → judge: impl_python/ compiles and lints; logged 2 new assumptions ("Greeting format", "Cache location"). Expected pass.

HANDOFF: judge → architect: contract "Caches repeat queries" failed; classified as spec gap because the contract's **Then:** clause references internal cache state, not observable behavior. Please rephrase.
```

Why explicit handoffs matter:

- They make the workflow legible to anyone reviewing the
  transcript later. A reader who scrolls through a long
  agent session can find the role boundaries without reading
  the full conversation.
- They force the role transition to be a deliberate act. An
  agent that's about to "just keep working" past a role
  boundary stops to write the handoff line, which gives both
  the agent and any human observer a chance to catch
  role-bleed before it ships.
- They produce a search-grep-friendly artifact. `grep HANDOFF`
  over a session log gives the workflow's structure at a
  glance.

A handoff is not a permission gate; it's a notation. The next
role does not "accept" the handoff before proceeding. The
notation just records that the work has crossed a boundary.

## The verification loop

Every cycle of architect → implementer → judge runs the same
loop:

1. **Lint the spec.** `dx lint <file>.md` must exit 0. A spec
   that doesn't lint is structurally untrustworthy; fix that
   before doing anything else.
2. **Generate or modify the implementation.** The implementer
   reads the spec (and only the spec, when synthesizing fresh)
   and produces or edits code under `impl_<lang>/`.
3. **Build the implementation.** Whatever the host language's
   build is — `go build`, `cargo build`, `python -m build`,
   `make`. The build must succeed before the judge phase begins;
   a build failure is not a contract failure.
4. **Walk every contract.** The judge enumerates contracts via
   `dx contracts list <file>.md` and walks each one. For each:
   establish the precondition stated in `**Given:**`, trigger
   the action stated in `**When:**`, observe the outcome and
   compare against `**Then:**`. Record the verdict.
5. **Classify failures.** Any failed contract is classified per
   the rules below.
6. **Route the finding.** Each failure is routed to the role
   responsible for fixing it.

### Failure classification

The judge classifies every failed contract into one of three
kinds, in this order of preference (most specific first):

- **Spec gap.** The contract is ambiguous, contradicts an
  invariant, or under-specifies the situation enough that two
  honest implementers could read it differently and produce
  divergent behavior. The declaration is malformed; the
  architect must tighten or clarify it.

- **Intent mismatch.** The contract is internally inconsistent
  with another invariant or with `## Intent`. The spec
  contradicts itself; the architect must reconcile.

- **Implementation bug.** The contract is unambiguous, no
  invariant or other contract contradicts it, and the observed
  behavior simply doesn't match. The implementer must fix the
  code.

When in doubt between **implementation bug** and **spec gap**,
the judge defaults to **spec gap**. The cost of an incorrect
spec-gap call is one extra architect-implementer cycle; the
cost of an incorrect implementation-bug call is the implementer
rewriting working code to satisfy a malformed declaration. The
asymmetry favors caution.

### Why the judge writes nothing

The judge's only output is verdicts. The judge does not edit the
code (that's the implementer) and does not edit the spec (that's
the architect). A judge that "fixes" a contract failure mid-walk
loses the audit trail and creates a role-bleed problem: the
finding becomes invisible because the failure that motivated
the fix is no longer reproducible.

If the judge sees an obvious fix, the judge announces it as a
handoff to the implementer or architect, not as a commit.

## The post-merge ritual

When a declaration is touched on multiple branches and merged,
the result requires special handling. A textual three-way merge
can produce structurally invalid CommonMark (a `##` heading
duplicated across both sides; a `###` key that lost its
enclosing `##`), or it can produce syntactically valid output
that hides a *semantic* conflict — one branch demoted an
invariant to `## Unconstrained` while another tightened it; one
branch added a contract that contradicts another branch's
mutation of the same invariant.

The ritual, performed by the architect after every merge that
touches a declaration:

1. **Lint the merge result.** `dx lint <merged>.md` must exit
   0. If it doesn't, the textual merge produced structurally
   invalid output; fix the structural issues before proceeding.
2. **Diff against the merge base.** `dx diff <merge-base>.md
   <merged>.md` produces the semantic delta — every operation
   against the schema introduced by the merge. Read the delta;
   look for surprises.
3. **Reconcile semantic conflicts in the spec.** Any conflict
   visible in the diff is a spec-level conflict and must be
   resolved by editing the declaration. Per [SPEC §3.1](SPECIFICATION.md#31-declarations),
   the declaration is the source of truth; if the
   implementations disagree about what the spec means, the spec
   is at fault and must be made unambiguous.

The semantic delta is the architect's review surface. A
post-merge declaration that produces a clean delta (no
operations, or only the operations the architect intended) is
safe to commit. A delta that surfaces unintended `[MUTATED]` or
`[REMOVED]` operations is a sign that the textual merge
collapsed two intentional changes into one, and the architect
must restore both.

The dx language does not require this ritual ([SPEC §3.9](SPECIFICATION.md#39-spec-evolution)
deliberately leaves merge reconciliation to the host VCS plus
human judgment). It is the workflow's recommended discipline,
not a normative spec rule.

## When to deviate from the workflow

The four-role workflow is recommended, not required. The dx
language permits — and these scenarios are common — operating
outside it. The right deviation depends on the situation:

### Solo human authoring

A single human writes a declaration and implements it
themselves, with no agent in the loop. This is fully supported.
The behavioral disciplines that survive the simplification:

- The spec is still primary. If the implementation diverges from
  what's in the declaration, the spec wins.
- Heuristic choices made during implementation should still be
  recorded in `## Assumptions`. The reviewer (your future self,
  six months later) will thank you.
- Contracts still gate conformance. Manually walking the
  contracts before declaring done remains useful.

The roles aren't separate agents in this scenario; they're
phases of your own work. The handoff format is still useful as a
self-discipline tool — switching from "I'm writing the spec" to
"I'm writing the code" is worth noting in the commit message.

### One-shot agent runs

An agent receives a prompt, produces a declaration and an
implementation in one session, and is done. No multi-role
choreography. This is appropriate for small, scoped tasks
(adding a flag, generating a script). It is *inappropriate* for
work where the spec needs human ratification before code is
written, or where the spec will be long-lived and evolved by
multiple parties.

A one-shot run still benefits from the workflow's disciplines:
the agent should still produce a declaration *first*, log
assumptions *before* writing code, and walk the contracts
before declaring done. The roles collapse but the loop doesn't.

### Collapsed roles

A single agent acts as both architect and implementer in
adjacent turns. Common in iterative work where the human is
reviewing each turn anyway. Legitimate, with one caveat: the
agent must know which role it's playing in each turn. An agent
that's "just iterating" without distinguishing spec edits from
code edits will eventually edit an invariant to make a failing
contract pass, which is the cardinal sin of dx-managed work.

Use the handoff format internally — even within a single agent,
even in a single session — to mark the role transitions.

### Skipping the judge

Tempting for "obviously correct" changes. Almost always wrong.
The judge exists precisely because the things humans (and
agents) think are obviously correct are exactly the things that
silently break. Walk the contracts. Every time.

## Mapping to the skills

The [`skills/`](skills/) directory implements this workflow as
agent-loadable Markdown skills. Each role has its own skill that
encodes the role's discipline in detail, including anti-patterns
and worked examples; the meta-router skill handles role
transitions; two reference skills support every role.

| Skill | Role | When loaded |
|---|---|---|
| [`dx-orchestrator`](skills/dx-orchestrator/SKILL.md) | Meta / router | Always, on entering a dx-managed repo. Routes work to the right role-skill. |
| [`dx-authoring`](skills/dx-authoring/SKILL.md) | Reference | Whenever a declaration is being written or modified. |
| [`dx-toolchain`](skills/dx-toolchain/SKILL.md) | Reference | Whenever the `dx` CLI is being invoked from an agent loop. |
| [`archaeologist`](skills/archaeologist/SKILL.md) | Archaeologist | Distilling existing code into a declaration. |
| [`architect`](skills/architect/SKILL.md) | Architect | Authoring or refining a declaration. |
| [`implementer`](skills/implementer/SKILL.md) | Implementer | Generating or modifying code from a declaration. |
| [`judge`](skills/judge/SKILL.md) | Judge | Verifying an implementation against contracts. |

The skills enforce this workflow's disciplines in practice; the
workflow document explains *why* they exist. A reader who wants
the operational details — anti-pattern lists, specific
prompting patterns, exit-code interpretations — should read the
relevant skill. A reader who wants the structural overview is
reading it.

## Mapping to the journeys

Every user journey under [`docs/journeys/`](docs/journeys/) follows
this workflow. Each journey is the four-role loop applied to a
specific task; the role-by-role disciplines below appear in each
journey at their natural phase.

| Journey | What it covers |
|---|---|
| [Greenfield development](docs/journeys/greenfield-development.md) | Architect → Implementer → Judge. No archaeologist (no existing code). |
| [Add a feature](docs/journeys/add-a-feature.md) | Architect → Implementer → Judge, with the architect mutating an existing spec. |
| [Add a feature to multiple implementations](docs/journeys/add-a-feature-to-multiple-implementations.md) | Architect → N parallel Implementers → Judge over an N×M grid. |
| [Port a program to another language](docs/journeys/port-to-another-language.md) | Archaeologist → Architect → Implementer → Judge. The full loop. |

Read the journeys for end-to-end walkthroughs of the workflow
applied to real tasks. Read this document for the underlying
structure that makes those journeys work.
