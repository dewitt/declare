# `dx` Skills

This directory contains the agent skills that operationalize the
`dx` paradigm. Each subdirectory follows the Anthropic Agent
Skills convention: a `SKILL.md` file with YAML frontmatter
(`name`, `description`) followed by prose instructions. The
format is intentionally portable across coding agents — Claude
Code, Cursor, Gemini CLI, and any other agent runtime that
consumes Markdown skills — so any agent that consumes Markdown
playbooks can use them directly.

## How to use

Point your agent at this directory (or copy the subdirectories
into the agent's skill search path). The orchestrator skill is
the entry point and explains how to route any incoming task to
the correct role.

If your agent does not auto-load skills, instruct it to read
`skills/dx-orchestrator/SKILL.md` first.

## Skill Index

| Skill              | Role                | When to load                                                                              |
| ------------------ | ------------------- | ----------------------------------------------------------------------------------------- |
| `dx-orchestrator`  | Meta / router       | Always, on entering any dx-managed repo. Routes to the role-skills below.                 |
| `dx-authoring`     | Spec reference      | Whenever you are about to write or modify a declaration.                                  |
| `dx-toolchain`     | CLI usage           | Whenever you are about to invoke `dx lint / fmt / diff / export`. Also covers the post-merge ritual (SPEC §3.9). |
| `archaeologist`    | Role: extraction    | "Reverse-engineer this codebase into a declaration."                                      |
| `architect`        | Role: refinement    | "Write/refine the declaration." Owns `## Intent`, `## Invariants`, `## Contracts`, `## Unconstrained`. |
| `implementer`      | Role: coding        | "Generate the implementation from `system.md`." May only modify `## Assumptions`.         |
| `judge`            | Role: verification  | "Verify the implementation against the contracts." (Until `dx verify` ships, the judge skill *is* the contract executor.) |

## Design notes

- The skills assume the universal rules in the repo-root
  `AGENTS.md` are in effect. They cite specific sections of
  `AGENTS.md` and `SPECIFICATION.md` rather than restating those
  documents.
- Declarations are CommonMark files (canonical extension `.md`)
  per `SPECIFICATION.md` §4 (v0.2.0). The skills are written
  against that serialization throughout.
- The four role-skills (`archaeologist`, `architect`,
  `implementer`, `judge`) deliberately have **non-overlapping
  write privileges** on the declaration:

  | Block               | Archaeologist | Architect | Implementer | Judge |
  | ------------------- | :-----------: | :-------: | :---------: | :---: |
  | `# <system>`        |       W       |     W     |      —      |   —   |
  | `## Intent`         |       W       |     W     |      —      |   —   |
  | `## Invariants`     |       W       |     W     |      —      |   —   |
  | `## Assumptions`    |       W       |     W     |      W      |   —   |
  | `## Contracts`      |       W       |     W     |      —      |   —   |
  | `## Unconstrained`  |       W       |     W     |      —      |   —   |

  (The archaeologist writes the file from scratch, so it can
  populate every block; once it exists, only the architect may
  modify the spec-defining blocks. The implementer's write
  privilege is limited to appending `## Assumptions`.)

- The handoff format `HANDOFF: <from> → <to>: <reason>` is
  shared across all skills. It is the audit trail.

## Contributing

When updating these skills, remember:

- The skills must remain portable. Avoid commands or features
  specific to one agent runtime.
- Keep frontmatter minimal: `name` and `description` only.
- The `description` field is what an auto-discovery system uses
  to decide whether to load the skill. Lead with concrete
  trigger phrases ("Use when the user asks to …").
- If you change a role's write privileges, update the table
  above and the matching language in the affected role's
  SKILL.md.
