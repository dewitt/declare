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

## A 30-second tour

A `.dx` file captures what a system should do — not how. It's a strict
YAML subset (so syntax highlighting works in any editor without a
plugin):

```yaml
system: hello-world

intent:
  primary: |
    Greet a user by name on standard output.

invariants:
  iface_stdout: |
    Writes a single UTF-8 line to stdout terminated by `\n`.

contracts:
  greets_named_user:
    given: |
      The argument vector contains exactly one non-empty name.
    when: |
      The binary is invoked.
    then: |
      stdout contains "Hello, <name>!\n" and the exit code is 0.

assumptions: {}

unconstrained:
  language: |
    Any language with a stable POSIX runtime is acceptable.
```

The `declare` CLI validates it:

```console
$ declare lint hello.dx
hello.dx: ok
```

When the spec evolves, `declare diff` reports the *semantic* change —
not the YAML noise. Reordering keys produces zero output; promoting an
assumption to an invariant produces one line. Here's a real diff
between two revisions of a slightly larger spec:

```console
$ declare diff HEAD:system.dx system.dx
[MUTATED]  intent.primary
[PROMOTED] assumptions.cache.location -> invariants.iface_cache_path
[ADDED]    unconstrained.language
```

That `[PROMOTED]` line is the tell. It's the architect saying "we
considered this assumption and we're committing to it now" — visible at
a glance, in code review, on a chat handoff between agents. No
diff-of-YAML can show that.

## Why declare exists

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

`declare` addresses both. The `.dx` file is the version-controlled,
machine-validated record of what was decided and what was deliberately
left open. The `assumptions:` block is where every heuristic an agent
makes gets recorded *before* it touches the code — turning silent
hallucinations into auditable, promotable workflow state. The CLI
enforces the spec and reports *semantic* changes (not text changes)
when it evolves.

> There is no LLM inside the `declare` binary. The intelligence lives
> in the agents that consume it; the binary is the referee.

## Use it with your coding agent

The interesting workflow isn't a human typing `declare lint` in a
terminal — it's a coding agent (Claude Code, Gemini CLI, Cursor, your
in-house agent loop, anything that consumes Markdown skills) that uses
`declare` to keep itself honest while it writes code.

The repo ships seven portable agent skills under [`skills/`](skills/)
that teach any compatible agent four roles:

- **archaeologist** — distills an existing codebase into a base `.dx`.
- **architect** — owns the `.dx`: prunes invariants, promotes
  assumptions.
- **implementer** — generates code from the `.dx` (and only the `.dx`).
- **judge** — verifies the code against the contracts.

Plus an orchestrator skill that routes between them.

The fastest way to see this work is to walk
[the port-to-another-language journey](docs/journeys/port-to-another-language.md):
hand a coding agent an existing program in one language, watch it
produce a `.dx` spec, then watch it synthesize an equivalent program in
a different language that passes every contract. End-user-tested with
Gemini CLI; portable across runtimes.

## Install

```bash
git clone https://github.com/dewitt/declare && cd declare
go build -o ./bin/declare ./cmd/declare
```

The result is a single statically-linked Go binary. To use it from any
directory, drop it on your `$PATH`:

```bash
cp ./bin/declare ~/bin/      # or wherever your $PATH lives
declare --version
```

For a tagged release, `git checkout v0.1.0` first.

## User journeys

Step-by-step walkthroughs for using `declare` to accomplish concrete
tasks live under [`docs/journeys/`](docs/journeys/). Each journey is
agent-agnostic with at least one concrete example, names the skills
that drive each phase, and lists known gaps in v0.1.0.

| Journey | What it covers |
| ------- | -------------- |
| [Port a program to another language](docs/journeys/port-to-another-language.md) | Reverse-engineer an existing implementation into a `.dx` spec, then synthesize an equivalent program in a new language without ever reading the original source. |

More journeys (greenfield prototyping, spec evolution, multi-language
reference sets) are planned; see the
[`docs/journeys/` index](docs/journeys/README.md) for the documentation
backlog and [`ROADMAP.md`](ROADMAP.md) for the prioritized list of
tooling and spec gaps.

## What's in `.dx`

A `.dx` file is YAML 1.2 with a strict subset enforced by `declare
lint`. Six top-level blocks, in canonical order:

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
| `declare lint`           | Validate `.dx` files against SPEC §2 and §3.                  |
| `declare fmt`            | Canonicalize `.dx` formatting (idempotent, AST-preserving).   |
| `declare diff`           | Emit a semantic ledger of operations between two `.dx` files. |
| `declare export`         | Emit the AST as canonical YAML (default) or compact JSON.     |
| `declare contracts list` | Enumerate the contract identifiers in a `.dx` file.           |

Every source-accepting command also takes git revision specs
(`HEAD:system.dx`, `main:foo.dx`, `v0.1.0:bar.dx`) anywhere a path is
expected, mirroring `git show` syntax.

`declare verify` — a black-box contract-execution harness — is
deliberately deferred to v0.2; the rationale and the v0.1.0 substitute
(the [`judge`](skills/judge/SKILL.md) skill) are documented in
[SPEC.md §4](SPEC.md#4-verification-model).

See [`skills/declare-toolchain/SKILL.md`](skills/declare-toolchain/SKILL.md)
for invocation details, exit codes, and the post-merge ritual.

## Project layout

```
.
├── ARCHITECTURE.md         # Why declare exists; the multi-agent loop.
├── SPEC.md                 # Normative .dx language definition (v0.1.0).
├── AGENTS.md               # Behavioral protocol for every agent in this repo.
├── ROADMAP.md              # Prioritized index of known gaps and v0.2 work.
├── cmd/declare/            # CLI entry point.
├── pkg/                    # Library packages (ast, lint, canonical, diff, export, contracts).
├── skills/                 # Seven portable agent skills (orchestrator + 4 roles + 2 references).
├── docs/journeys/          # End-to-end walkthroughs for real tasks.
└── examples/               # hello.dx, weather_cli/, plus deliberate-failure fixtures for tests.
```

## Contributing

Four documents govern this project. Read them in this order if you're
making non-trivial changes:

1. [`AGENTS.md`](AGENTS.md) — the behavioral protocol every contributor
   (human or AI) follows in this repository.
2. [`SPEC.md`](SPEC.md) — the normative definition of the `.dx`
   language.
3. [`ARCHITECTURE.md`](ARCHITECTURE.md) — design rationale and the
   multi-agent loop.
4. The [`skills/`](skills/) directory — operational playbooks per role.

Standard build / vet / test:

```bash
go build ./...
go vet ./...
go test ./...
```

Lint every `.dx` in the repo:

```bash
./bin/declare lint examples/hello.dx examples/weather_cli/system.dx
```

If you're not sure where to start, [`ROADMAP.md`](ROADMAP.md) lists the
v0.1.x and v0.2 gaps in priority order. The `declare verify` design and
the implementer no-peeking convention are the two largest open
questions and welcome real proposals.
