# dx

A declarative specification language for AI-assisted software
development.

A dx declaration describes what a software system should do — its
intent, its invariants, the contracts that prove conformance, the
choices left open to the implementer. The language and its
toolchain exist to give humans and coding agents a shared,
auditable record of what was decided, separate from any particular
implementation in any particular programming language.

Declarations are CommonMark files (canonical extension `.md`).
The same declaration can govern any number of concrete
implementations. When the spec changes, the diff is a list of
operations against the schema (an invariant added, an assumption
promoted), not a wall of red and green prose. When an agent has
to guess, the guess is recorded as an explicit assumption before
the code is written, so a human can review it later.

## A 30-second tour

````markdown
# hello-world

## Intent

Greet a user by name on standard output.

## Invariants

### Single line on stdout

Writes a single UTF-8 line to stdout terminated by `\n`.

## Assumptions

## Contracts

### Greets a named user

**Given:** The argument vector contains exactly one non-empty name.

**When:** The binary is invoked.

**Then:** stdout contains `Hello, <name>!\n` and the exit code is 0.

## Unconstrained

### Implementation language

Any language with a stable POSIX runtime is acceptable.
````

Validate it:

```console
$ dx lint hello.md
hello.md: ok
```

When the spec evolves, `dx diff` reports the semantic change.
Reordering blocks produces no output, and polishing a heading
without changing its meaning produces no output. Promoting an
assumption to an invariant produces one line:

```console
$ dx diff HEAD:system.md system.md
[MUTATED]  intent[0]
[PROMOTED] assumptions.cache_location -> invariants.cache_path
[ADDED]    unconstrained.implementation_language
```

That `[PROMOTED]` line is the discipline. It records that the
architect considered an open question and committed to a constraint,
in a form that survives code review and chat-handoff.

## Use it with a coding agent

The `dx` binary is one half of the story. The other half is the
seven agent skills shipped under [`skills/`](skills/), which teach
a coding agent (Claude Code, Gemini CLI, Cursor, or any agent that
reads Markdown skills) the four roles the workflow defines:
archaeologist, architect, implementer, and judge. Each role has
its own write privileges on the declaration; the orchestrator skill
routes work between them.

The fastest way to see the workflow in motion is the
[port-to-another-language journey](docs/journeys/port-to-another-language.md):
hand a coding agent an existing program in one language, watch it
extract a declaration, then watch it synthesize an equivalent
program in a different language that passes every contract. The
journey doc has been clean-room-tested end-to-end against
gemini-cli on a fresh system; the other three documented journeys
(greenfield, add-a-feature, multi-implementation) follow the same
shape.

## Install

```bash
git clone https://github.com/dewitt/dx && cd dx
go build -o ./bin/dx ./cmd/dx
```

The result is a single statically-linked Go binary. Drop it on
`$PATH` to use from anywhere:

```bash
cp ./bin/dx ~/bin/
dx --version
```

## CLI reference

| Command                  | Purpose                                                       |
| ------------------------ | ------------------------------------------------------------- |
| `dx lint <source>`       | Validate a declaration against the spec.                      |
| `dx fmt <source>`        | Canonicalize formatting (idempotent, AST-preserving).         |
| `dx diff <old> <new>`    | Emit a semantic ledger of operations between two declarations. |
| `dx export <source>`     | Emit the AST as canonical CommonMark or compact JSON.         |
| `dx contracts list <source>` | Enumerate the contract identifiers in a declaration.       |

Every source argument accepts a filesystem path or a git revision
spec (`HEAD:foo.md`, `main:bar.md`, `v0.2.0:baz.md`), mirroring
`git show` syntax.

`dx verify`, a black-box contract executor, is deferred to a
later release. The judge skill is its current substitute.

For invocation details, exit codes, and the post-merge ritual, see
[`skills/dx-toolchain/SKILL.md`](skills/dx-toolchain/SKILL.md).

## User journeys

End-to-end walkthroughs for using `dx` to accomplish concrete
tasks live under [`docs/journeys/`](docs/journeys/):

| Journey | When to use it |
| ------- | -------------- |
| [Greenfield development](docs/journeys/greenfield-development.md) | Starting from a vague idea; iterate the spec to convergence before any code is written. |
| [Add a feature](docs/journeys/add-a-feature.md) | Extend an existing dx-managed system; spec changes first, then the implementer reads the diff and updates the code. |
| [Add a feature to multiple implementations](docs/journeys/add-a-feature-to-multiple-implementations.md) | One declaration governs N language implementations (the SDK case); one architect commit, N parallel implementer sessions, one verification grid. |
| [Port a program to another language](docs/journeys/port-to-another-language.md) | Reverse-engineer an existing implementation, then synthesize an equivalent in a different language without reading the original source. |

See the [journeys index](docs/journeys/README.md) for how the
journeys relate to each other.

## Project layout

```
.
├── SPECIFICATION.md        The dx language definition (RFC-style).
├── WORKFLOW.md             The recommended multi-agent operating workflow.
├── AGENTS.md               Behavioral protocol for agents in this repo.
├── README.md               This file.
├── cmd/dx/                 CLI entry point.
├── pkg/                    Library packages: ast, lint, canonical, diff, export, contracts.
├── skills/                 Seven portable agent skills.
├── docs/journeys/          End-to-end walkthroughs.
└── examples/               hello.md, weather_cli/, plus deliberate-failure fixtures for tests.
```

## Contributing

The project is governed by three documents and one directory of
skills, in reading order:

1. [`SPECIFICATION.md`](SPECIFICATION.md) — what the dx language is.
2. [`WORKFLOW.md`](WORKFLOW.md) — how the multi-agent workflow operates on a declaration.
3. [`AGENTS.md`](AGENTS.md) — behavioral protocol for any agent (human or AI) modifying this repository.
4. The [`skills/`](skills/) directory — operational playbooks per role.

Build, vet, and test:

```bash
go build ./...
go vet ./...
go test ./...
```

Lint the bundled examples:

```bash
./bin/dx lint examples/hello.md examples/weather_cli/system.md
```

Open design questions worth contributing to are listed in
[`SPECIFICATION.md` §3.10](SPECIFICATION.md#310-future-directions).
The serialization itself is at v0.2.0 (CommonMark); see
[`SPECIFICATION.md` §4](SPECIFICATION.md#4-serialization-v020).
