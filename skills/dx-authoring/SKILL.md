---
name: dx-authoring
description: |
  Authoritative reference for the dx language (v0.2.0). Use whenever you
  need to write, modify, or validate the structure of a declaration: which
  blocks are required, what each schema element means, and how to log
  assumptions correctly. Load this any time you are about to emit
  CommonMark into a `.md` declaration or answer a spec-conformance
  question.
---

# Authoring Declarations

This skill is the working reference for the dx language. The normative
source is `SPECIFICATION.md` at the repo root; this document is a
denser, more prescriptive version meant to be read by an agent before
each emission.

## 1. Physical Format (SPEC §4.1–§4.2)

| Rule                          | Allowed                          | Forbidden                  |
| ----------------------------- | -------------------------------- | -------------------------- |
| Serialization                 | CommonMark                       | YAML, JSON, other          |
| File extension                | `.md`                            | `.dx`, `.yaml`, `.yml`     |
| `#` system heading            | Exactly one, first non-comment   | More than one; not first   |
| `##` block headings           | The seven canonical names (Title Case) | Any other heading body |
| `###` key headings            | Free-form human prose, inside `##` blocks | Outside any `##` block; inside `## Intent` |
| `####`+ headings              | Allowed inside leaf prose        | At the structural layer    |
| Contract sub-fields           | `**Given** ... **When** ... **Then** ...` | Any other label form |
| Block ordering (SHOULD)       | Intent, Invariants, Assumptions, Contracts, Unconstrained | other |

**Why the two-tier grammar matters.** The structural layer (heading
levels 1–3 plus the Given/When/Then bold convention) is enforced
mechanically by the linter. Everything between two structural headings
is **opaque CommonMark leaf content**: ordinary prose, inline code,
links, tables, fenced code blocks, even `####+` headings for prose
sub-structure. The leaf layer is not interpreted by the toolchain.

When prose contains characters CommonMark interprets (asterisks,
underscores, backticks, square brackets), escape them per standard
CommonMark rules.

**The slug rule.** Each `###` heading body is human prose
(e.g., `### Single line on stdout`). The toolchain reduces every
heading body to a **slug** for AST keys, diff stability, and any
shell-script consumer: lowercase, ASCII alphanumerics preserved,
runs of other characters collapsed to a single underscore,
leading/trailing underscores trimmed. `Single line on stdout`
reduces to `single_line_on_stdout`. Two headings within the same
block that reduce to the same slug are a structural error.

## 2. Required vs. Optional Blocks

| Block             | Required | Empty allowed?                                   | Notes                              |
| ----------------- | -------- | ------------------------------------------------ | ---------------------------------- |
| `# <system>`      | Yes      | No                                               | Short phrase in heading body.       |
| `## Intent`       | Yes      | No (the body MUST be a paragraph or a list)      | One sentence or a priority list.    |
| `## Invariants`   | Yes      | Yes (heading with zero `###` children)           | Heading itself must exist.          |
| `## Assumptions`  | Yes      | Yes (heading with zero `###` children, semantically meaningful) | Heading itself must exist. |
| `## Contracts`    | No       | —                                                | Add as soon as a contract is real. |
| `## Unconstrained`| No       | —                                                | Use to prevent over-specification. |

Note: even when there are no invariants or assumptions, the `##`
**heading must be present**. The linter distinguishes "intentionally
empty" (heading present, no `###` children) from "forgot to write it
down" (heading missing entirely).

## 3. Block-by-Block Schema

### 3a. `# <system>`

The `#` level-1 heading body is the system name. A short phrase;
conventionally kebab-case (`# hello-world`) but a human-readable
form (`# Hello World`) is also valid. Treat as the namespace for
this declaration.

```markdown
# hello-world
```

### 3b. `## Intent`

The body of `## Intent` is either a single CommonMark paragraph
or an unordered list (priority order, most important first).
Choose one form; mixing them is a structural error.

A single-sentence intent:

```markdown
## Intent

Greet a user by name on standard output.
```

Or a priority list (when more than one intent matters):

```markdown
## Intent

- Greet a user by name on standard output.
- Be friendly.
- Exit cleanly.
```

Each item is a goal, not a task. "Be fast" is fine; "use a
thread pool" is not — that's implementation, and belongs nowhere
in a declaration.

### 3c. `## Invariants`

A block of `###` key sections. Each heading body is free-form
prose naming the invariant; the content under the heading is the
invariant body.

```markdown
## Invariants

### Single line on stdout

Writes a single UTF-8 line to stdout terminated by `\n`.

### Cold-start latency

Cold-start latency must remain under 50ms on commodity hardware.

### No network access

The implementation must not open any network sockets.
```

Multi-paragraph bodies are fine; they read as ordinary
CommonMark:

```markdown
### Complex rule

The first paragraph states the rule.

A subsequent paragraph elaborates with examples or edge cases.

Where useful, leaf prose may include a fenced code block to pin
observable output exactly:

` ` `
expected stdout line one
expected stdout line two
` ` `
```

**Category convention (SHOULD).** When grouping a long list of
invariants by concern, prefix the heading with the category
followed by a colon: `### Interface: single line on stdout`,
`### Performance: cold-start latency`, `### Security: no network
access`. The resulting slugs (`interface_single_line_on_stdout`,
etc.) are scannable in tool output and group together
alphabetically. Conventional categories: `Interface`,
`Performance`, `Security`, `Observability`, `Data`, `User
experience`. Invent new ones as needed but stay consistent within
a single declaration.

**Each invariant is a black-box statement.** It describes
observable system behavior, never internal architecture. "Uses a
Bloom filter" is not an invariant; "membership queries return
false-negative rate of 0" is.

### 3d. `## Assumptions`

The most important block in the system. Same shape as Invariants.

```markdown
## Assumptions

### Default output format

When neither `--json` nor `--text` is passed, the CLI defaults
to human-readable text. The spec does not pin the default, and
the original C++ implementation defaulted to text; preserving
that minimizes surprise for existing users of the legacy tool.
```

An assumption is a heuristic choice the agent had to make
because the human's intent + invariants did not uniquely
determine the answer. Every assumption entry must include **what
was decided** and **why**. The architect later promotes (move to
`## Invariants`) or rejects (rewrite the code and delete the
entry) each one.

An empty `## Assumptions` block (heading with zero `###`
children) is a meaningful state: it asserts "I made no
unrecorded heuristic choices." Use it deliberately.

### 3e. `## Contracts`

Black-box verification rules in given/when/then form. The three
sub-fields are introduced by paragraph-leading bold markers
exactly: `**Given**`, `**When**`, `**Then**`.

```markdown
## Contracts

### Greets a named user

**Given** the argument vector contains exactly one non-empty name.

**When** the binary is invoked.

**Then** stdout contains `Hello, <name>!\n` and the exit code is 0.
```

- All three sub-fields are REQUIRED per contract; a missing
  sub-field is a lint error.
- The body of a sub-field is the paragraph that begins with the
  bold marker, plus any subsequent paragraphs up to the next
  recognized marker or heading.
- `**Then**` clauses must be **observable** (stdout, exit code,
  file state, HTTP response, …). Never reference internal state.
- One contract = one observable outcome. If you need
  conjunctions, prefer multiple contracts.

### 3f. `## Unconstrained`

A block of `###` key sections; each heading body names the
degree of freedom, and the content under the heading is the
description of what was granted.

```markdown
## Unconstrained

### Implementation language

Any language with a stable POSIX runtime is acceptable.

### Storage backend

Choose any durable key-value store; SQLite is acceptable.
```

If you find yourself wanting to write "we don't care about X," X
belongs here — not in Invariants, not in a comment.

## 4. The Assumption-Logging Protocol (AGENTS.md §2)

Whenever you (the agent) face a choice not specified by
`## Intent` or `## Invariants`:

1. **Stop before emitting.** Do not write the choice into code or
   prose until step 4 is done.
2. **Pick a heading.** Use prose that names the choice
   (e.g., `### Default output format`). The toolchain will
   reduce it to a stable slug (`default_output_format`).
3. **Write the entry** in `## Assumptions` as a `### <heading>`
   section. The body must answer: *what did you decide* and
   *why was that the most defensible choice given the
   ambiguity*.
4. **Then proceed** with implementation.

Anti-pattern to avoid: appending the assumption *after* the
implementation is written. The whole point is that the
assumption is recorded **before** the heuristic leaks into code.

## 5. Pruning Heuristic (AGENTS.md §4)

Before adding any new invariant, ask:

- Could the human's stated intent be satisfied without this
  constraint?
- Is this a *requirement* of the system, or a *preference* of mine?
- Would relaxing this invariant change anything observable?

If any answer suggests the constraint isn't truly required, move
it to `## Unconstrained` (with a description) or omit it
entirely.

## 6. Self-Validation Checklist

Before considering a declaration write complete:

- [ ] `dx lint <file>.md` exits 0.
- [ ] Exactly one `#` heading, first in the document, with a
      non-empty body.
- [ ] Every `##` heading body is one of: `Intent`, `Invariants`,
      `Assumptions`, `Contracts`, `Unconstrained`.
- [ ] No `##` heading appears more than once.
- [ ] Every `###` key sits inside a recognized `##` block (and
      not inside `## Intent`).
- [ ] `## Intent` has a non-empty body (a paragraph or a list).
- [ ] No two `###` headings in the same block reduce to the
      same slug.
- [ ] Every contract has all three of `**Given**`, `**When**`,
      `**Then**`, each appearing at most once.
- [ ] Every invariant is a *black-box* statement (no
      implementation details).
- [ ] Every contract `**Then**` clause is observable.
- [ ] Every assumption has a *why*, not just a *what*.
- [ ] Anything the human didn't constrain lives in
      `## Unconstrained` or not at all.

## 7. Worked Example

A minimal but complete declaration demonstrating every block:

```markdown
# hello-world

## Intent

- Greet a user by name on standard output.
- Be friendly.
- Exit cleanly.

## Invariants

### Single line on stdout

Writes a single UTF-8 line to stdout terminated by `\n`.

### Cold-start latency

Cold-start latency must remain under 50ms on commodity hardware.

## Assumptions

### Greeting format

The greeting is "Hello, <name>!" — the spec does not pin the
punctuation or word choice, and this matches the canonical
POSIX-tutorial form.

## Contracts

### Greets a named user

**Given** the argument vector contains exactly one non-empty name.

**When** the binary is invoked.

**Then** stdout contains `Hello, <name>!\n` and the exit code is 0.

## Unconstrained

### Implementation language

Any language with a stable POSIX runtime is acceptable.
```

A real example lives at `examples/hello.md`.
