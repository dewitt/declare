---
name: declare-toolchain
description: |
  How to invoke the deterministic `declare` CLI (`lint`, `fmt`, `diff`,
  `export`) from inside any agent's event loop. Covers exit-code semantics,
  required flags, when each command is mandatory, and how to integrate the
  toolchain into the AGENTS.md verification loop. Load this whenever you are
  about to run `declare` as a subprocess or wire it into CI.
---

# The `declare` Toolchain

The `declare` binary contains **no LLM**. Every command is a deterministic
operation over the `.dx` AST. This skill tells you when to invoke each
command and how to interpret its output.

## 1. Command Inventory

| Command          | Status         | Purpose                                                 |
| ---------------- | -------------- | ------------------------------------------------------- |
| `declare lint`   | implemented    | Validate `.dx` files against SPEC structural rules.     |
| `declare fmt`    | stub           | Canonicalize formatting (whitespace, key order).        |
| `declare diff`   | not yet built  | Emit a semantic ledger between two `.dx` files.         |
| `declare export` | stub           | Emit the AST in an agent-optimized format (e.g. JSON).  |

The current binary lives at `./cmd/declare`. Build with `go build ./...`.
For one-off invocations during development, prefer:

```bash
go run ./cmd/declare <subcommand> [args...]
```

## 2. `declare lint`

### Invocation

```bash
declare lint path/to/file.dx [more.dx ...]
```

Accepts one or more `.dx` files. Reports each file's status to stdout
(`<path>: ok`) and per-issue diagnostics to stderr in the format
`<path>:<line>:<col>: <message>` (line/col omitted when unknown).

### Exit codes

| Code | Meaning                                                        |
| ---- | -------------------------------------------------------------- |
| 0    | All input files passed lint.                                   |
| 1    | At least one file had a structural issue, or I/O failed.       |

### What it currently checks

- Strict YAML decode into the AST (`KnownFields(true)`): unknown top-level
  fields fail.
- Required-key presence: `system`, `intent.primary`, `invariants`,
  `assumptions`. The `invariants` and `assumptions` checks consult the raw
  YAML node graph, so explicitly-empty maps (`{}`) are accepted while
  absent keys are flagged.

### What it does **not yet** check (planned)

- SPEC §2 physical rules: anchors/aliases, custom tags, folded scalars.
  The AST retains `*yaml.Node` so these rules can be added without an
  API break.

If your task depends on one of those checks (e.g., you need to refuse
folded scalars), do the inspection by hand on the YAML source and file
the gap.

### When `declare lint` is mandatory

Per AGENTS.md §3 ("Verification Loop"):

- **Before** writing or generating code from a `.dx` file.
- **After** any modification to a `.dx` file, before declaring the task
  complete.
- **In CI**, against every `.dx` file in the repo.

A non-zero `declare lint` exit means the spec is structurally untrustworthy.
Fix it (acting as `architect`) before running any other tool.

## 3. `declare fmt`

Currently a stub: prints `fmt: not yet implemented` and exits 0. Do not
treat the absence of changes as confirmation of canonical form.

When implemented, the contract will be:

- Idempotent: `fmt(fmt(x)) == fmt(x)`.
- Order-normalizing: top-level keys reordered to SPEC §2 canonical order.
- Whitespace-normalizing: literal-scalar bodies preserved byte-for-byte;
  surrounding whitespace canonicalized.

Until then, hand-edit to canonical order and rely on `declare lint` for
structural checks.

## 4. `declare diff`

Not yet built. The intended contract (ARCHITECTURE.md §4):

```
declare diff old.dx new.dx
```

Emits a **semantic ledger** of operations:

```
[ADDED]    invariants.perf_p99_ms
[REMOVED]  unconstrained.storage_backend
[PROMOTED] assumptions.cli_default → invariants.iface_cli_default
[MUTATED]  intent.primary
```

Until it ships, when reporting `.dx` changes to a human, summarize in this
shape **manually** rather than pasting a text diff. That is the spirit of
AGENTS.md §5 ("Communication with Humans").

## 5. `declare export`

Currently a stub: prints `Error: export: not yet implemented` and exits 1.

Eventual purpose: emit a token-optimized projection of the AST (default
format: compact JSON) for ingestion into another agent's context window.
Comments stripped, keys ordered, whitespace minimized.

When you need to hand a `.dx` to a downstream agent today, paste the raw
file. Do not synthesize a JSON form by hand — the canonical projection
needs to come from a deterministic source so two agents can agree on
hashes.

## 6. The Verification Loop (canonical sequence)

This is the loop every role-skill invokes when work touches both the
spec and the implementation.

```
1. declare lint <changed>.dx                    # exit 0 required
2. <generate or modify implementation>
3. <build / compile the implementation>          # exit 0 required
4. <execute every contract in contracts:>        # all must pass
5. If any contract fails:
     - HANDOFF to judge for triage.
     - Judge classifies: implementation bug OR spec gap.
     - Implementation bug → fix code, return to step 3.
     - Spec gap         → HANDOFF to architect, return to step 1.
6. Done.
```

Skipping step 1 or step 4 is the failure mode `declare` exists to
prevent. Do not skip them under time pressure.

## 7. CI Snippet (reference)

A minimal GitHub-Actions-style block, illustrative only:

```yaml
- name: Build declare
  run: go build -o ./bin/declare ./cmd/declare

- name: Lint all .dx files
  run: |
    set -euo pipefail
    find . -name '*.dx' -print0 | xargs -0 ./bin/declare lint
```

The `set -euo pipefail` is important: a missing pipefail will let a
broken `find` mask a real lint failure.

## 8. Common Failure Modes

| Symptom                                                       | Likely cause                                                  | Fix                                                  |
| ------------------------------------------------------------- | ------------------------------------------------------------- | ---------------------------------------------------- |
| `field <x> not found in type ast.Declaration`                 | Top-level typo or unknown key.                                | Remove or rename to a SPEC §3 key.                   |
| `missing required key …`                                      | Structural omission.                                          | Add the key (use `{}` for empty maps).               |
| Lint passes but a contract fails immediately on a clean impl. | Contract `then` references internal state, not output.        | Rewrite the contract (architect's job).              |
| `declare export` exits 1 with `not yet implemented`.          | Stub.                                                         | Use the raw `.dx` file until the command is shipped. |
| Lint silently passes a file with `>` folded scalars.          | Physical-checks not yet wired up.                             | Manually grep for `: >` and fail the task.           |

## 9. Anti-Patterns

- **Running the implementation without first linting the spec.** The spec
  may have drifted into an undecodable state during a previous edit.
- **Treating `declare fmt`'s no-op as "already canonical."** It's a stub.
- **Hand-rolling a JSON projection of a `.dx` file** for downstream
  agents. Wait for `declare export`, or paste the raw file.
- **Shelling out to `yq`/`jq` to mutate `.dx` files.** Mutate via the
  `architect` skill and re-lint; ad-hoc YAML editing tools don't enforce
  SPEC §2 physical rules.
