# Journey: Greenfield Development

**Goal:** You have an idea — a vague paragraph of prose — and you want
a working implementation. Instead of jumping straight to code (where
the agent will silently invent the parts you didn't specify), you
*iterate the spec first* until you and the agent agree on what's being
built. Then the code falls out in one shot.

**Time budget (rough):** 20–60 minutes for a CLI; a few hours for
something with state, network, or auth. Most of that time is
spec-iteration; once the spec is steady, the implementation phase is
fast.

**Prerequisites:**

- A working installation of the `dx` CLI on `$PATH`. See the
  [README install section](../../README.md#install) for build
  instructions.
- A coding agent with file-system tools (read, write, run shell
  commands) and the seven `dx` skills installed. See the
  [port-to-another-language journey §0](port-to-another-language.md#0-one-time-setup)
  for runtime-specific install steps and the headless-mode caveats.
- A clean git workspace.

## TL;DR

```
human + agent draft v0 system.md     ← architect skill, with the human as co-author
   ↓ (iterate, prune, ratify)
spec settles                          ← when `dx diff` produces zero ops between rounds
   ↓
agent one-shots impl_<lang>/         ← implementer skill, reading only system.md
   ↓
judge runs every contract            ← judge skill
   ↓
done
```

This journey is the **inverse** of
[port-to-another-language](port-to-another-language.md): there is no
existing source, so no archaeologist phase. The architect and the
human together author the spec from prose; the implementer reads only
the spec. The journey's value is in the iteration loop *before* the
first line of code.

## 0. Setup

Identical to [port §0](port-to-another-language.md#0-one-time-setup):
install the `dx` CLI, install the seven skills in your agent
runtime of choice, and (if you're driving headlessly) make sure your
agent's auto-approve and trust flags are set correctly.

## 1. Prepare the workspace

```bash
mkdir my-thing && cd my-thing && git init -q -b main
```

You'll add `system.md` in step 2 and `impl_<lang>/` in step 4. Commit
between every phase so the git history reads like a design diary.

## 2. Architect phase: draft v0 with the human

Load the [`architect`](../../skills/architect/SKILL.md) skill. Unlike
the port journey, the architect here is *authoring*, not refining. The
human supplies the prose; the architect translates it into a
`system.md` and surfaces every implicit choice as either an explicit
invariant, an `assumptions:` entry, or an `unconstrained:` entry.

### Turn 1: write the prose

Tell the agent your idea in plain English. Don't try to be
comprehensive — describe what *you* care about; the agent's job is to
notice what you didn't say. Examples:

> "I want a command-line app that watches my Downloads folder and
> moves screenshots into ~/Pictures/Screenshots, organized by date."

> "I want a small HTTP service that serves a JSON 'haiku of the day'
> at GET /haiku, with a fixed list of haikus rotating daily by date."

> "I want a script that takes a CSV of customer emails and produces a
> Markdown report of who's churned in the last 30 days."

### Turn 2: have the agent draft the spec

> "Draft a `system.md` for the idea above per the architect skill.
> Surface every choice you had to make — output format, error
> handling, file locations, performance expectations — as either an
> invariant (we're committing to it), an assumption (we'll ratify it
> later), or an `## Unconstrained` entry (we explicitly don't care).
> Run `dx lint system.md` when done."

The first draft should over-produce assumptions. That's correct
behavior — the architect is showing you every gap in your prose.

### Turn 3+: iterate

For each `## Assumptions` entry, decide:

- **Promote** to `## Invariants` — yes, I care about this; commit.
- **Demote** to `## Unconstrained` — I don't care; the implementer
  picks.
- **Reject** — the assumption is wrong; rewrite the relevant
  invariant or intent so the assumption is unnecessary.
- **Defer** — leave it for the implementer to handle, knowing they
  may surface a different choice.

Use the three-turn pattern from
[port §3](port-to-another-language.md#3-architect-phase-ratify-and-prune-the-spec):
ask the agent for recommendations first, then explicitly tell it
what to apply. After every round of edits:

```bash
dx lint system.md                     # must exit 0
dx diff HEAD:system.md system.md      # see what you changed
git add system.md && git commit -m "Architect: <describe the change>"
```

### When is the spec done?

You're done iterating when **two consecutive `dx diff` invocations
produce zero output** — meaning the architect's last round of changes
were a no-op because the spec had already converged. In practice this
happens when:

- Every `## Assumptions` entry is one you *consciously* want the
  implementer to handle (or is empty).
- Every `## Invariants` entry survives the pruning question: "would
  relaxing this change anything observable?"
- Every `## Contracts` entry is testable as a black box (no internal
  state references) and exists for a load-bearing reason.
- The body of `## Intent` (whether a single sentence or the
  first item of a priority list) is something you'd put on a
  project card — not a paragraph trying to anticipate the spec.

This convergence test is the journey's main quality gate. Don't skip
it: an under-iterated spec produces an under-specified implementation,
and you'll be debugging both at once.

## 3. Implementer phase: one-shot the code

Open a fresh agent session if you can. Load the
[`implementer`](../../skills/implementer/SKILL.md) skill and prompt:

> "Read only `system.md`. Generate a complete implementation in
> `<target_language>` under `impl_<target_lang>/` that satisfies every
> entry in `## Invariants` and every contract in `## Contracts`. Use
> the language's native idioms. When the spec is ambiguous, append a
> `### <id>` section to `## Assumptions` in `system.md` *before*
> writing the code that makes the assumption."

Because you iterated the spec to convergence, the implementer should
have very little to add to `## Assumptions`. If they add more than
two or three, that's a signal you over-trusted the convergence and a
real spec gap leaked through; loop back to step 2 to ratify each new
assumption.

After the implementer finishes:

```bash
cd impl_<target_lang> && <build command>   # must succeed
dx lint system.md                     # implementer may have appended assumptions
dx diff HEAD:system.md system.md      # see what they were
git add . && git commit -m "Implementer: generate impl_<target_lang>"
```

## 4. Judge phase: verify against the contracts

Identical to [port §5](port-to-another-language.md#5-judge-phase-verify-against-the-contracts):

```bash
dx contracts list system.md           # enumerate
# for each contract: set up given, trigger when, evaluate then
```

Load the [`judge`](../../skills/judge/SKILL.md) skill and walk every
contract. Classify any failure per the judge's rules.

For greenfield work, **most failures are spec gaps, not impl bugs.**
A new implementation against a fresh spec is the situation in which
the architect's blind spots are most likely to bite. Lean toward
"spec gap" classifications when in doubt.

## 5. Done — what you have now

- `system.md` — your spec, iterated to convergence and machine-validated.
- `impl_<lang>/` — a working implementation that satisfies every
  contract.
- A git history that reads as: prose → v0 spec → iteration rounds →
  implementer commit → contract-passing judgment.

From here, [add-a-feature](add-a-feature.md) is the journey for
extending it. If you ever want it in another language too, run
[port-to-another-language](port-to-another-language.md) using your
`impl_<lang>/` as the legacy artifact.

## Anti-patterns (don't do)

These are the failure modes specific to greenfield work. The
[`architect` skill's anti-pattern list](../../skills/architect/SKILL.md#7-anti-patterns)
covers the universal cases.

### "Let me just sketch the code first"

The whole journey rests on iterating the spec *before* implementation.
Once you have working code, the spec becomes "what the code does"
instead of "what we agreed should be true." Resist the urge to write
even a stub; the architect will produce concrete `## Intent`,
`## Invariants`, and contract examples in turn 2 if you let it.

### One-shot the spec

The architect's first draft is a strawman, not a finished spec. If you
ratify it without iteration you've reproduced the original "vibe-code
from a vague prompt" failure mode, just one indirection deeper.
Convergence (two consecutive diffs of zero ops) is the test.

### Specifying the implementation

The `## Intent` body says *what* and *why*. If you find yourself
writing "use Postgres" or "implement with goroutines", that's an
implementation choice — either move it to `## Unconstrained` (with
a note) or drop it. Over-specification is a defect; see
[AGENTS.md §4](../../AGENTS.md#4-pruning-and-parsimony).

### Skipping `## Contracts` because "the invariants speak for themselves"

Invariants tell the implementer what's true; contracts tell the judge
how to *check* what's true. A spec without contracts is a spec without
a verification story — and verification is the whole point of the
loop. Aim for at least one contract per `Interface:`-categorized invariant.

## Known gaps in this journey (priority TODOs)

These reuse the gap catalog from
[port §"Known gaps"](port-to-another-language.md#known-gaps-in-this-journey-priority-todos);
the same v0.1.0 limitations apply. Two are particularly acute for
greenfield work:

- **Gap 1 (no `dx verify`)** bites harder here than in the port
  journey because there is no parallel implementation to cross-check
  against. The judge has to walk every contract on a brand-new
  codebase by hand.
- **Spec convergence is human-mediated.** The "two consecutive
  zero-ops diffs" rule is in this doc, not in the toolchain. A future
  `dx review` or `dx suggest` command could make convergence
  more rigorous.

## Worked example

> **TODO:** Walk this journey end-to-end in a clean room (per the
> testing methodology used for
> [port-to-another-language](port-to-another-language.md)) and link
> the resulting `examples/<name>/` here.

## Related reading

- [`AGENTS.md`](../../AGENTS.md) — universal rules every contributor
  follows in a dx-managed repo.
- [`SPECIFICATION.md`](../../SPECIFICATION.md) — normative dx reference.
- [`skills/architect/SKILL.md`](../../skills/architect/SKILL.md) — the
  central skill for this journey; do this with the human as
  co-architect, not as a passive reviewer.
- [`skills/dx-authoring/SKILL.md`](../../skills/dx-authoring/SKILL.md)
  — the dense dx language reference your agent should consult.
- [add-a-feature](add-a-feature.md) — the natural follow-up journey
  once you have a working spec + implementation.
