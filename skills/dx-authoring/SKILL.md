---
name: dx-authoring
description: |
  Authoritative reference for the `.dx` language (v0.1.0). Use whenever you
  need to write, modify, or validate the structure of a `.dx` file: which
  blocks are required, which YAML features are forbidden, what each schema
  field means, and how to log assumptions correctly. Load this any time you
  are about to emit YAML into a `.dx` file or answer a spec-conformance
  question.
---

# Authoring `.dx` Files

This skill is the working reference for the `.dx` language. The normative
source is `SPEC.md` at the repo root; this document is a denser, more
prescriptive version meant to be read by an agent before each emission.

## 1. Physical Format (SPEC §1–§2)

| Rule                          | Allowed              | Forbidden               |
| ----------------------------- | -------------------- | ----------------------- |
| YAML version                  | 1.2                  | older                   |
| File extension                | `.dx`                | `.yaml`, `.yml`         |
| Anchors / aliases             | —                    | `&anchor`, `*alias`     |
| Custom tags                   | —                    | `!!binary`, `!!set`, …  |
| Multiline string scalar style | literal `\|`         | folded `>`              |
| Top-level key ordering        | `system`, `intent`, `invariants`, `assumptions`, `contracts`, `unconstrained` | any other order |

**Why the literal-scalar rule matters:** folded scalars (`>`) collapse
newlines into spaces in a way that varies subtly across YAML libraries and
LLM tokenizers. Literal (`|`) preserves the human-authored bytes exactly,
which is critical when contracts reference observable output.

When in doubt, prefer `|` even for single-line strings — it is always safe.

## 2. Required vs. Optional Blocks

| Block            | Required | Empty allowed?                | Notes                              |
| ---------------- | -------- | ----------------------------- | ---------------------------------- |
| `system`         | Yes      | No                            | Slug-format string.                |
| `intent`         | Yes      | No (`primary` is required)    | `secondary` is optional.           |
| `invariants`     | Yes      | Empty map allowed (`{}`)      | Key must exist.                    |
| `assumptions`    | Yes      | Empty map allowed (`{}`)      | Key must exist (zero-assumption).  |
| `contracts`      | No       | —                             | Add as soon as a contract is real. |
| `unconstrained`  | No       | —                             | Use to prevent over-specification. |

Note: even when there are no invariants or assumptions, the **keys must be
present** (with `{}`). The linter distinguishes "intentionally empty" from
"forgot to write it down."

## 3. Block-by-Block Schema

### 3a. `system`

Slug-format string identifying the declaration. Convention: kebab-case,
no leading numbers. Treat as the namespace for this `.dx` file.

```yaml
system: hello-world
```

### 3b. `intent`

```yaml
intent:
  primary: One sentence stating the core observable purpose.
  secondary:
    - One supporting goal per list entry.
    - Keep these short and goal-shaped, not implementation-shaped.
```

- `primary` is a string scalar. Required. Keep it on one line if you
  can; for longer rationale, use the literal block scalar (`|`).
- `secondary` is an optional list of strings. Each entry is a goal, not a
  task. "Be fast" is fine; "use a thread pool" is not — that's
  implementation, and belongs nowhere in `.dx`.

### 3c. `invariants`

A map from category-prefixed slug IDs to prose statements of
non-negotiable constraints.

```yaml
invariants:
  iface_stdout: Writes a single UTF-8 line to stdout terminated by `\n`.
  perf_startup_ms: Cold-start latency must remain under 50ms on commodity hardware.
  sec_no_network: The implementation must not open any network sockets.
```

Single-line invariants use plain scalars; multi-line ones use the
literal block scalar (`|`). Both forms decode to the same value, and
`dx fmt` chooses between them automatically — write whichever is
natural and let the formatter canonicalize.

```yaml
invariants:
  iface_complex_rule: |
    A multi-line invariant uses `|`. The body is preserved verbatim,
    line by line. Use this form when the constraint genuinely needs
    more than one sentence to express.
```

Conventional prefixes: `iface_`, `perf_`, `sec_`, `obs_` (observability),
`data_`, `ux_`. Invent new ones as needed but stay consistent within a
single `.dx` file.

**Each invariant is a black-box statement.** It describes observable
system behavior, never internal architecture. "Uses a Bloom filter" is
not an invariant; "membership queries return false-negative rate of 0"
is.

### 3d. `assumptions`

The most important block in the system. Same shape as `invariants`.

```yaml
assumptions:
  ast.intent_secondary_shape: |
    `intent.secondary` is modelled as a list of strings; the spec does
    not pin the shape, and a list is the natural fit for an
    enumeration of goals.
```

An assumption is a heuristic choice the agent had to make because the
human's intent + invariants did not uniquely determine the answer. Every
assumption entry must include **what was decided** and **why**. The human
later promotes (move to `invariants`) or rejects (rewrite the code and
delete the entry) each one.

Empty `assumptions: {}` is a meaningful state: it asserts "I made no
unrecorded heuristic choices." Use it deliberately.

### 3e. `contracts`

Black-box verification rules in given/when/then form.

```yaml
contracts:
  greets_named_user:
    given: The argument vector contains exactly one non-empty name.
    when: The binary is invoked.
    then: stdout contains "Hello, <name>!\n" and the exit code is 0.
```

- All three fields are string scalars (plain on one line, `|` for
  multi-line).
- `then` clauses must be **observable** (stdout, exit code, file state,
  HTTP response, …). Never reference internal state.
- One contract = one observable outcome. If you need conjunctions, prefer
  multiple contracts.

### 3f. `unconstrained`

A map from category to a description of the freedom granted.

```yaml
unconstrained:
  language: Any language with a stable POSIX runtime is acceptable.
  storage_backend: Choose any durable key-value store; SQLite is acceptable.
```

If you find yourself wanting to write "we don't care about X," X belongs
here — not in `invariants`, not as a comment.

## 4. The Assumption-Logging Protocol (AGENTS.md §2)

Whenever you (the agent) face a choice not specified by `intent` or
`invariants`:

1. **Stop before emitting.** Do not write the choice into code or prose
   until step 4 is done.
2. **Pick a stable ID.** Convention: `<file_or_module>.<short_phrase>`,
   e.g. `cli.default_output_format`. Reuse the ID forever; never rename.
3. **Write the entry** in `assumptions:`. The body must answer: *what
   did you decide* and *why was that the most defensible choice given
   the ambiguity*.
4. **Then proceed** with implementation.

Anti-pattern to avoid: appending the assumption *after* the implementation
is written. The whole point is that the assumption is recorded **before**
the heuristic leaks into code.

## 5. Pruning Heuristic (AGENTS.md §4)

Before adding any new invariant, ask:

- Could the human's stated intent be satisfied without this constraint?
- Is this a *requirement* of the system, or a *preference* of mine?
- Would relaxing this invariant change anything observable?

If any answer suggests the constraint isn't truly required, move it to
`unconstrained:` (with a description) or omit it entirely.

## 6. Self-Validation Checklist

Before considering a `.dx` write complete:

- [ ] `dx lint <file>.dx` exits 0.
- [ ] Every multiline string uses `|`, not `>`.
- [ ] No anchors (`&`) or aliases (`*`).
- [ ] Top-level keys appear in the canonical order.
- [ ] Required keys (`system`, `intent.primary`, `invariants`,
      `assumptions`) are all present.
- [ ] Every invariant ID has a category prefix.
- [ ] Every invariant is a *black-box* statement (no implementation
      details).
- [ ] Every contract `then` clause is observable.
- [ ] Every assumption has a *why*, not just a *what*.
- [ ] Anything the human didn't constrain lives in `unconstrained:` or
      not at all.

## 7. Worked Example

A minimal but complete `.dx` file demonstrating every block:

```yaml
system: hello-world
intent:
  primary: Greet a user by name on standard output.
  secondary:
    - Be friendly.
    - Exit cleanly.
invariants:
  iface_stdout: Writes a single UTF-8 line to stdout terminated by `\n`.
  perf_startup_ms: Cold-start latency must remain under 50ms on commodity hardware.
assumptions:
  greeting.format: |
    The greeting is "Hello, <name>!" — the spec does not pin the
    punctuation or word choice, and this matches the canonical
    POSIX-tutorial form.
contracts:
  greets_named_user:
    given: The argument vector contains exactly one non-empty name.
    when: The binary is invoked.
    then: stdout contains "Hello, <name>!\n" and the exit code is 0.
unconstrained:
  language: Any language with a stable POSIX runtime is acceptable.
```

A real example lives at `examples/hello.dx`.
