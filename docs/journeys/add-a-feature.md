# Journey: Add a Feature to an Existing Program

**Goal:** You have a working `.dx`-managed system (a `system.dx` plus
at least one `impl_<lang>/`) and you want to add a new capability —
a CLI flag, a new RPC endpoint, a behavior change. You want the
addition to land in the spec *first*, so the implementation is
checkable against it, and the change is visible as a clean
[`PROMOTED]`/`[ADDED]`/`[MUTATED]` semantic diff rather than a wall of
red and green YAML.

**Time budget (rough):** 10–30 minutes for a small, well-scoped
addition. Spec changes are tight; implementer changes are usually a
single edit.

**Prerequisites:**

- A `declare`-managed project: `system.dx` + `impl_<lang>/` + a clean
  git workspace.
- The `declare` CLI on `$PATH` and the seven `declare` skills
  installed in your agent runtime. See
  [port §0](port-to-another-language.md#0-one-time-setup) for setup.
- A clear feature request, even if rough. ("Add a `--dry-run` flag",
  "Persist data to JSON instead of CSV", "Make the API return a
  request ID header.")

## TL;DR

```
architect mutates system.dx          ← add invariant + contract for the feature
   ↓
declare diff HEAD:system.dx system.dx ← human reviews the semantic delta
   ↓
implementer edits impl_<lang>/        ← reads the diff; updates code to match
   ↓
judge runs all contracts              ← old + new; nothing regressed
   ↓
done
```

The journey's distinctive shape: it's mostly a **diff-driven** flow.
The architect's mutation produces a small, focused
`declare diff` ledger; the implementer reads that ledger to know
exactly what to change; the judge re-runs every contract (old and
new) to catch regressions.

## 0. Setup

Identical to [port §0](port-to-another-language.md#0-one-time-setup).
If you've already walked one journey on this project, you're set.

## 1. Pre-flight

Before changing anything, confirm the existing state is clean:

```bash
declare lint system.dx                                      # exit 0
cd impl_<lang> && <build command> && <test command>         # all green
```

If lint fails, the spec is structurally untrustworthy — fix that
first. If the existing tests don't pass, you can't tell whether the
new feature broke them or they were already broken; sort that out
before adding anything new.

Note the current commit; you'll diff against it later:

```bash
git rev-parse HEAD                                          # remember this SHA
```

## 2. Architect phase: mutate the spec

Load the [`architect`](../../skills/architect/SKILL.md) skill. The
architect's job here is **smaller** than in the greenfield journey —
they're editing a known-good spec, not authoring one. But the
discipline is the same: every change happens in the spec first, with
a matching contract.

### Turn 1: propose the changes

> "I want to add `<describe the feature>`. Per the architect skill,
> propose the smallest set of changes to `system.dx` that captures
> the new capability: typically one or two new `invariants:` entries
> and at least one new `contracts:` entry. Output the proposed
> changes as prose; do not edit the file yet."

Read the proposal critically. Common patterns to watch for:

- **One invariant, one contract.** Usually correct. The invariant
  describes the new observable behavior; the contract proves it.
- **Multiple invariants.** Decompose carefully — each invariant must
  survive the pruning question ("would relaxing this change anything
  observable?"). A feature that needs five invariants is usually
  three or four "implementation-flavored" invariants in disguise.
- **No contract.** Push back. If the new behavior isn't testable as
  a black box, it doesn't belong in `invariants:` — it's either a
  documentation note or an over-specified implementation detail.
- **Mutating an existing invariant.** Rare and serious. A mutated
  invariant means existing behavior is changing, which means existing
  contracts may now need updates too. Confirm both.

### Turn 2: apply the changes

> "Edit `system.dx` to apply the changes you proposed: <recap or
> adjustments here>. Then run `declare lint system.dx` and
> `declare diff HEAD:system.dx system.dx` and report both results."

The `declare diff` output is your review surface. A focused feature
addition should produce something like:

```
[ADDED] invariants.iface_dry_run_flag
[ADDED] contracts.dry_run_does_not_mutate.given
[ADDED] contracts.dry_run_does_not_mutate.then
[ADDED] contracts.dry_run_does_not_mutate.when
```

If the diff is much larger than that, the architect is doing too much
in one step. Stop, ask why, and either narrow the scope or split into
multiple commits.

### Turn 3 (optional): ratify any new assumptions

If the architect logged any new `assumptions:` entries while shaping
the invariants, decide whether to promote, demote, or reject each one
before handing off to the implementer. The same three-turn pattern
from
[port §3](port-to-another-language.md#3-architect-phase-ratify-and-prune-the-spec)
applies; usually for a small feature the assumption set is small or
empty.

Commit:

```bash
declare lint system.dx
git add system.dx && git commit -m "Architect: add <feature> to spec"
git rev-parse HEAD                    # the spec-only commit; useful later
```

The spec-only commit is a real handoff artifact: the implementer's
job in step 3 is to make the implementation catch up to it.

## 3. Implementer phase: update the code

Open a fresh agent session if you can. Load the
[`implementer`](../../skills/implementer/SKILL.md) skill.

The key prompt difference from the port and greenfield journeys: the
implementer here is *editing*, not *generating*. Give them the diff
explicitly:

> "Read `system.dx` for context, then read the output of
> `declare diff HEAD~1:system.dx system.dx` to see what changed in
> the spec. Update the code under `impl_<lang>/` to satisfy the new
> invariants and contracts. Do not refactor unrelated code. Re-run
> the build and any existing tests; both must continue to pass."

(`HEAD~1` here points at the previous commit — i.e., before the
architect's spec mutation. Adjust if your project has different commit
granularity.)

The "do not refactor unrelated code" instruction is doing real work.
A diligent agent will see opportunities to "clean up while I'm here"
and silently change behavior the contracts don't cover. Refactors and
features are different journeys; mixing them defeats the purpose of
the focused diff.

If the implementer logs new `assumptions:` while editing, treat them
the same way as in any journey:

- **Cheap and fast:** accept and move on; the architect ratifies in
  a follow-up.
- **Strict:** loop back to step 2 to ratify before re-running the
  implementer.

After the implementer finishes:

```bash
cd impl_<lang> && <build command> && <test command>        # both must pass
declare lint system.dx
declare diff HEAD:system.dx system.dx                      # see any new assumptions
git add . && git commit -m "Implementer: <feature> in impl_<lang>"
```

## 4. Judge phase: re-verify everything

Adding a feature is precisely the situation in which **regression
contracts matter most**. The judge here doesn't just verify the new
contract — they re-run *every* contract, because the implementer's
edit may have silently broken existing behavior.

Load the [`judge`](../../skills/judge/SKILL.md) skill and walk the
full contract set:

```bash
declare contracts list system.dx
```

For each contract, set up the `given`, trigger the `when`, evaluate
the `then`. Tag each verdict explicitly:

- **PASS (existing)** — old contract, still passes. Most should be
  here; if not, you have a regression.
- **PASS (new)** — added by this feature. Should always be here for
  the contracts the architect just added.
- **FAIL** — classify per the
  [judge skill's classification heuristics](../../skills/judge/SKILL.md#4-classification-heuristics).

A FAIL on a new contract usually means the implementer didn't fully
satisfy the spec; route to implementer.

A FAIL on an old contract is a **regression** — the new feature broke
something. This is the most expensive failure mode and the reason the
journey insists on re-running the full contract set. Route to
implementer with a sharp note: "old contract X failed after your
feature edit; restore the prior behavior while keeping the new
feature."

A spec-gap classification on a new contract usually means the
architect over-promised — the contract is unverifiable as written.
Route to architect.

## 5. Done — what you have now

- `system.dx` — extended with the new feature, still byte-stable
  under `declare fmt`.
- `impl_<lang>/` — updated to satisfy the new contracts, with all
  prior contracts still passing.
- A two-commit pair (architect → implementer) that reads as a clean
  feature addition. `declare diff <pre-architect>:system.dx system.dx`
  shows exactly what changed at the intent level.

## Anti-patterns (don't do)

### Skipping the architect phase

Tempting for "small" features ("it's just a flag, why update the
spec?"). The result is implementation drift: the code does something
the spec doesn't sanction, future agents have no idea why, and the
contracts don't cover it. Always: spec first, code second.

### Conflating refactor and feature

If the implementer also "fixed" code unrelated to the new feature,
your `declare diff` looks clean but your `git diff` is chaotic. A
reviewer can't tell which lines are the feature. If the agent wants
to refactor, do that as its own commit (and ideally its own journey
turn) before or after the feature.

### Editing the contract because the implementation doesn't satisfy it

This is the cardinal sin of `declare`-managed work. The contract is
the architect's commitment; the implementation is the implementer's
attempt. If they disagree, the contract leads — unless the architect
explicitly decides the contract was wrong, in which case the
contract change is its own architect-skill commit, not a stealth edit
during implementation.

### Re-running only the new contracts

The single biggest reason to use `declare contracts list` is so the
judge runs *all* of them, every time. A judge who skips
"obviously-still-true" old contracts is exactly how regressions ship.

## Known gaps in this journey

These reuse the gap catalog from
[port §"Known gaps"](port-to-another-language.md#known-gaps-in-this-journey-priority-todos);
the same v0.1.0 limitations apply. Two are particularly acute:

- **Gap 1 (no `declare verify`)** is felt every time you add a
  feature, because the regression-vs-new-feature contract sweep is
  the largest manual workload in the journey. A future
  `declare verify` would re-run all contracts deterministically and
  flag regressions as a first-class result type.
- **No reason field on invariants/contracts.** When you mutate an
  existing invariant for a feature, future architects won't know
  *why*. The
  [SPEC §6](../../SPEC.md#6-reserved-field-names-future-compatibility)
  reserved `reason:` field exists exactly for this; it's not
  expressible in v0.1.0. Until then, the git commit message is the
  audit trail.

## Worked example

> **TODO:** Walk this journey end-to-end in a clean room (per the
> testing methodology used for
> [port-to-another-language](port-to-another-language.md)) by adding
> a `--dry-run` flag (or a similar small feature) to the
> [weather_cli example](../../examples/weather_cli/) and link the
> resulting commit pair here.

## Related reading

- [`AGENTS.md`](../../AGENTS.md) — universal rules; §3 (Verification
  Loop) and §6 (post-merge ritual) are particularly relevant for
  feature work.
- [`skills/architect/SKILL.md`](../../skills/architect/SKILL.md) §3a
  (adding a new invariant), §3c (adding a contract), §3d
  (restructuring): the operational skills this journey leans on.
- [`skills/implementer/SKILL.md`](../../skills/implementer/SKILL.md)
  — particularly the boundary against `intent`/`invariants`/
  `contracts` modification; the implementer can only append to
  `assumptions:`.
- [add-a-feature-to-multiple-implementations](add-a-feature-to-multiple-implementations.md)
  — the next journey if your spec governs more than one
  `impl_<lang>/`.
