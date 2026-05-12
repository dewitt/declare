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
| `declare diff`   | implemented    | Emit a semantic ledger between two `.dx` files.         |
| `declare export` | stub           | Emit the AST in an agent-optimized format (e.g. JSON).  |
| `declare verify` | deferred to v0.2 | Run the `contracts:` block as a black-box test harness. |

The current binary lives at `./cmd/declare`. Build with `go build ./...`.
For one-off invocations during development, prefer:

```bash
go run ./cmd/declare <subcommand> [args...]
```

## 1a. Source resolution: file paths and git revisions {#git-revision-sources}

Every command that takes a `<source>` argument (`lint`, `diff`,
`export`) accepts the same two forms:

| Form              | Example                       | Resolution                                |
| ----------------- | ----------------------------- | ----------------------------------------- |
| Filesystem path   | `examples/hello.dx`           | Read directly from disk.                  |
| Git revision spec | `HEAD:examples/hello.dx`      | `git show <rev>:<path>`. Requires being inside a git working tree. |

The git-revision form mirrors `git show` syntax exactly. Anything
git accepts as `<rev>` works: a branch (`main:foo.dx`), a tag
(`v0.1.0:SPEC.md`), a relative ref (`HEAD~3:system.dx`), an explicit
SHA (`abc123:system.dx`).

### Disambiguation rules

The CLI distinguishes the two forms purely by syntax. An input is
treated as a filesystem path if any of the following hold:

- It contains no colon.
- It begins with `./`, `../`, `/`, or `-`.
- The pre-colon segment is a single character (so `C:\foo` is not
  mistaken for a git ref — real git revs are essentially never one
  character).

Everything else is parsed as `<rev>:<path>`.

### When to use the git-revision form

The canonical use is the architect's review loop after editing a
`.dx` file in place:

```bash
# What did I change semantically since the last commit?
declare diff HEAD:system.dx system.dx

# How does the spec on main differ from this branch's spec?
declare diff main:system.dx HEAD:system.dx

# Did some prior version even lint cleanly?
declare lint v0.1.0:examples/hello.dx
```

This obviates the previous `git show > /tmp/old.dx && declare diff
/tmp/old.dx system.dx` shell dance.

### Failure modes

Resolution failures (bad rev, missing path-in-rev) surface git's own
diagnostic verbatim, prefixed with `git show <input>:`. Examples:

```
$ declare diff badbad12345:system.dx system.dx
Error: git show badbad12345:system.dx: fatal: invalid object name 'badbad12345'.

$ declare diff HEAD:nope.dx system.dx
Error: git show HEAD:nope.dx: fatal: path 'nope.dx' does not exist in 'HEAD'
```

Empty rev (`:foo.dx`) or empty path (`HEAD:`) is rejected before any
`git` call, with a `declare`-side `invalid revision spec` diagnostic.

## 2. `declare lint`

### Invocation

```bash
declare lint <source> [<source> ...]
```

Accepts one or more sources. Each source may be either a filesystem
path (`examples/hello.dx`) or a git revision spec
(`HEAD:examples/hello.dx`, `HEAD~1:system.dx`, `main:foo.dx`),
mirroring `git show` syntax. See [§4
"git-revision sources"](#git-revision-sources) for the full grammar.

Reports each source's status to stdout (`<source>: ok`) and per-issue
diagnostics to stderr in the format `<source>:<line>:<col>: <message>`
(line/col omitted when unknown).

### Exit codes

| Code | Meaning                                                       |
| ---- | ------------------------------------------------------------- |
| 0    | All inputs passed lint.                                       |
| 1    | At least one source had a structural issue, or I/O / git resolution failed. |

### What it checks

- **SPEC §2 physical rules** (walked over the raw `*yaml.Node` graph):
  - No anchors (`&name`) or aliases (`*name`).
  - No custom or non-default YAML tags (`!!binary`, `!!set`, user
    `!foo`, etc.).
  - No folded block scalars (`>`); literal `|` is the only allowed
    multi-line form.
  - Map values under `invariants:`, `assumptions:`, and
    `unconstrained:` must be scalar strings, not nested mappings or
    sequences.
- **Strict structural decode** into the AST (`KnownFields(true)`):
  unknown top-level fields fail.
- **Required-key presence** (SPEC §3): `system`, `intent.primary`,
  `invariants`, `assumptions`. The `invariants` and `assumptions`
  checks consult the raw YAML node graph, so explicitly-empty maps
  (`{}`) are accepted while absent keys are flagged.

### What it does **not yet** check

- Slug-format validation on `system:` (SPEC §3 says "Type: String
  (Slug format)" but doesn't define the regex). Treated as advisory
  for v0.1.0; the architect's pruning pass should catch obvious
  violations.
- Category-prefix discipline on invariant IDs (also advisory; the
  prefix convention is enforced socially via skill review, not
  mechanically).

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

### Invocation

```bash
declare diff <old> <new>
```

Both `<old>` and `<new>` may be filesystem paths or git revision
specs (see [§1a "Source resolution"](#git-revision-sources)).

Emits a **semantic ledger** of operations to stdout, one per line, in
SPEC §2 canonical block order:

```
[MUTATED] intent.primary
[PROMOTED] assumptions.cache.location -> invariants.iface_cache_path
[ADDED] unconstrained.language
```

### Operation taxonomy

| Op           | Meaning                                                                              |
| ------------ | ------------------------------------------------------------------------------------ |
| `[ADDED]`    | A path exists in `<new>` but not in `<old>`.                                         |
| `[REMOVED]`  | A path exists in `<old>` but not in `<new>`.                                         |
| `[MUTATED]`  | Same path on both sides; value differs.                                              |
| `[PROMOTED]` | Same body, moved toward `invariants` (more committed). E.g., `assumptions.x → invariants.x`. |
| `[DEMOTED]`  | Same body, moved away from `invariants` (less committed). E.g., `invariants.x → unconstrained.x`. |
| `[RENAMED]`  | Same body, same block, different key.                                                |

### Exit codes

| Code | Meaning                                                          |
| ---- | ---------------------------------------------------------------- |
| 0    | Diff completed (whether or not changes were found).              |
| 1    | One of the inputs failed to decode; the file path is reported.   |

The diff command does **not** require either input to lint cleanly; an
architect may legitimately diff a known-broken spec against its fix.
It does require both files to decode into a `Declaration`.

### When to use it (vs. text diff)

Always, when communicating spec changes to a human or another agent.
This is the canonical mechanism for AGENTS.md §5 ("Communication with
Humans"): a text diff over YAML is hostile to architectural review;
the semantic ledger is built for it.

## 5. `declare export`

Currently a stub: prints `Error: export: not yet implemented` and exits 1.

Eventual purpose: emit a token-optimized projection of the AST (default
format: compact JSON) for ingestion into another agent's context window.
Comments stripped, keys ordered, whitespace minimized.

When you need to hand a `.dx` to a downstream agent today, paste the raw
file. Do not synthesize a JSON form by hand — the canonical projection
needs to come from a deterministic source so two agents can agree on
hashes.

## 5a. `declare verify` (deferred to v0.2)

There is no `declare verify` command in v0.1.0. SPEC §4 explains why:
contract execution is intentionally human/agent-driven for the first
release, performed by an agent operating under the `judge` skill.

If you find yourself wanting to write `declare verify`, instead:

1. Load the `judge` skill.
2. Walk every entry in `contracts:` by hand (or via your agent
   runtime's tool-use), setting up `given`, triggering `when`,
   evaluating `then`.
3. Classify any failure per the judge's failure-classification rules.

A future `declare verify` will automate steps 1–3 against a strict
contract grammar; until that ships, the judge skill is the contract.

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

## 6a. Post-Merge Ritual

When a `.dx` file is touched on multiple branches and merged, the
architect MUST run, in order:

1. `declare lint <merged>.dx` — a textual three-way merge can produce
   structurally invalid YAML (duplicate keys, broken indentation).
2. `declare diff <merge-base>.dx <merged>.dx` — surfaces every
   semantic operation introduced by the merge in one glance. A clean
   text-merge can still hide a semantic conflict (e.g., one branch
   demoted an invariant to `unconstrained:` while the other tightened
   it).
3. Reconcile any conflict in the **spec**, not the implementation.
   Per AGENTS.md §1 the `.dx` file leads.

This is the v0.1.0 stance per SPEC §5. A future revision may introduce
`declare merge` for AST-level structural merge; until then, the
architect runs the ritual manually after every merge that touches a
`.dx` file.

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

| Symptom                                                                         | Likely cause                                                  | Fix                                                  |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------- | ---------------------------------------------------- |
| `field <x> not found in type ast.Declaration`                                   | Top-level typo or unknown key.                                | Remove or rename to a SPEC §3 key.                   |
| `missing required key …`                                                        | Structural omission.                                          | Add the key (use `{}` for empty maps).               |
| `folded block scalar `>` forbidden by SPEC §2`                                  | Used `>` instead of `\|` for a multiline string.              | Replace `>` with `\|`.                               |
| `anchor &x forbidden by SPEC §2` / `alias node forbidden by SPEC §2`            | Used `&` / `*` to share content between blocks.               | Inline the content; SPEC §2 forbids hidden state.    |
| `explicit YAML tag "X" forbidden by SPEC §2`                                    | Used a custom tag like `!!binary` or `!foo`.                  | Remove the tag; encode the data as a normal string.  |
| `invariants.X must be a scalar string`                                          | Tried to give an invariant a structured body (e.g., `rule:`/`reason:`). | Flatten to a single literal scalar (v0.1.0); see SPEC §6 for the v0.2 audit-trail proposal. |
| Lint passes but a contract fails immediately on a clean impl.                   | Contract `then` references internal state, not output.        | Rewrite the contract (architect's job).              |
| `declare export` exits 1 with `not yet implemented`.                            | Stub.                                                         | Use the raw `.dx` file until the command is shipped. |

## 9. Anti-Patterns

- **Running the implementation without first linting the spec.** The spec
  may have drifted into an undecodable state during a previous edit.
- **Treating `declare fmt`'s no-op as "already canonical."** It's a stub.
- **Hand-rolling a JSON projection of a `.dx` file** for downstream
  agents. Wait for `declare export`, or paste the raw file.
- **Shelling out to `yq`/`jq` to mutate `.dx` files.** Mutate via the
  `architect` skill and re-lint; ad-hoc YAML editing tools don't enforce
  SPEC §2 physical rules.
