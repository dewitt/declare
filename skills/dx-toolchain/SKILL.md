---
name: dx-toolchain
description: |
  How to invoke the deterministic `dx` CLI (`lint`, `fmt`, `diff`,
  `export`) from inside any agent's event loop. Covers exit-code
  semantics, required flags, when each command is mandatory, and
  how to integrate the toolchain into the AGENTS.md verification
  loop. Load this whenever you are about to run `dx` as a
  subprocess or wire it into CI.
---

# The `dx` Toolchain

The `dx` binary contains **no LLM**. Every command is a
deterministic operation over the dx AST. This skill tells you when
to invoke each command and how to interpret its output.

Declarations are CommonMark files (canonical extension `.md`) per
SPECIFICATION.md §4 (v0.2.0).

## 1. Command Inventory

| Command                  | Status           | Purpose                                                       |
| ------------------------ | ---------------- | ------------------------------------------------------------- |
| `dx lint`           | implemented      | Validate declarations against SPEC structural rules.          |
| `dx fmt`            | implemented      | Canonicalize formatting (block order, alphabetized keys, fixed Given/When/Then). |
| `dx diff`           | implemented      | Emit a semantic ledger between two declarations.              |
| `dx export`         | implemented      | Emit the AST as canonical CommonMark (default) or compact JSON. |
| `dx contracts list` | implemented      | Enumerate the contract identifiers in a declaration.          |
| `dx verify`         | deferred         | Run the `## Contracts` block as a black-box test harness.     |

The current binary lives at `./cmd/dx`. Build with `go build ./...`.
For one-off invocations during development, prefer:

```bash
go run ./cmd/dx <subcommand> [args...]
```

## 1a. Source resolution: file paths and git revisions {#git-revision-sources}

Every command that takes a `<source>` argument (`lint`, `diff`,
`export`, `contracts list`) accepts the same two forms:

| Form              | Example                       | Resolution                                |
| ----------------- | ----------------------------- | ----------------------------------------- |
| Filesystem path   | `examples/hello.md`           | Read directly from disk.                  |
| Git revision spec | `HEAD:examples/hello.md`      | `git show <rev>:<path>`. Requires being inside a git working tree. |

The git-revision form mirrors `git show` syntax exactly. Anything
git accepts as `<rev>` works: a branch (`main:foo.md`), a tag
(`v0.2.0:SPECIFICATION.md`), a relative ref (`HEAD~3:system.md`),
an explicit SHA (`abc123:system.md`).

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
declaration in place:

```bash
# What did I change semantically since the last commit?
dx diff HEAD:system.md system.md

# How does the spec on main differ from this branch's spec?
dx diff main:system.md HEAD:system.md

# Did some prior version even lint cleanly?
dx lint v0.2.0:examples/hello.md
```

This obviates the previous `git show > /tmp/old.md && dx diff
/tmp/old.md system.md` shell dance.

### Failure modes

Resolution failures (bad rev, missing path-in-rev) surface git's
own diagnostic verbatim, prefixed with `git show <input>:`.

Empty rev (`:foo.md`) or empty path (`HEAD:`) is rejected before
any `git` call, with a `dx`-side `invalid revision spec`
diagnostic.

## 2. `dx lint`

### Invocation

```bash
dx lint <source> [<source> ...]
```

Accepts one or more sources. Each source may be either a
filesystem path (`examples/hello.md`) or a git revision spec
(`HEAD:examples/hello.md`, `HEAD~1:system.md`, `main:foo.md`),
mirroring `git show` syntax.

Reports each source's status to stdout (`<source>: ok`) and
per-issue diagnostics to stderr in the format
`<source>:<line>:<col>: <message>` (line/col omitted when
unknown).

### Exit codes

| Code | Meaning                                                       |
| ---- | ------------------------------------------------------------- |
| 0    | All inputs passed lint.                                       |
| 1    | At least one source had a structural issue, or I/O / git resolution failed. |

### What it checks (SPEC §4.2 / §4.3)

- **Structural layer rules:**
  - Exactly one `#` heading, first in the document, non-empty body.
  - Every `##` heading body is one of the seven canonical names
    (`Intent`, `Invariants`, `Assumptions`, `Contracts`,
    `Unconstrained`).
  - No `##` heading appears more than once.
  - Every `###` key heading sits inside a recognized `##` block.
  - `## Intent` contains no `###` keys (its body is either a
    single paragraph or an unordered list).
  - Each contract has all three of `**Given:**`, `**When:**`,
    `**Then:**`, each appearing at most once.
- **Required-block presence:** `## Intent` (with a non-empty
  body), `## Invariants`, and `## Assumptions` must all be
  present. The REQUIRED Invariants and Assumptions blocks MAY
  have zero `###` children; what is forbidden is omitting the
  `##` heading entirely.
- **Code-block immunity:** `#` characters inside fenced code blocks
  or other CommonMark leaf content are NOT treated as headings,
  thanks to a proper CommonMark parse pass.

### What it does **not** check

- Slug-format validation on the `#` heading body (treated as
  advisory; the architect's pruning pass should catch obvious
  violations).
- Category-prefix discipline on invariant IDs (advisory; enforced
  socially via skill review, not mechanically).

### When `dx lint` is mandatory

Per AGENTS.md §3 ("Verification Loop"):

- **Before** writing or generating code from a declaration.
- **After** any modification to a declaration, before declaring the
  task complete.
- **In CI**, against every declaration in the repo.

A non-zero `dx lint` exit means the spec is structurally
untrustworthy. Fix it (acting as `architect`) before running any
other tool.

## 3. `dx fmt`

### Invocation

```bash
dx fmt <file> [<file> ...]            # writes canonical output to stdout
dx fmt --write <file> [<file> ...]    # overwrites each input in place
dx fmt -w <file> [<file> ...]         # short form
```

`dx fmt` accepts only filesystem paths (not git-revision specs):
the `--write` semantics on a git revision would be nonsensical.

### What canonical means (SPEC §4.5)

- The `#` system heading appears on the first line of the document.
- `##` block headings appear in canonical order: Intent,
  Invariants, Assumptions, Contracts, Unconstrained.
- Within each `##` block, `###` key headings appear in ascending
  lexicographic order by identifier.
- `## Intent` body is canonicalized: a single-item intent
  renders as a paragraph; a multi-item intent renders as an
  unordered list in the architect's priority order.
- Within each `## Contracts` entry, sub-fields appear in fixed
  order: `**Given:**`, `**When:**`, `**Then:**`, regardless of the
  order the author wrote them.
- Empty REQUIRED blocks (`## Invariants`, `## Assumptions`) are
  emitted as a heading with no children, preserving the SPEC §4.3.4
  semantically-meaningful empty form.
- Empty optional blocks (`## Contracts`, `## Unconstrained`) are
  omitted entirely.
- Trailing whitespace is stripped from every line; the file ends
  with exactly one newline.
- One blank line separates every structural heading from the
  preceding content.

### Properties

- **Idempotent.** `fmt(fmt(x))` is byte-identical to `fmt(x)`.
- **AST-preserving.** `fmt(x)` decodes to the same AST as `x`.
- **Lint-safe.** `fmt(x)` always lints cleanly if `x` did.
- **Refuses invalid input.** A file with lint errors is not
  formatted; `fmt` reports the lint issues and exits non-zero.

### What does NOT get preserved (known limitation)

- HTML comments (`<!-- ... -->`) are not preserved across
  formatting in v0.2.0; the AST does not retain them. If you have
  load-bearing prose that needs to survive `fmt`, put it in the
  leaf body of a key (e.g., a paragraph under the relevant `###`
  heading) rather than in a comment.

### Exit codes

| Code | Meaning                                                          |
| ---- | ---------------------------------------------------------------- |
| 0    | All inputs formatted successfully (and written, if `-w`).        |
| 1    | At least one input had lint errors, or `-w` write failed.        |

## 4. `dx diff`

### Invocation

```bash
dx diff <old> <new>
```

Both `<old>` and `<new>` may be filesystem paths or git revision
specs (see §1a above).

Emits a **semantic ledger** of operations to stdout, one per line,
in SPEC §4.5 canonical block order:

```
[MUTATED] intent.primary
[PROMOTED] assumptions.cache_location -> invariants.interface_cache_path
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

The diff command does **not** require either input to lint
cleanly; an architect may legitimately diff a known-broken spec
against its fix. It does require both files to decode into a
`Declaration`.

### When to use it (vs. text diff)

Always, when communicating spec changes to a human or another
agent. This is the canonical mechanism for AGENTS.md §5
("Communication with Humans"): a text diff over CommonMark is
hostile to architectural review (a single block-reorder dominates
the output); the semantic ledger is built for it.

## 5. `dx export`

### Invocation

```bash
dx export <source>                        # canonical CommonMark to stdout (default)
dx export -f markdown <source>            # explicit
dx export -f json <source>                # compact one-line JSON
```

`<source>` may be a filesystem path or a git revision spec.

### Markdown format

The same canonical form `dx fmt` writes. Byte-stable for the same
AST; suitable for handing to a fresh agent.

Two agents that export the same declaration will produce
byte-identical output, so they can agree on a content hash without
coordinating.

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
  unconstrained), matching SPEC §4.5.
- Map keys inside each block are sorted alphabetically.
- HTML-escaping is disabled (`<`, `>`, `&` appear literally rather
  than as `\u003c` etc.) for token efficiency.
- Output ends with exactly one newline.
- Required `invariants` / `assumptions` always appear as `{}` when
  empty (preserves the SPEC §4.3 zero-state); empty optional
  blocks are omitted.

### When to use which

| Situation                                          | Format     |
| -------------------------------------------------- | ---------- |
| Handing the spec to a coding agent or LLM         | `markdown` |
| Piping into `jq` / a tool / a non-LLM consumer    | `json`     |
| Computing a content hash to coordinate two agents | either, but pick one and stick with it |

## 5a. `dx contracts list`

### Invocation

```bash
dx contracts list <source>            # one ID per line, alphabetical
dx contracts list -v <source>         # adds a one-line preview of given/when/then
dx contracts list -f json <source>    # full-fidelity JSON object
```

`<source>` may be a filesystem path or a git revision spec.

### Behavior

- **Text output (default).** One contract identifier per line, in
  alphabetical order. No trailing newline if there are zero
  contracts -- so a `for c in $(dx contracts list ...)` loop
  naturally does nothing for a spec with no `## Contracts` block.
- **Verbose text (`-v`).** Each ID is followed by indented `given:`,
  `when:`, `then:` lines showing the first non-empty line of each
  clause; multi-line bodies get a trailing `…` to signal
  truncation. Always exactly four lines per contract.
- **JSON (`-f json`).** A single object: `{"contracts":[{"name":...,
  "given":...,"when":...,"then":...}]}` followed by one newline.
  Bodies are full-fidelity (multi-paragraph preserved verbatim).
  Empty contracts: emits `{"contracts":[]}`. HTML escaping is
  disabled so `<name>` appears literally instead of
  `\u003cname\u003e`.

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

### Why this command exists (despite no `dx verify` yet)

The judge skill walks each contract by hand today. `dx contracts
list` lets that walk be driven by a deterministic enumeration
rather than by scrolling through `system.md`. When `dx verify`
lands it will live here as `dx contracts run`, sharing the same
parent command and inheriting the same alphabetical ordering.

## 5b. `dx verify` (deferred)

There is no `dx verify` command in v0.2.0. SPEC §3.8 explains why:
contract execution is intentionally human/agent-driven for now,
performed by an agent operating under the `judge` skill.

If you find yourself wanting to write `dx verify`, instead:

1. Run `dx contracts list <source>` to get a deterministic,
   alphabetical enumeration of every contract you need to check.
2. Load the `judge` skill.
3. For each ID from step 1, walk that contract by hand (or via
   your agent runtime's tool-use): set up `**Given:**`, trigger
   `**When:**`, evaluate `**Then:**`.
4. Classify any failure per the judge's failure-classification
   rules.

A future `dx verify` will mechanize steps 1–4 against a strict
contract grammar; until that ships, the judge skill plus
`dx contracts list` are the contract.

## 6. The Verification Loop (canonical sequence)

This is the loop every role-skill invokes when work touches both
the spec and the implementation.

```
1. dx lint <changed>.md                    # exit 0 required
2. <generate or modify implementation>
3. <build / compile the implementation>          # exit 0 required
4. <execute every contract in ## Contracts>      # all must pass
5. If any contract fails:
     - HANDOFF to judge for triage.
     - Judge classifies: implementation bug OR spec gap.
     - Implementation bug → fix code, return to step 3.
     - Spec gap         → HANDOFF to architect, return to step 1.
6. Done.
```

Skipping step 1 or step 4 is the failure mode `dx` exists to
prevent. Do not skip them under time pressure.

## 6a. Post-Merge Ritual

When a declaration is touched on multiple branches and merged, the
architect MUST run, in order:

1. `dx lint <merged>.md` — a textual three-way merge can produce
   structurally invalid CommonMark (a `##` heading duplicated
   across both sides, a `###` key that lost its enclosing `##`).
2. `dx diff <merge-base>.md <merged>.md` — surfaces every
   semantic operation introduced by the merge in one glance. A
   clean text-merge can still hide a semantic conflict (e.g., one
   branch demoted an invariant to `## Unconstrained` while the
   other tightened it).
3. Reconcile any conflict in the **spec**, not the implementation.
   Per AGENTS.md §1 the declaration leads.

This is the v0.2.0 stance per SPEC §3.9. A future revision may
introduce `dx merge` for AST-level structural merge; until then,
the architect runs the ritual manually after every merge that
touches a declaration.

## 7. CI Snippet (reference)

A minimal GitHub-Actions-style block, illustrative only:

```yaml
- name: Build dx
  run: go build -o ./bin/dx ./cmd/dx

- name: Lint all declarations
  run: |
    set -euo pipefail
    find . -name '*.md' -path '*/examples/*' -print0 | xargs -0 ./bin/dx lint
```

The `set -euo pipefail` is important: a missing pipefail will let
a broken `find` mask a real lint failure. Adjust the `find` to
match wherever your declarations live; `.md` is also the extension
for ordinary documentation so don't blindly lint every `.md` in
the tree.

## 8. Common Failure Modes

| Symptom                                                                 | Likely cause                                                  | Fix                                                  |
| ----------------------------------------------------------------------- | ------------------------------------------------------------- | ---------------------------------------------------- |
| `missing required #  system heading`                                    | No `# <slug>` at the top.                                     | Add `# <system-name>` as the first line.             |
| `missing required ## Invariants block`                                  | Heading omitted.                                              | Add `## Invariants` (heading alone is valid).        |
| `missing intent body under ## Intent`                                   | Forgot to write the intent itself.                            | Add a paragraph or an unordered list under `## Intent`. |
| `## Foo is not a canonical block heading`                               | Typo or made-up block name.                                   | Use one of the seven canonical names (Title Case).   |
| `duplicate ## Invariants block`                                         | The same block heading appears twice.                         | Merge the two sections.                              |
| ``### foo appears outside any recognized ## block``                     | A `###` key sits before its `##` parent or after an unknown `##`. | Move the key under a recognized block.           |
| `## Intent does not use ### keys`                                       | Used `### Something` under `## Intent`.                       | Replace the `###` heading with a paragraph or list item. |
| `### X collides with ### Y (both reduce to slug Z)`                     | Two distinct heading bodies that reduce to the same slug.     | Reword one of them so the slug differs.              |
| `contract X is missing **Then:** sub-field`                             | Contract section incomplete.                                  | Add the missing sub-field.                           |
| Lint passes but a contract fails immediately on a clean impl.           | Contract `**Then:**` references internal state, not output.   | Rewrite the contract (architect's job).              |

## 9. Anti-Patterns

- **Running the implementation without first linting the spec.**
  The spec may have drifted into an undecodable state during a
  previous edit.
- **Hand-rolling a JSON projection of a declaration** for
  downstream agents. Use `dx export -f json`.
- **Shelling out to `sed`/`awk` to mutate declarations.** Mutate
  via the `architect` skill and re-lint; ad-hoc text-editing tools
  don't enforce SPEC §4.2 structural rules.
- **Using a `####` heading for structural purposes.** Heading
  levels 4+ are reserved for leaf content; if you need
  fourth-level structure, the model is wrong, not the format.
