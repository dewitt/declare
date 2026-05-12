# `declare` user journeys

End-to-end walkthroughs for using `declare` to accomplish real
tasks. Each journey is written for a developer who has the `declare`
CLI installed and a coding agent of their choice; instructions are
agent-agnostic with one concrete example (typically Claude Code).

| Journey | What it covers | Status |
| ------- | -------------- | ------ |
| [Port a program to another language](port-to-another-language.md) | Reverse-engineer an existing implementation into a `.dx` spec, then synthesize an equivalent implementation in a new language without ever reading the original source. | Documented; see "Known gaps" in the journey doc for v0.1.0 limitations. |

## Journeys we plan to add

These are not yet documented. Each will land as its own file under
this directory.

- **Greenfield: prototype-first.** Start from a vague human idea,
  collaboratively author `system.dx` with an agent, then synthesize
  the first implementation. Inverts the port journey — there is no
  archaeologist phase, and the architect drives extraction directly
  from the human's prose.
- **Spec evolution: tighten or relax.** You have a working
  `system.dx` and an implementation. You want to add a new
  invariant, promote an old assumption, or relax a constraint that
  turned out to be over-specified. Walks the architect → implementer
  → judge loop on a small, focused change.
- **Multi-language reference set.** Maintain `impl_<lang>/`
  directories for two or more languages from the same `system.dx`,
  using the spec as the cross-language conformance test. Useful for
  reference implementations, SDKs, and language-shootout
  demonstrations.

If you want to lobby for one of these to land sooner, file an issue
or open a PR with a draft.

## Contributing a journey

A good journey:

- **Has a real, named outcome.** "Port to another language", not
  "use declare effectively".
- **Names the agent runtime explicitly** for at least one concrete
  example, even if the rest of the doc is agent-agnostic.
- **Lists known gaps honestly** at the end. Documenting the broken
  parts is more valuable than pretending they don't exist; it gives
  the reader a fair appraisal and the project a prioritized backlog.
- **Cites the skills that drive each phase**, so a reader can drill
  into the operational rules without re-reading the whole journey.
- **Includes a worked example** if one exists in the repo, with a
  pointer to it.

Use [`port-to-another-language.md`](port-to-another-language.md) as
a structural template.
