# Journey: Add a Feature to Multiple Implementations

**Goal:** You have a `.dx`-managed library (or service, or CLI) with
two or more implementations in different languages — Google's
[Agent Development Kit](https://google.github.io/adk-docs/) is the
canonical real-world example, with Python, Java, Go, and TypeScript
SDKs that must remain observably equivalent. You want to add a single
feature *once*, in the spec, and have every implementation pick it up
in a way that keeps cross-language behavior identical.

**Time budget (rough):** 30–60 minutes for the architect step (unchanged
from a single-impl feature add); then roughly *N* × the per-impl
implementer time, where *N* is the number of languages. Multi-impl
work scales linearly in implementer effort but the architect and
judge phases stay close to constant.

**Prerequisites:**

- A `declare`-managed project with a `system.dx` and **two or more**
  `impl_<lang>/` subtrees, each independently building and passing
  every contract today. (If you don't have this shape yet, you
  probably want
  [add-a-feature](add-a-feature.md) first; if you have one impl and
  want a second, walk
  [port-to-another-language](port-to-another-language.md) to get
  there.)
- The `declare` CLI on `$PATH` and the seven `declare` skills
  installed in your agent runtime.

## TL;DR

```
architect mutates system.dx               ← single source of truth, one commit
   ↓
declare diff HEAD:system.dx system.dx     ← human reviews; same diff for all impls
   ↓
implementer A → impl_<lang_A>/   ┐
implementer B → impl_<lang_B>/   ┤  N parallel sessions, each forbidden from
implementer C → impl_<lang_C>/   ┤  reading the others
                                 ┘
   ↓
judge runs all contracts on every impl    ← cross-language observable equivalence
   ↓
done
```

This journey is structurally [add-a-feature](add-a-feature.md) with a
**fork-join** at the implementer phase: one architect commit, then
N parallel implementer sessions (one per language, each blind to the
others), then a judge that runs the same contracts against every
implementation. The point of the journey is to keep the N
implementations from drifting in observable behavior.

## 0. Setup

Identical to [port §0](port-to-another-language.md#0-one-time-setup)
plus one workspace convention:

For a multi-impl project, the canonical layout is:

```
project/
├── system.dx                ← single source of truth
├── impl_python/
├── impl_go/
├── impl_typescript/
└── impl_<other>/
```

Every implementation lives at the same depth under the project root.
The architect's commit is one file (`system.dx`); each implementer's
commit is scoped to a single `impl_<lang>/` subtree. This separation
is what makes the per-impl no-peeking discipline enforceable by
convention (and tractable by code-review tools).

## 1. Pre-flight

Confirm every implementation is currently green:

```bash
declare lint system.dx                       # exit 0 required

# For each language:
cd impl_<lang> && <build command> && <test command>
```

Then walk every contract against every implementation to establish a
clean baseline. The
[judge skill](../../skills/judge/SKILL.md)'s pre-flight covers this;
the multi-impl extension is "do it for each `impl_*/`."

```bash
# Capture the contract IDs once:
declare contracts list system.dx

# Then for each <lang>: walk every contract, confirm PASS.
```

If any implementation fails any contract *before* you start the
feature, fix that first — adding a feature on top of a baseline drift
makes the cause of any later failure ambiguous.

## 2. Architect phase: mutate the spec (once)

Identical to
[add-a-feature §2](add-a-feature.md#2-architect-phase-mutate-the-spec).
The architect's job is the same regardless of how many implementations
the spec governs: produce the smallest set of `invariants:` and
`contracts:` changes that captures the new capability.

The crucial discipline: **the architect must not phrase the invariant
in language-specific terms.** A new invariant for an ADK-shaped
project that says "the Python SDK exposes a `with_session()` context
manager" is wrong — `with` is a Python idiom that has no equivalent in
Go or Java. The right phrasing is something like "a session may be
opened, used, and guaranteed to close on completion or error" — which
each language implements idiomatically (`with` in Python, `defer
session.Close()` in Go, try-with-resources in Java).

Run the same three-turn architect pattern from
[port §3](port-to-another-language.md#3-architect-phase-ratify-and-prune-the-spec).
Commit the result as a single architect-only commit; this becomes the
shared input for every implementer in step 3.

```bash
declare lint system.dx
declare diff HEAD:system.dx system.dx        # tight, focused ledger
git add system.dx && git commit -m "Architect: add <feature> to spec"
```

## 3. Implementer phase: N parallel sessions

This is where multi-impl work diverges from
[add-a-feature](add-a-feature.md). For each language, open a
**separate** agent session and prompt it to update *only* its
`impl_<lang>/`. Critically, each session must not read the other
implementations.

### Why the per-language no-peeking rule matters

If implementer A peeks at implementer B's existing code while adding
the new feature, A's output silently inherits B's idioms — and the
"observable equivalence across languages" property collapses into "we
all converge on whichever language the agent looked at first."
Worse: the spec could be ambiguous in a way that B's previous
implementation accidentally resolved one way; A copies that
resolution; nobody notices the spec is under-specified until a third
implementation surfaces a different reading.

The discipline:

- **One agent session per language.** Don't reuse a session across
  languages.
- **Restrict the session's working directory** to `impl_<lang>/` (and
  read-only access to `system.dx`). If your runtime has an allowlist,
  use it; if not, instruct the agent explicitly.
- **Add a workspace-level pattern** like a `.declare-implementer-allowlist`
  convention if you find yourself doing many of these (the
  no-peeking-enforcement design is on the
  [ROADMAP](../../ROADMAP.md) for v0.2; until then it's
  honor-system).

### Per-language prompt template

For each `impl_<lang>/`, in a fresh session:

> "Read `system.dx` for context, then read the output of
> `declare diff HEAD~1:system.dx system.dx` to see what changed in
> the spec. Update the code under `impl_<lang>/` to satisfy the new
> invariants and contracts. Do **not** read or reference any other
> `impl_*/` directory; pretend they don't exist. Use the language's
> native idioms. Re-run the build and existing tests; both must
> continue to pass."

After each implementer finishes, commit per-language:

```bash
cd impl_<lang> && <build> && <test>
git add impl_<lang>
git commit -m "Implementer (<lang>): <feature>"
```

You'll end up with one architect commit + N implementer commits.
Reviewing the per-implementer diffs side by side is itself useful: if
they diverge a lot in *what they implemented* (not how — different
languages should look different), that's a smell that the spec was
ambiguous.

### Handling new assumptions

Each implementer may log new `assumptions:` to `system.dx` while
working. Two complications unique to multi-impl:

- **Two implementers may log conflicting assumptions** about the same
  ambiguity. Implementer A says "we assume the timeout is 30s";
  implementer B says "we assume there is no timeout." This is a
  spec-gap signal of the highest order — route both back to the
  architect, who must reconcile and re-commit before either
  implementer proceeds.
- **One implementer may log an assumption the others didn't need.**
  Often means that implementer's language pushed a default the others
  inherited from elsewhere. Ratify it (probably as an invariant) so
  the others know the spec just tightened.

Use `declare diff HEAD:system.dx system.dx` after each implementer
session to surface the deltas:

```bash
declare diff HEAD:system.dx system.dx
# [ADDED] assumptions.timeout.policy   ← implementer A's guess
# [ADDED] assumptions.retry.behavior   ← implementer A's guess
```

Pause the parallel work, drive an architect ratification round, then
let the remaining implementers proceed against the tightened spec.

## 4. Judge phase: full-grid verification

The judge's matrix grows from 1×N (contracts × one impl) to N×M
(contracts × M implementations). Use `declare contracts list` to
drive a full grid:

```bash
declare contracts list system.dx        # one contract per row
# Languages are the columns: impl_python, impl_go, impl_typescript, ...
```

Walk every cell. The expected outcome is a fully PASS grid; any FAIL
gets classified per the judge skill, with two multi-impl-specific
patterns to recognize:

### Same contract fails in *one* implementation

Almost always an **implementation bug** in that one. The other
implementations passing the same contract means the spec is
unambiguous and the lone-failure language got it wrong.

→ Route to that language's implementer. Specific prompt: "Contract
X passes in `impl_<lang_A>/` but fails in `impl_<lang_B>/`; here's the
observed behavior in B; please fix B."

Don't share the *code* of A — that defeats the no-peeking rule. Share
the *observation*.

### Same contract fails in *all* implementations

Almost always a **spec gap**. If every implementer interpreted the
contract the same way and every implementer got it wrong, the
contract is ambiguous or contradicts an invariant.

→ Route to the architect. Multi-impl projects make this failure mode
much more visible than single-impl projects do; in single-impl work
you can't tell whether the problem is the code or the spec.

### Same contract passes in some but fails in others (with different observations)

The most interesting case. The spec is **systematically ambiguous** —
different implementers read it differently, and the differences
manifest at the contract boundary. This is the classic ADK-shaped
problem (e.g., one SDK serializes timestamps as ISO 8601, another as
Unix epoch; the contract said "the timestamp field" without pinning
the format).

→ Route to architect with all the observations. The fix is usually
a tightened invariant + a more specific contract, after which all
implementers re-update.

## 5. Done — what you have now

- One architect commit on `system.dx` (the canonical change ledger).
- N implementer commits, each scoped to one `impl_<lang>/`.
- A green N×M judgment grid: every contract passes against every
  implementation.

The git history reads as: spec change (one commit) → N implementer
commits (any order) → judge verdicts. `declare diff` between the
pre-architect commit and HEAD shows the *whole* feature at the spec
level in one ledger, regardless of how many languages it touched.

## Anti-patterns (don't do)

### One agent session, sequential per-language updates

Tempting because it's faster than spinning up N sessions. The
result: implementer 2 reads implementer 1's code in the prior turn
of the same session, and the no-peeking property is silently
violated. If you only have one agent runtime instance available, run
the per-language sessions strictly sequentially, in *fresh* sessions,
and give each one only its own `impl_<lang>/` and `system.dx`.

### Letting implementers see each other's work for "consistency"

This is the multi-impl version of the
[port journey's no-peeking gap](port-to-another-language.md#gap-2--no-mechanism-to-enforce-implementer-must-not-read-the-source-medium-priority).
The "consistency" you'd get is fake consistency: it would mean every
implementation converged on whichever language was looked at first.
Real consistency is what the judge confirms by passing the same
contracts against every implementation.

### Skipping the full N×M judge grid

If you only re-judge the languages you "think" you might have
broken, you'll miss the regressions in the languages you didn't
touch — which can absolutely happen, because spec changes apply to
all languages even when only some implementers updated their code.
Always walk the full grid.

### A "language-specific" invariant

`iface_python_with_statement_supported` is wrong. So is
`iface_typescript_promise_returned`. Invariants are observable
behaviors of the system, not language-idiom prescriptions. If you
find yourself naming a language in an invariant, restate it in
behavior terms ("a session resource is opened, used, and released
deterministically on completion or error") and let each implementer
pick the idiom.

### Architect changes that aren't atomic

A `system.dx` commit that adds invariant A but only *partially*
specifies the contract for it puts implementers in different states:
the fast ones implement A and pass the partial contract; the slow
ones see a half-cooked spec and stall. Spec changes should land as
single, atomic commits that pass `declare lint` and represent a
complete addition.

## Known gaps in this journey

These reuse the gap catalog from
[port §"Known gaps"](port-to-another-language.md#known-gaps-in-this-journey-priority-todos);
the same v0.1.0 limitations apply. Three are especially acute for
multi-impl work:

- **Gap 1 (no `declare verify`)** is most painful here. Walking an
  N×M judge grid by hand for a real ADK-sized project is
  infeasible; this is the journey that most needs `declare verify`
  to ship. v0.2 design priority.
- **Gap 2 (no-peeking enforcement)** is most consequential here.
  In single-impl work, peeking at the original collapses the
  language-portability property; in multi-impl work it collapses
  the cross-language consistency property, which is the entire
  point of the journey.
- **No "test the whole grid" CLI verb.** Even before
  `declare verify` ships, a thin convenience like
  `declare contracts list --json` plus a project-supplied test
  runner would make the grid walk less manual. The
  [`declare contracts list`](../../skills/declare-toolchain/SKILL.md)
  command is the foundation; the runner is project-specific until
  v0.2.

## Worked example

> **TODO:** Walk this journey end-to-end in a clean room (per the
> testing methodology used for
> [port-to-another-language](port-to-another-language.md)) using a
> small library with two or three language implementations, and link
> the resulting `examples/<name>/` here.
>
> A natural candidate would be a tiny "haiku of the day" service
> implemented in Python, Go, and TypeScript: small enough to walk in
> one session, structured enough to expose cross-language ambiguity
> (date handling, timezone semantics, JSON shape) at the judge step.

## Related reading

- [add-a-feature](add-a-feature.md) — the single-impl version of
  this journey; the architect step is identical and shares the
  diff-driven discipline.
- [port-to-another-language](port-to-another-language.md) — how to
  *get* a second implementation in the first place if you only have
  one today.
- [`AGENTS.md`](../../AGENTS.md) — universal rules; §1 (primacy of
  the declaration) is the load-bearing rule for multi-impl work.
- [`skills/architect/SKILL.md`](../../skills/architect/SKILL.md) §2b
  ("Black-box statements only"): the discipline that prevents
  language-specific invariants from sneaking in.
- [`skills/judge/SKILL.md`](../../skills/judge/SKILL.md) — the
  classification heuristics; cross-implementation symmetry is your
  best signal for the spec-gap-vs-impl-bug call.
