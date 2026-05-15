# dx

A declarative specification language and toolchain for agentic AI.

`dx` provides a structured boundary between high-level design and
imperative implementation. In a world where AI writes the code, `dx`
ensures that humans (and the agents that work with them) maintain
control over the **intent** and **constraints** of a system without
being mired in the syntax of any particular implementation.

## A 30-second tour

A `.dx` file captures what a system should do — not how. E.g., here is
a `dx` declaration serialized as YAML:

```yaml
system: hello-world
intent:
  primary: Greet a user by name on standard output.
invariants:
  iface_stdout: Writes a single UTF-8 line to stdout terminated by `\n`.
assumptions: {}
contracts:
  greets_named_user:
    given: The argument vector contains exactly one non-empty name.
    when: The binary is invoked.
    then: stdout contains "Hello, <name>!\n" and the exit code is 0.
unconstrained:
  language: Any language with a stable POSIX runtime is acceptable.
```

### Validation

The `dx` CLI validates it:

```console
$ dx lint hello.dx
hello.dx: ok
```

### Semantic Diff

When the spec evolves, `dx diff` reports the *semantic* change — not
artifacts from serialization. E.g., reordering keys produces zero
output; whereas promoting an `assumption` to an `invariant` produces a
semantically meaningful diff.  Here's a real diff between two
revisions of a slightly larger spec:

```console
$ dx diff HEAD:system.dx system.dx
[MUTATED]  intent.primary
[PROMOTED] assumptions.cache.location -> invariants.iface_cache_path
[ADDED]    unconstrained.language
```

## Why dx exists

Modern coding agents excel at writing imperative code from prompts but
fail in two predictable ways:

- **They silently invent missing details.** A prompt says "fetch the
  user's email"; the agent quietly picks a timeout, a retry policy, a
  cache strategy, and a JSON shape, and bakes them into the code. The
  human never sees the choices, can't audit them, and can't ratify or
  reject them later.
- **They drift across iterations.** What was implicit in last week's
  prompt is gone from this week's context. The new code subtly
  contradicts the old. Six iterations in, nobody knows what the system
  is supposed to do.

`dx` addresses both. The `.dx` file is the version-controlled,
machine-validated record of what was decided and what was deliberately
left open. The `assumptions:` block is where every heuristic an agent
makes gets recorded *before* it touches the code — turning silent
hallucinations into auditable, promotable workflow state. The CLI
enforces the spec and reports *semantic* changes (not text changes)
when it evolves.

> There is no LLM inside the `dx` binary. The intelligence lives
> in the agents that consume it; the binary is the referee.

The philosophy behind this — why a separate declarative artifact, why
this particular moment, what the prior art looks like — is laid out in
[SPEC.md Part I §1 (Philosophy)](SPEC.md#1-philosophy). qThe full
language definition (concepts plus the v0.1.0 YAML serialization)
lives in the same document.

## Use it with your coding agent

The interesting workflow isn't a human typing `dx lint` in a
terminal — it's a coding agent (Claude Code, Gemini CLI, Cursor, your
in-house agent loop, anything that consumes Markdown skills) that uses
`dx` to keep itself honest while it writes code.

The repo ships seven portable agent skills under [`skills/`](skills/)
that teach any compatible agent various roles found to be valuable in
separating the various critical roles (e.g., reverse engineering
existing code, architecting new code, implementing code, evaluating
conformance to specification, etc).

The fastest way to see this work is to walk
[the port-to-another-language journey](docs/journeys/port-to-another-language.md):
hand a coding agent an existing program in one language, watch it
produce a `.dx` spec, then watch it synthesize an equivalent program in
a different language that passes every contract. 

## Install

```bash
git clone https://github.com/dewitt/dx && cd dx
go build -o ./bin/dx ./cmd/dx
```

The result is a single statically-linked Go binary. To use it from any
directory, drop it on your `$PATH`:

```bash
cp ./bin/dx ~/bin/      # or wherever your $PATH lives
dx --version
```

## User journeys

Step-by-step walkthroughs for using `dx` to accomplish concrete
tasks live under [`docs/journeys/`](docs/journeys/). Each journey is
agent-agnostic with at least one concrete example, names the skills
that drive each phase, and lists known gaps in v0.1.0.

| Journey | What it covers |
| ------- | -------------- |
| [Greenfield development](docs/journeys/greenfield-development.md) | Start from a vague idea, iterate `system.dx` with an agent until the spec converges, then one-shot the first implementation. |
| [Add a feature to an existing program](docs/journeys/add-a-feature.md) | Extend a `.dx`-managed system: spec changes first; the implementer reads the diff to know exactly what to update; the judge re-runs every contract to catch regressions. |
| [Add a feature to multiple implementations](docs/journeys/add-a-feature-to-multiple-implementations.md) | The library / SDK case: one spec governs N language implementations (e.g., Python + Go + TypeScript). One architect commit; N parallel implementer sessions blind to each other; one judge grid that catches cross-language drift. |
| [Port a program to another language](docs/journeys/port-to-another-language.md) | Reverse-engineer an existing implementation into a `.dx` spec, then synthesize an equivalent program in a new language without ever reading the original source. |

See the [`docs/journeys/` index](docs/journeys/README.md) for how the
journeys relate to each other, plus
[`ROADMAP.md`](ROADMAP.md) for the prioritized list of tooling and
spec gaps.

## What's in `.dx`

A `.dx` file is YAML 1.2 with a strict subset enforced by `dx lint`.
Six top-level blocks, in canonical order:

| Block            | Required | Holds                                                          |
| ---------------- | :------: | -------------------------------------------------------------- |
| `system`         |    yes   | A short slug naming the declaration.                           |
| `intent`         |    yes   | The high-level purpose: `primary` (one sentence) plus optional `secondary` goals. |
| `invariants`     |    yes   | Non-negotiable observable behaviors.                           |
| `assumptions`    |    yes   | Heuristic choices the agent made that the human hasn't ratified. May be empty (`{}`), but the key must exist. |
| `contracts`      |    no    | Black-box `given` / `when` / `then` rules a judge can check.   |
| `unconstrained`  |    no    | Explicitly-declared degrees of freedom (e.g., language choice, internal storage). Prevents over-specification. |

The 30-second tour above is one full example. A larger working example
with a real C++ legacy implementation and a Python re-synthesis lives
at [`examples/weather_cli/`](examples/weather_cli/).

For the formal grammar and SPEC §2 physical-rule list, see
[`SPEC.md`](SPEC.md). For the dense, agent-facing language reference,
see [`skills/dx-authoring/SKILL.md`](skills/dx-authoring/SKILL.md).

## CLI reference

| Command                  | Purpose                                                       |
| ------------------------ | ------------------------------------------------------------- |
| `dx lint`           | Validate `.dx` files against SPEC §2 and §3.                  |
| `dx fmt`            | Canonicalize `.dx` formatting (idempotent, AST-preserving).   |
| `dx diff`           | Emit a semantic ledger of operations between two `.dx` files. |
| `dx export`         | Emit the AST as canonical YAML (default) or compact JSON.     |
| `dx contracts list` | Enumerate the contract identifiers in a `.dx` file.           |

Every source-accepting command also takes git revision specs
(`HEAD:system.dx`, `main:foo.dx`, `v0.1.0:bar.dx`) anywhere a path is
expected, mirroring `git show` syntax.

`dx verify` — a black-box contract-execution harness — is
deliberately deferred to v0.2; the rationale and the v0.1.0 substitute
(the [`judge`](skills/judge/SKILL.md) skill) are documented in
[SPEC.md §4](SPEC.md#4-verification-model).

See [`skills/dx-toolchain/SKILL.md`](skills/dx-toolchain/SKILL.md)
for invocation details, exit codes, and the post-merge ritual.

## Project layout

```
.
├── SPEC.md                 # Concepts + v0.1.0 serialization + reference-implementation pointer.
├── AGENTS.md               # Behavioral protocol for every agent in this repo.
├── ROADMAP.md              # Prioritized index of known gaps and v0.2 work.
├── cmd/dx/            # CLI entry point.
├── pkg/                    # Library packages (ast, lint, canonical, diff, export, contracts).
├── skills/                 # Seven portable agent skills (orchestrator + 4 roles + 2 references).
├── docs/journeys/          # End-to-end walkthroughs for real tasks.
└── examples/               # hello.dx, weather_cli/, plus deliberate-failure fixtures for tests.
```

## Contributing

Three documents govern this project. Read them in this order if
you're making non-trivial changes:

1. [`AGENTS.md`](AGENTS.md) — the behavioral protocol every
   contributor (human or AI) follows in this repository.
2. [`SPEC.md`](SPEC.md) — the language definition. Part I is the
   philosophy and the conceptual framework; Part II is the v0.1.0
   YAML serialization; Part III points at the reference
   implementation that ships in this repo.
3. The [`skills/`](skills/) directory — operational playbooks per
   role.

Standard build / vet / test:

```bash
go build ./...
go vet ./...
go test ./...
```

Lint every `.dx` in the repo:

```bash
./bin/dx lint examples/hello.dx examples/weather_cli/system.dx
```

If you're not sure where to start, [`ROADMAP.md`](ROADMAP.md) lists the
v0.1.x and v0.2 gaps in priority order. The `dx verify` design and
the implementer no-peeking convention are the two largest open
questions and welcome real proposals.
