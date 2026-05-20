# Journey: Port a Program to Another Language

**Goal:** You have a working program in language A. You want a
behaviorally equivalent program in language B. You want the port to
be auditable, language-agnostic, and produce a declaration you can
keep using going forward to govern the new implementation.

**Time budget (rough):** 30–90 minutes for a small CLI;
half a day for a few-thousand-line service.

**Prerequisites:**

- A working installation of the `dx` CLI on `$PATH`. See the
  [README quick-start](../../README.md#build) for build instructions.
- A coding agent with file-system tools (read, write, run shell
  commands). The journey works with any agent that can be pointed at
  a Markdown skills directory; concrete examples below use Claude
  Code.
- The source tree of the program you want to port, in a clean git
  workspace.
- Optionally, a runnable copy of the original program plus any
  end-to-end tests it ships with — invaluable for the Judge phase.

## TL;DR

```
agent loads skills/dx-orchestrator/SKILL.md
  → archaeologist  : reads legacy code, writes v0 system.md
  → architect      : prunes / promotes assumptions, ratifies the spec
  → implementer    : reads only system.md, generates impl_<lang>/
  → judge          : runs every ## Contracts entry against impl_<lang>/
  → loop until clean
```

The same `system.md` governs both implementations from then on.

## 0. One-time setup

### 0a. Install dx

```bash
git clone https://github.com/dewitt/dx /tmp/dx
cd /tmp/dx && go build -o ./bin/dx ./cmd/dx
dx --version
```

### 0b. Wire up your agent

Copy, symlink, or `install` the `skills/` tree from this repo into
wherever your agent loads skills from. The exact mechanism varies by
runtime:

| Agent runtime         | How to install                                                                       |
| --------------------- | ------------------------------------------------------------------------------------ |
| Claude Code (CLI)     | `cp -r skills/* ~/.claude/skills/` (or use a per-project `.claude/skills/` symlink) |
| Gemini CLI            | `for s in skills/*/; do echo "" \| gemini skills link "$s"; done` — uses the built-in `gemini skills link` command, which symlinks each skill so updates to the source are reflected immediately. |
| Cursor                | Workspace `.cursorrules` referencing the skill files                                 |
| Aider                 | `.aider.conf.yml` `read:` list pointing at `skills/*/SKILL.md`                       |
| Generic / unknown     | Open `skills/dx-orchestrator/SKILL.md` and paste it as a system prompt; instruct the agent to read sibling `SKILL.md` files when routing requires it. |

The skills are deliberately written as portable Markdown with
Anthropic-style YAML frontmatter; any agent that can read Markdown
playbooks can use them.

**Concrete example (Claude Code):**

```bash
mkdir -p ~/.claude/skills
cp -r /tmp/dx/skills/* ~/.claude/skills/
```

**Concrete example (Gemini CLI):**

```bash
# `gemini skills link` prompts for confirmation; piping empty stdin
# accepts the default ('Y') in headless contexts.
for s in /tmp/dx/skills/*/; do
  echo "" | gemini skills link "$s"
done
gemini skills list   # confirm all seven are Enabled
```

Then start a session in your target project and prompt:

> "Read `skills/dx-orchestrator/SKILL.md` and follow it. We are
> doing the port-to-another-language journey from
> `docs/journeys/port-to-another-language.md` of the dx repo."

### 0c. Headless / non-interactive mode pitfalls

If you intend to drive the journey from a script (or another agent
loop) rather than an interactive REPL, most agent runtimes have an
extra-flag dance to skip the per-action confirmation prompts that
they show by default. Some examples:

| Runtime         | Headless invocation                                                  |
| --------------- | -------------------------------------------------------------------- |
| Claude Code     | `claude -p "<prompt>"` (auto-approves edits in headless mode).       |
| Gemini CLI      | `gemini --yolo --skip-trust --prompt "<prompt>"` — both flags are required: `--yolo` auto-approves tool calls, **and** `--skip-trust` is needed because gemini-cli refuses YOLO mode in untrusted folders by default. Without `--skip-trust` the command exits with a trusted-folders error. |
| Other runtimes  | Consult `<runtime> --help` for the equivalent of "auto-approve all tools" and "skip the workspace-trust prompt". The two often have to be combined explicitly. |

Symptom of forgetting these flags: the command appears to hang. It is
actually waiting for a confirmation you can't see (because stdout is
captured) or has already exited with a permissions error you missed.
If the journey appears stuck, run the same prompt interactively first
to surface any prompts the runtime is showing you.

## 1. Prepare the workspace

In the project you want to port, create a sibling layout that mirrors
[`examples/weather_cli/`](../../examples/weather_cli/) of the `dx`
repo:

```
your_project/
├── system.md               # will be created in step 2
├── impl_<source_lang>/     # the existing implementation
│   └── ...                 # (move existing source here if needed)
└── impl_<target_lang>/     # empty for now; the implementer fills it
```

Why move the existing source under `impl_<source_lang>/`? Two reasons:

1. It frames both implementations as siblings under a shared spec,
   reinforcing that neither is privileged.
2. It makes step 4 (the **implementer must not peek at the original
   source**) enforceable by simple convention: the implementer's
   working directory is `impl_<target_lang>/` and the agent is told
   to treat `impl_<source_lang>/` as off-limits.

Commit this layout before proceeding. Each subsequent phase should
land as its own commit so `dx diff` and `git diff` together
form an auditable trail.

## 2. Archaeologist phase: extract `system.md`

Load the [`archaeologist`](../../skills/archaeologist/SKILL.md) skill
and prompt:

> "Read every file under `impl_<source_lang>/`. Distill the program's
> *observable* behavior into a `system.md` file at the project root.
> Follow the archaeologist skill exactly. When you must guess, log
> the guess as a `### <id>` section under `## Assumptions`; do not
> embed it silently."

What the agent should produce:

- A `system.md` with `# <system>`, `## Intent`, `## Invariants`,
  `## Assumptions`, and (ideally) some `## Contracts` entries
  derived from any tests the source ships with.
- An `## Unconstrained` block listing the implementation choices
  that were arbitrary in the original (storage backend, internal
  threading model, output formatting nuances, …).

What to verify before moving on:

```bash
dx lint system.md                # must exit 0
git add system.md && git commit -m "Archaeologist: extract v0 spec from impl_<source_lang>/"
```

**Smell tests:**

- **Empty `## Assumptions`** is almost always a lie. Real
  archaeology involves guesses. If the agent produced a
  `## Assumptions` heading with zero `###` children, push back:
  "What did you have to infer? Restate those as assumptions."
- **Invariants that name a data structure or library** (`uses a
  Bloom filter`, `imports requests`) are not invariants — they're
  implementation details that leaked through. Send them back to be
  rewritten as black-box statements ("membership queries return in
  O(log n) time", "issues HTTP GET requests over TLS").
- **Contracts that reference internal state** are unverifiable.
  Push back: "Rewrite this `**Then**` clause to reference only
  stdout / exit code / file system / HTTP response."

## 3. Architect phase: ratify and prune the spec

Load the [`architect`](../../skills/architect/SKILL.md) skill. For
small specs (a handful of assumptions and invariants, ~50 lines or
less), the architect pass is often faster as a direct hand-edit by
the human reviewer than as an agent-driven multi-turn loop. The
architect's work is judgment-heavy and the agent has no information
the human reviewer doesn't already have for a small spec.

For larger specs (5+ assumptions, 7+ invariants is the rough
threshold), drive the pass in three smaller, action-oriented turns
instead. Single mega-instructions ("review every entry, decide
promote/demote/reject for each, prune the invariants") tend to hang
or time out on current models against a realistic spec, and produce
worse output than focused turns when they do return:

**Turn 1 — assumption triage:**

> "Read `system.md`. For each entry under `## Assumptions`,
> recommend one of: promote (give a category-prefixed invariant
> ID), demote (to `## Unconstrained`), or reject (delete; we'll fix
> the code instead). Output a numbered list; do not edit the file
> yet."

Read the recommendations, intervene where you disagree, then:

**Turn 2 — apply the triage:**

> "Edit `system.md` to apply: promote A as `### Interface: X`,
> promote B as `### Performance: Y`, demote C, leave D as an
> assumption. Then run `dx lint system.md` and report the
> result."

**Turn 3 — invariant pruning pass:**

> "For each entry currently in `## Invariants`, ask: would
> relaxing this change anything observable to a user of the
> system? If no, recommend demoting to `## Unconstrained` or
> deleting. Output a numbered list; do not edit the file yet."

Apply the pruning recommendations the same way as turn 2. Repeat the
three-turn cycle until the spec settles.

This is the human's most important checkpoint. You are deciding what
the *next* implementer is allowed to assume, what they're allowed to
change, and what they cannot touch.

After each round of edits:

```bash
dx lint system.md                       # must exit 0
dx diff HEAD:system.md system.md        # see what you changed semantically
git add system.md && git commit -m "Architect: <describe the semantic change>"
```

You are done with this phase when:

- `## Assumptions` contains only entries you *consciously* want
  to leave for the next implementer to handle.
- Every `## Invariants` entry survives the pruning pass.
- Every `## Invariants` entry is testable as a black box (the
  Judge needs to be able to verify it).
- `## Contracts` covers the load-bearing behaviors. Five
  well-chosen contracts beat fifty redundant ones.

**Don't skip this phase.** A v0 spec straight from the archaeologist
is biased toward the original implementation's idiosyncrasies. The
architect pass is what makes the spec language-agnostic.

## 4. Implementer phase: synthesize the new code

Load the [`implementer`](../../skills/implementer/SKILL.md) skill,
open a fresh session if you can (so the agent has no memory of the
source code from earlier phases), and prompt:

> "Read only `system.md`. Do **not** read anything under
> `impl_<source_lang>/`. Generate a complete implementation in
> `<target_language>` under `impl_<target_lang>/` that satisfies
> every entry in `## Invariants` and every contract in `## Contracts`.
> Use the language's native idioms; you do not need to mimic the
> original's structure. When the spec is ambiguous, append a
> `### <id>` section to `## Assumptions` in `system.md` *before*
> writing the code that makes the assumption."

The "do not read the source" instruction is doing real work here.
The whole point of the port is to prove the spec is sufficient. If
the implementer peeks, the new code inherits the original's
biases — and you've reduced the journey to a translation pass.

**Operational note (current gap):** No tool today *enforces* the
no-peeking rule. It is honor-system, mediated by the prompt and by
keeping `impl_<source_lang>/` out of the implementer session's open
files. A future `dx` may grow a sandboxing primitive
(see *Known gaps* below).

**Operational note (storage isolation):** if the system under port
persists state to a default location (a file in `~`, a database in
`/var`, a socket, etc.), the implementer will happily write there
during smoke tests — and so will the legacy implementation when you
re-run it for comparison. The two will silently mix and you'll spend
an hour debugging a "judge failure" that turns out to be cross-impl
state leakage. Mitigate by isolating storage for the duration of the
journey:

```bash
# If the system honors an env var (the common case):
export TODO_FILE=/tmp/dx-port-scratch.json

# If it doesn't, override the default location at the OS level
# (containers, fakeroot, a chroot, or just `cd` into a clean dir for
# both implementations).
```

Treat this as part of the workspace setup in step 1 if your spec has
an `Interface:`-categorized invariant for an env-var override or a similar
configuration knob — apply it to both implementations during testing.

After the implementer finishes:

```bash
# Build the new implementation in its native toolchain.
cd impl_<target_lang> && <build command>

# Re-lint the spec — the implementer may have appended assumptions.
dx lint system.md
dx diff HEAD:system.md system.md

git add . && git commit -m "Implementer: generate impl_<target_lang> from system.md"
```

If the implementer logged new assumptions, you have two options:

- **Cheap and fast:** accept them and move on; the architect will
  ratify them in a follow-up pass.
- **Strict:** loop back to step 3, ratify each new assumption, and
  re-run the implementer. Better for high-stakes ports.

## 5. Judge phase: verify against the contracts

Load the [`judge`](../../skills/judge/SKILL.md) skill and prompt:

> "Walk every entry in `## Contracts` against the new
> implementation under `impl_<target_lang>/`. For each contract:
> set up the `**Given**`, trigger the `**When**`, observe the
> outcome, compare to the `**Then**`. Report PASS/FAIL for each.
> For FAILs, classify as implementation bug, spec gap, or intent
> mismatch per the judge skill."

**Operational note (current gap):** there is no `dx verify`
command in v0.2.0 (deferred per
[SPEC §3.8](../../SPECIFICATION.md#38-conformance)). The judge
**is** the contract executor today: an agent walks each contract
by hand or via its tool-use. This works fine for a handful of
contracts; it does not scale to dozens. The biggest priority gap
in this journey is mechanizing this step. See *Known gaps*
below.

Cross-check (recommended): run the same contracts against the
*original* `impl_<source_lang>/` implementation. If a contract fails
on the original, it's almost certainly a spec gap, not an
implementation bug — the archaeologist over-specified.

For each FAIL, the routing is:

| Verdict                | Send to       | Action                                                |
| ---------------------- | ------------- | ----------------------------------------------------- |
| FAIL (impl bug)        | implementer   | Fix the new code. Loop back to step 5.                |
| FAIL (spec gap)        | architect     | Tighten or relax the contract / invariant. Loop to 4. |
| FAIL (intent mismatch) | architect     | Spec contradicts itself. Reconcile. Loop to 3.       |
| PASS                   | —             | Move on to the next contract.                         |

## 6. Done — what you have now

- `system.md` — a language-agnostic, version-controlled spec for the
  program.
- `impl_<source_lang>/` — the original implementation, preserved for
  comparison.
- `impl_<target_lang>/` — the new implementation, demonstrably
  satisfying every contract.
- A git history that reads like a design conversation:
  *"Archaeologist extracted v0 → Architect promoted assumptions
  X, Y → Implementer generated Rust → Judge found impl bug Z →
  Implementer fixed Z → Judge clean."*

From here on, the spec leads. New features land as architect commits
to `system.md` first; the implementer then catches both
implementations up.

## Known gaps in this journey (priority TODOs)

The following are real, blocking-or-painful gaps in v0.2.0. Each
one is a candidate priority TODO for a future `dx` revision. They
are listed in roughly the order they bite an end-user trying to
follow this journey today.

### Gap 1 — No `dx verify` (high priority)

**Where it bites:** step 5 (Judge phase).

**Symptom:** you have 30 contracts, and the judge has to walk each
one by prose. By the time you've executed contract 30 you've
forgotten the setup for contract 1, and the audit trail is buried
in chat history.

**What's needed:** a `dx verify <system>.md --impl <command>`
command that:

1. Parses `## Contracts` into a structured execution plan.
2. For each contract, sets up `**Given**` (via a small embedded
   grammar: env vars, files, args), runs `**When**` (executes
   `<command>` with the right args), evaluates `**Then**` (matches
   stdout/stderr/exit code/file state).
3. Emits a deterministic pass/fail summary plus per-contract
   diagnostics.

This requires designing a contract grammar that's expressive
enough for real-world preconditions but constrained enough to
stay language-agnostic. SPEC §3.8 explicitly defers this.

### Gap 2 — No mechanism to enforce "implementer must not read the source" (medium priority)

**Where it bites:** step 4 (Implementer phase).

**Symptom:** the no-peeking rule is honor-system. A diligent
agent honors it; a less diligent one quietly inherits the original's
quirks and the journey reduces to a translation pass.

**What's needed:** at minimum, a documented convention (e.g., a
`.dx-implementer-allowlist` file the agent runtime respects).
At maximum, a sandboxing primitive — though that crosses into
the agent-runtime layer, which is explicitly out of scope for
the `dx` binary itself.

## Working example

[`examples/weather_cli/`](../../examples/weather_cli/) in this repo
is a fully-worked instance of this journey:

- `impl_cpp/weather_cli.cc` — the source-language artifact (the
  archaeologist's input).
- `system.md` — the spec extracted by the archaeologist and ratified
  by the architect.
- `impl_python/weather_cli.py` — the target-language artifact
  generated by the implementer from `system.md` alone.

The README in that directory walks through which artifact
corresponds to which phase of this journey. Both implementations
satisfy the four directly-runnable contracts manually; the fifth
(`caches_repeat_queries`) needs `dx verify` to be checked
mechanically — a perfect illustration of Gap 1 above.

## Related reading

- [`AGENTS.md`](../../AGENTS.md) — the universal rules every agent
  follows in this repo, including the verification loop and the
  post-merge ritual.
- [`SPECIFICATION.md`](../../SPECIFICATION.md) — the normative dx language reference.
- [`skills/dx-orchestrator/SKILL.md`](../../skills/dx-orchestrator/SKILL.md)
  — the meta-routing skill an agent loads on entering a
  dx-managed repo.
- The four role-skills referenced throughout this journey:
  [archaeologist](../../skills/archaeologist/SKILL.md),
  [architect](../../skills/architect/SKILL.md),
  [implementer](../../skills/implementer/SKILL.md),
  [judge](../../skills/judge/SKILL.md).

## Appendix: Driving the agent runtime

The journey above describes *what* to ask each role to do.
This appendix captures *how* — the runnable invocations and
prompt patterns that work in practice. The forms below have
been exercised end-to-end against Gemini CLI on real codebases
(see this journey's Working example, and a follow-up test
porting a C++ template-iterator library to Go).

### Setup once per machine

Per [§0a (Install dx)](#0a-install-dx) and [§0b (Wire up your
agent)](#0b-wire-up-your-agent), but condensed:

```bash
# Build and install the dx CLI.
cd /tmp && git clone https://github.com/dewitt/dx && cd dx
go build -o ~/bin/dx ./cmd/dx

# Link all seven skills into Gemini CLI. The `echo "" |` accepts
# the default confirmation in headless mode.
for s in /tmp/dx/skills/*/; do
  echo "" | gemini skills link "$s"
done

# Confirm.
gemini skills list | grep -E "^(dx-|archaeologist|architect|implementer|judge) "
```

### The Gemini CLI invocation

Every per-phase prompt in this appendix uses the same wrapper:

```bash
gemini --yolo --skip-trust --prompt "<phase prompt here>"
```

`--yolo` auto-approves tool calls; `--skip-trust` bypasses the
trusted-folder check that otherwise refuses YOLO mode in
unfamiliar directories. Both are required for headless
operation. Without them the command appears to hang because it
is silently waiting on a confirmation you cannot see (see
[§0c (Headless / non-interactive mode pitfalls)](#0c-headless--non-interactive-mode-pitfalls)).

### Archaeologist phase (worked prompt)

```bash
cd <project-root>
gemini --yolo --skip-trust --prompt "Read every file under \
impl_<source_lang>/. Distill the program's observable behavior \
into a system.md file at the project root (<absolute-path>). \
Follow the archaeologist skill exactly. When you must guess, \
log the guess as a ### <id> section under ## Assumptions; do \
not embed it silently."
```

Two notes about this prompt:

- It names the absolute path of the output file. Without this, the
  agent sometimes asks where to write or writes to a guessed path.
- It instructs explicitly that guesses go in `## Assumptions`.
  The archaeologist skill says this too, but stating it in the prompt
  reinforces the discipline and surfaces problems faster if the
  skill isn't fully loaded.

After the agent emits its `HANDOFF: archaeologist → architect`
line, validate and commit:

```bash
dx lint system.md              # must exit 0
git add system.md && git commit -m "Archaeologist: extract v0 spec"
```

### Architect phase (worked prompts)

For small specs, hand-edit (see §3 in this journey).

For larger specs, the three-turn pattern in §3 uses three
`gemini` invocations of the same shape, with the prompts from
§3 turns 1, 2, and 3. Each invocation is independent; you read
the output between turns and decide what to do.

```bash
cd <project-root>
gemini --yolo --skip-trust --prompt "<turn-N prompt from §3>"
```

After applying the architect's changes (typically by your own
hand-edit, sometimes by an agent edit you direct):

```bash
dx lint system.md
dx diff HEAD:system.md system.md     # see what you changed
git add system.md && git commit -m "Architect: <semantic change>"
```

### Implementer phase (worked prompt)

```bash
cd <project-root>
mkdir -p impl_<target_lang>
gemini --yolo --skip-trust --prompt "You are the implementer per \
the implementer skill. Read ONLY system.md in the current \
directory. Do NOT read anything under impl_<source_lang>/. \
Generate a complete <target_lang> implementation under \
impl_<target_lang>/ that satisfies every entry in ## Invariants \
and every contract in ## Contracts. Use <target_lang>'s native \
idioms; do not mimic the original <source_lang> structure. \
Place the main file at impl_<target_lang>/<filename> and a \
build/dependency manifest at impl_<target_lang>/<manifest>. \
When the spec is ambiguous, append a ### <id> section under \
## Assumptions in system.md BEFORE writing the code that makes \
the assumption. After writing, run the build and report any errors."
```

Three things this prompt does that bare invocations miss:

- Names the source-language directory explicitly as off-limits.
  This is the project's main no-peeking discipline; relying on
  the implementer skill to say it is not enough.
- Names the target-language file paths so the agent doesn't
  guess at conventions for that ecosystem.
- Tells the agent to log assumptions *before* writing code, not
  after. Logging after is the failure mode that hides silent
  invention behind a reverse-engineered explanation.

After the implementer's handoff, build and commit:

```bash
cd impl_<target_lang> && <build command>     # must succeed
dx lint ../system.md                          # implementer may have appended assumptions
git add . && cd .. && git commit -m "Implementer: generate <target_lang> port"
```

### Judge phase (worked prompt)

```bash
cd <project-root>
gemini --yolo --skip-trust --prompt "You are the judge per the \
judge skill. Walk every contract in system.md against the \
implementation under impl_<target_lang>/. Use 'dx contracts list \
system.md' first to enumerate. For each contract: set up \
**Given**, trigger **When**, evaluate **Then**. Emit PASS or \
FAIL per contract per the judge skill format, with classification \
for any FAIL."
```

The judge typically does its work in a single turn, returning a
JUDGEMENT block followed by per-contract handoffs to architect or
implementer for any FAIL. If the judge returns all-PASS, the
journey is done; commit nothing further.

### What still doesn't work via this invocation

- **Multi-turn agent sessions** are not supported by `gemini
  --prompt`. Each invocation is independent. For workflows
  that benefit from session memory (long iterative architect
  passes, debugging dialogs), drive `gemini` interactively.
- **Tool-use beyond filesystem read/write/exec** is sometimes
  brittle in YOLO mode. Network requests and other extensions
  may need explicit confirmation that `--yolo` doesn't cover.
- **Error reporting from the agent is uneven.** A failed
  command may surface as a vague "I encountered an issue"
  rather than a stack trace. When this happens, drop into
  interactive `gemini` to retry the same prompt and watch the
  tool-call output directly.

### Other agent runtimes

Claude Code's headless invocation is `claude -p "<prompt>"`
with auto-approval enabled by default. Cursor and other agent
runtimes have their own conventions. The prompts above transfer;
the wrapper changes per runtime.
