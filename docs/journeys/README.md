# `dx` user journeys

End-to-end walkthroughs for using `dx` to accomplish real
tasks. Each journey is written for a developer who has the `dx`
CLI installed and a coding agent of their choice; instructions are
agent-agnostic with at least one concrete example (typically Claude
Code or Gemini CLI).

| Journey | What it covers | Status |
| ------- | -------------- | ------ |
| [Greenfield development](greenfield-development.md) | Start from a vague human idea, iterate `system.dx` with an agent until the spec converges, then one-shot the first implementation. The inverse of the port journey: no archaeologist phase, no existing source. | Documented; worked example pending clean-room test. |
| [Add a feature to an existing program](add-a-feature.md) | Add a new capability to a `.dx`-managed system. Spec changes first; the implementer reads the resulting `dx diff` to know exactly what to update; the judge re-runs *every* contract to catch regressions. | Documented; worked example pending clean-room test. |
| [Add a feature to multiple implementations](add-a-feature-to-multiple-implementations.md) | The library / SDK case: one `system.dx` governs N `impl_<lang>/` subtrees (e.g., Python + Go + TypeScript). One architect commit; N parallel implementer sessions, each blind to the others; one N×M judge grid that catches cross-language drift. | Documented; worked example pending clean-room test. |
| [Port a program to another language](port-to-another-language.md) | Reverse-engineer an existing implementation into a `.dx` spec, then synthesize an equivalent implementation in a new language without ever reading the original source. | Documented and clean-room-validated end-to-end. |

## How the journeys relate

The four journeys are not isolated. They form a small lattice of
related tasks; one often leads naturally to another:

```
greenfield ──→ add-a-feature ──→ add-a-feature-to-multiple-implementations
                                   ↑
                             port-to-another-language
                            (gets you the second impl)
```

- **Start with [greenfield](greenfield-development.md)** if you don't
  have any code yet.
- **Use [add-a-feature](add-a-feature.md)** for incremental work
  against a single implementation.
- **Use [port-to-another-language](port-to-another-language.md)** the
  first time you want a second-language implementation, or to
  reverse-engineer an existing program into a `.dx`.
- **Use [add-a-feature-to-multiple-implementations](add-a-feature-to-multiple-implementations.md)**
  once you have two or more language implementations and want them to
  evolve together.

## Possible future journeys

Not currently planned but plausible candidates if there's demand:

- **Spec deprecation: retiring an invariant.** The mirror image of
  feature addition — removing or weakening a constraint and
  confirming nothing in the implementations relied on it implicitly.
- **Cross-team handoff.** Transferring a `.dx`-managed project
  between teams or contributors, with the goal of preserving design
  intent across the change.
- **Onboarding via `.dx`.** Bringing a new contributor up to speed
  on an existing `.dx`-managed project; the spec as the entry-point
  artifact instead of the README.

If you'd like one of these (or another) sooner, file an issue or
open a PR with a draft. The
[ROADMAP](../../ROADMAP.md) tracks documentation gaps alongside
toolchain gaps.

## Contributing a journey

A good journey:

- **Has a real, named outcome.** "Add a feature to multiple
  implementations", not "use dx effectively".
- **Names the agent runtime explicitly** for at least one concrete
  example, even if the rest of the doc is agent-agnostic.
- **Lists known gaps honestly** at the end. Documenting the broken
  parts is more valuable than pretending they don't exist; it gives
  the reader a fair appraisal and the project a prioritized backlog.
- **Cites the skills that drive each phase**, so a reader can drill
  into the operational rules without re-reading the whole journey.
- **Includes a worked example** if one exists in the repo, with a
  pointer to it. If a worked example doesn't exist yet, mark the
  slot as `> **TODO:**` so it shows up in `grep`.

Use [`port-to-another-language.md`](port-to-another-language.md) as
a structural template — it has the most field-tested shape.
