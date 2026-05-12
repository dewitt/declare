# declare

A declarative specification language and toolchain for the agentic AI era.

`declare` provides a formal boundary between high-level design and
imperative implementation. In a world where AI writes the code,
`declare` ensures that humans (and the agents that work with them)
maintain control over the **intent** and **constraints** of a system
without being mired in the syntax of any particular implementation.

> **Status:** v0.1.0 development. The CLI lints, formats, diffs, and
> exports `.dx` files today; `verify` is deferred to v0.2 (see
> [SPEC.md](SPEC.md) §4 for the rationale).

## Why declare exists

Modern coding agents excel at writing imperative code from prompts but
fail in two predictable ways: they silently invent missing details, and
they drift across iterations. `declare` addresses both:

- **`.dx` files** — a strict YAML subset — capture *intent*,
  *invariants*, *contracts*, and *unconstrained degrees of freedom* in
  a form humans can read and machines can validate.
- **An `assumptions:` block** turns silent hallucinations into
  auditable, promotable workflow state. Every heuristic an agent makes
  is recorded *before* it touches the code.
- **A deterministic CLI** (`declare lint`, `declare diff`) enforces
  the spec and reports semantic changes — not text changes — when the
  spec evolves.

There is no LLM inside the `declare` binary. The intelligence lives in
the agents that consume it; the binary is the referee.

## Quick start

### Build

```bash
git clone https://github.com/dewitt/declare && cd declare
go build -o ./bin/declare ./cmd/declare
```

The binary is statically linked Go; drop it on `$PATH` to use it
anywhere.

### Lint a `.dx` file

```bash
./bin/declare lint examples/hello.dx
# examples/hello.dx: ok
```

### Diff two specs semantically

```bash
./bin/declare diff old.dx new.dx
# [MUTATED] intent.primary
# [PROMOTED] assumptions.cache.location -> invariants.iface_cache_path
# [ADDED] unconstrained.language
```

`declare diff` reports operations against the schema, not lines of
YAML — so reordering keys or reflowing a literal scalar produces zero
noise, and the canonical assumption-promotion workflow shows up in one
line.

### Bootstrap an agent

Any coding agent that consumes Markdown skills can adopt the
`declare` workflow today. Point it at the [`skills/`](skills/)
directory; the [`declare-orchestrator`](skills/declare-orchestrator/SKILL.md)
skill is the entry point and routes to the four role-skills
(archaeologist, architect, implementer, judge).

## User journeys

Step-by-step walkthroughs for using `declare` to accomplish concrete
tasks live under [`docs/journeys/`](docs/journeys/). Each journey is
agent-agnostic with at least one concrete example, names the skills
that drive each phase, and lists known gaps in v0.1.0.

| Journey | What it covers |
| ------- | -------------- |
| [Port a program to another language](docs/journeys/port-to-another-language.md) | Reverse-engineer an existing implementation into a `.dx` spec, then synthesize an equivalent program in a new language without ever reading the original source. |

More journeys (greenfield prototyping, spec evolution,
multi-language reference sets) are planned; see the
[`docs/journeys/` index](docs/journeys/README.md) for the documentation
backlog and [`ROADMAP.md`](ROADMAP.md) for the prioritized list of
tooling and spec gaps.

## What's in `.dx`

A `.dx` file is YAML 1.2 with a strict subset enforced by `declare
lint`. The required blocks are `system`, `intent`, `invariants`, and
`assumptions`; `contracts` and `unconstrained` are optional but
strongly encouraged.

A minimal example:

```yaml
system: hello-world

intent:
  primary: |
    Greet a user by name on standard output.

invariants:
  iface_stdout: |
    Writes a single UTF-8 line to stdout terminated by `\n`.

assumptions:
  greeting.format: |
    The greeting is "Hello, <name>!". The spec does not pin the
    punctuation; this matches the canonical POSIX-tutorial form.

contracts:
  greets_named_user:
    given: |
      The argument vector contains exactly one non-empty name.
    when: |
      The binary is invoked.
    then: |
      stdout contains "Hello, <name>!\n" and the exit code is 0.

unconstrained:
  language: |
    Any language with a stable POSIX runtime is acceptable.
```

A larger worked example with a real C++ legacy implementation and a
Python re-synthesis lives at
[`examples/weather_cli/`](examples/weather_cli/).

For the formal grammar and SPEC §2 physical-rule list, see
[`SPEC.md`](SPEC.md). For the language reference your agent should
consult before authoring, see
[`skills/dx-authoring/SKILL.md`](skills/dx-authoring/SKILL.md).

## Project layout

```
.
├── ARCHITECTURE.md         # Why declare exists; the multi-agent loop.
├── SPEC.md                 # Normative .dx language definition (v0.1.0).
├── AGENTS.md               # Behavioral protocol for every agent in this repo.
├── README.md               # You are here.
├── ROADMAP.md              # Prioritized index of known gaps and v0.2 work.
├── cmd/
│   └── declare/            # CLI entry point (cobra).
├── pkg/
│   ├── ast/                # In-memory representation of a .dx declaration.
│   ├── lint/               # SPEC §2 + §3 enforcement.
│   ├── canonical/          # Deterministic AST -> YAML serializer (used by fmt + export).
│   ├── diff/               # Semantic ledger between two declarations.
│   ├── export/             # Canonical YAML / compact JSON projections.
│   └── contracts/          # Operations over the contracts: block.
├── skills/
│   ├── declare-orchestrator/   # Meta router; load this first.
│   ├── dx-authoring/           # Spec reference for writing .dx files.
│   ├── declare-toolchain/      # How to use the CLI from an agent loop.
│   ├── archaeologist/          # Role: distill code into a .dx.
│   ├── architect/              # Role: own intent / invariants / contracts.
│   ├── implementer/            # Role: generate code from a .dx.
│   └── judge/                  # Role: verify code against contracts.
├── docs/
│   └── journeys/               # End-to-end walkthroughs for real tasks.
└── examples/
    ├── hello.dx                # Minimal valid .dx file.
    ├── broken.dx               # Deliberately malformed (lint smoke test).
    ├── missing.dx              # Missing required keys (lint smoke test).
    └── weather_cli/            # Canonical worked example.
```

## CLI commands

| Command                  | Status             | Purpose                                                       |
| ------------------------ | ------------------ | ------------------------------------------------------------- |
| `declare lint`           | implemented        | Validate `.dx` files against SPEC §2 and §3.                  |
| `declare fmt`            | implemented        | Canonicalize `.dx` formatting (idempotent, AST-preserving).   |
| `declare diff`           | implemented        | Emit a semantic ledger of operations between two `.dx` files. |
| `declare export`         | implemented        | Emit the AST as canonical YAML (default) or compact JSON.     |
| `declare contracts list` | implemented        | Enumerate the contract identifiers in a `.dx` file.           |
| `declare verify`         | deferred to v0.2   | Run the `contracts:` block as a black-box test harness.       |

See [`skills/declare-toolchain/SKILL.md`](skills/declare-toolchain/SKILL.md)
for invocation details, exit codes, and the post-merge ritual.

## Contributing

The project is governed by the documents in this repo, in this order
of precedence:

1. [`AGENTS.md`](AGENTS.md) — behavioral protocol for any agent
   (human or AI) modifying this repository.
2. [`SPEC.md`](SPEC.md) — normative definition of the `.dx` language.
3. [`ARCHITECTURE.md`](ARCHITECTURE.md) — design rationale and the
   multi-agent loop.
4. The [`skills/`](skills/) directory — operational playbooks per role.

Tests, build, vet:

```bash
go build ./...
go vet ./...
go test ./...
```

Lint every `.dx` in the repo:

```bash
./bin/declare lint examples/hello.dx examples/weather_cli/system.dx
```
