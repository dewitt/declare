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

| Command                  | Status           | Purpose                                                       |
| ------------------------ | ---------------- | ------------------------------------------------------------- |
| `declare lint`           | implemented      | Validate `.dx` files against SPEC structural rules.           |
| `declare fmt`            | implemented      | Canonicalize formatting (key order, alphabetized maps, scalars). |
| `declare diff`           | implemented      | Emit a semantic ledger between two `.dx` files.               |
| `declare export`         | implemented      | Emit the AST as canonical YAML (default) or compact JSON.     |
| `declare contracts list` | implemented      | Enumerate the contract identifiers in a `.dx` file.           |
| `declare verify`         | deferred to v0.2 | Run the `contracts:` block as a black-box test harness.       |

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

### Invocation

```bash
declare fmt <file> [<file> ...]            # writes canonical output to stdout
declare fmt --write <file> [<file> ...]    # overwrites each input in place
declare fmt -w <file> [<file> ...]         # short form
```

`declare fmt` accepts only filesystem paths (not git-revision specs):
the `--write` semantics on a git revision would be nonsensical.

### What canonical means

- Top-level keys appear in SPEC §2 order (`system`, `intent`,
  `invariants`, `assumptions`, `contracts`, `unconstrained`).
- Map entries inside `invariants:`, `assumptions:`, `contracts:`,
  and `unconstrained:` are sorted alphabetically by key.
- `intent.secondary` list order is preserved (lists are semantic).
- Multi-line strings use the literal block scalar (`|`); single-line
  strings use plain or double-quoted form per yaml.v3's defaults.
- Trailing whitespace is stripped from every line; the file ends
  with exactly one newline.
- Empty `invariants:` / `assumptions:` are emitted as `{}` (the
  SPEC §3 zero-state).
- Empty optional blocks (`contracts:`, `unconstrained:`) are
  omitted entirely.

### Properties

- **Idempotent.** `fmt(fmt(x))` is byte-identical to `fmt(x)`.
- **AST-preserving.** `fmt(x)` decodes to the same AST as `x`.
- **Lint-safe.** `fmt(x)` always lints cleanly if `x` did.
- **Refuses invalid input.** A file with lint errors is not
  formatted; `fmt` reports the lint issues and exits non-zero.

### What gets preserved across formatting

- Top-level head comments (e.g., a comment above `system:`).

### What does NOT get preserved (known limitation)

- Comments inside `invariants:`, `assumptions:`, `contracts:`, and
  `unconstrained:` map entries. Preserving them across formatting
  requires content-keyed identity, which is brittle when entries
  are renamed or reordered. If you have load-bearing prose that
  needs to survive `fmt`, put it in a top-level head comment or in
  the leaf body itself.

### Exit codes

| Code | Meaning                                                          |
| ---- | ---------------------------------------------------------------- |
| 0    | All inputs formatted successfully (and written, if `-w`).        |
| 1    | At least one input had lint errors, or `-w` write failed.        |

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

### Invocation

```bash
declare export <source>                    # canonical YAML to stdout (default)
declare export -f yaml <source>            # explicit YAML
declare export -f json <source>            # compact one-line JSON
```

`<source>` may be a filesystem path or a git revision spec (see
[§1a "Source resolution"](#git-revision-sources)).

### YAML format

The output of `declare fmt`, **with all comments stripped**. This is
the form to hand to a fresh agent: byte-stable for the same AST,
free of editorial chatter, and densely packed in the YAML idioms
LLMs handle natively.

Two agents that export the same `.dx` will produce byte-identical
output, so they can agree on a content hash without coordinating.

### JSON format

A compact one-line JSON projection of the AST. Best for non-LLM
consumers (other tools, structured-input sub-agents, automated
checks):

```json
{"system":"hello-world","intent":{"primary":"...","secondary":["..."]},...}
```

Properties:

- Object keys are emitted in declaration order at the top level
  (system, intent, invariants, assumptions, contracts,
  unconstrained), matching SPEC §2.
- Map keys inside each block are sorted alphabetically.
- HTML-escaping is disabled (`<`, `>`, `&` appear literally rather
  than as `\u003c` etc.) for token efficiency.
- Output ends with exactly one newline.
- Required `invariants:` / `assumptions:` always appear as `{}`
  when empty (preserves the SPEC §3 zero-state); empty optional
  blocks are omitted.

### When to use which

| Situation                                          | Format  |
| -------------------------------------------------- | ------- |
| Handing the spec to a coding agent or LLM         | `yaml`  |
| Piping into `jq` / a tool / a non-LLM consumer    | `json`  |
| Computing a content hash to coordinate two agents | either, but pick one and stick with it |

## 5a. `declare contracts list`

### Invocation

```bash
declare contracts list <source>            # one ID per line, alphabetical
declare contracts list -v <source>         # adds a one-line preview of given/when/then
declare contracts list -f json <source>    # full-fidelity JSON object
```

`<source>` may be a filesystem path or a git revision spec (see
[§1a "Source resolution"](#git-revision-sources)).

### Behavior

- **Text output (default).** One contract identifier per line, in
  alphabetical order. No trailing newline if there are zero
  contracts -- so a `for c in $(declare contracts list ...)` loop
  naturally does nothing for a spec with no `contracts:` block.
- **Verbose text (`-v`).** Each ID is followed by indented `given:`,
  `when:`, `then:` lines showing the first non-empty line of each
  clause; multi-line bodies get a trailing `…` to signal
  truncation. Always exactly four lines per contract.
- **JSON (`-f json`).** A single object: `{"contracts":[{"name":...,
  "given":...,"when":...,"then":...}]}` followed by one newline.
  Bodies are full-fidelity (multi-line preserved verbatim). Empty
  contracts:  emits `{"contracts":[]}`. HTML escaping is disabled
  so `<name>` appears literally instead of `\u003cname\u003e`.

### When to use which

| Situation                                                | Form         |
| -------------------------------------------------------- | ------------ |
| Pipe into a shell loop                                   | text         |
| Quick human scan of which contracts exist                | text + `-v`  |
| Feed full bodies to a runner or sub-agent                | `-f json`    |
| Compute a content hash of the contract enumeration       | `-f json`    |

### Exit codes

| Code | Meaning                                                          |
| ---- | ---------------------------------------------------------------- |
| 0    | Source decoded; output written (possibly empty in text mode).    |
| 1    | Source had lint errors, or the format flag was unrecognized.     |

### Why this command exists in v0.1.0 (despite no `declare verify`)

The judge skill walks each contract by hand today. `declare contracts
list` lets that walk be driven by a deterministic enumeration rather
than by scrolling through `system.dx`. When `declare verify` lands in
v0.2 it will land here as `declare contracts run`, sharing the same
parent command and inheriting the same alphabetical ordering.

## 5b. `declare verify` (deferred to v0.2)

There is no `declare verify` command in v0.1.0. SPEC §4 explains why:
contract execution is intentionally human/agent-driven for the first
release, performed by an agent operating under the `judge` skill.

If you find yourself wanting to write `declare verify`, instead:

1. Run `declare contracts list <source>` to get a deterministic,
   alphabetical enumeration of every contract you need to check.
2. Load the `judge` skill.
3. For each ID from step 1, walk that contract by hand (or via your
   agent runtime's tool-use): set up `given`, trigger `when`,
   evaluate `then`.
4. Classify any failure per the judge's failure-classification rules.

A future `declare verify` will mechanize steps 1–4 against a strict
contract grammar; until that ships, the judge skill plus
`declare contracts list` are the contract.

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
