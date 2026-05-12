# Journey: Port a Program to Another Language

**Goal:** You have a working program in language A. You want a
behaviorally equivalent program in language B. You want the port to
be auditable, language-agnostic, and produce a `.dx` artifact you can
keep using going forward to govern the new implementation.

**Time budget (rough):** 30–90 minutes for a small CLI;
half a day for a few-thousand-line service.

**Prerequisites:**

- A working installation of the `declare` CLI on `$PATH`. See the
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
agent loads skills/declare-orchestrator/SKILL.md
  → archaeologist  : reads legacy code, writes v0 system.dx
  → architect      : prunes / promotes assumptions, ratifies the spec
  → implementer    : reads only system.dx, generates impl_<lang>/
  → judge          : runs every contracts: entry against impl_<lang>/
  → loop until clean
```

The same `system.dx` governs both implementations from then on.

## 0. One-time setup

### 0a. Install declare

```bash
git clone https://github.com/dewitt/declare /tmp/declare
cd /tmp/declare && go build -o ~/bin/declare ./cmd/declare
declare --version
```

### 0b. Wire up your agent

Copy, symlink, or `install` the `skills/` tree from this repo into
wherever your agent loads skills from. The exact mechanism varies by
runtime:

| Agent runtime         | How to install                                                                       |
| --------------------- | ------------------------------------------------------------------------------------ |
| Claude Code (CLI)     | `cp -r skills/* ~/.config/cloudcode/skills/` (or use a per-project `.claude/skills/` symlink) |
| Gemini CLI            | `for s in skills/*/; do echo "" \| gemini skills link "$s"; done` — uses the built-in `gemini skills link` command, which symlinks each skill so updates to the source are reflected immediately. |
| Cursor                | Workspace `.cursorrules` referencing the skill files                                 |
| Aider                 | `.aider.conf.yml` `read:` list pointing at `skills/*/SKILL.md`                       |
| Generic / unknown     | Open `skills/declare-orchestrator/SKILL.md` and paste it as a system prompt; instruct the agent to read sibling `SKILL.md` files when routing requires it. |

The skills are deliberately written as portable Markdown with
Anthropic-style YAML frontmatter; any agent that can read Markdown
playbooks can use them.

**Concrete example (Claude Code):**

```bash
mkdir -p ~/.config/cloudcode/skills
cp -r /tmp/declare/skills/* ~/.config/cloudcode/skills/
```

**Concrete example (Gemini CLI):**

```bash
# `gemini skills link` prompts for confirmation; piping empty stdin
# accepts the default ('Y') in headless contexts.
for s in /tmp/declare/skills/*/; do
  echo "" | gemini skills link "$s"
done
gemini skills list   # confirm all seven are Enabled
```

Then start a session in your target project and prompt:

> "Read `skills/declare-orchestrator/SKILL.md` and follow it. We are
> doing the port-to-another-language journey from
> `docs/journeys/port-to-another-language.md` of the declare repo."

### 0c. Headless / non-interactive mode pitfalls

If you intend to drive the journey from a script (or another agent
loop) rather than an interactive REPL, most agent runtimes have an
extra-flag dance to skip the per-action confirmation prompts that
they show by default. Some examples:

| Runtime         | Headless invocation                                                  |
| --------------- | -------------------------------------------------------------------- |
| Claude Code     | `cloudcode -p "<prompt>"` (auto-approves edits in headless mode).    |
| Gemini CLI      | `gemini --yolo --skip-trust --prompt "<prompt>"` — both flags are required: `--yolo` auto-approves tool calls, **and** `--skip-trust` is needed because gemini-cli refuses YOLO mode in untrusted folders by default. Without `--skip-trust` the command exits with a trusted-folders error. |
| Other runtimes  | Consult `<runtime> --help` for the equivalent of "auto-approve all tools" and "skip the workspace-trust prompt". The two often have to be combined explicitly. |

Symptom of forgetting these flags: the command appears to hang. It is
actually waiting for a confirmation you can't see (because stdout is
captured) or has already exited with a permissions error you missed.
If the journey appears stuck, run the same prompt interactively first
to surface any prompts the runtime is showing you.

## 1. Prepare the workspace

In the project you want to port, create a sibling layout that mirrors
[`examples/weather_cli/`](../../examples/weather_cli/) of the `declare`
repo:

```
your_project/
├── system.dx               # will be created in step 2
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
land as its own commit so `declare diff` and `git diff` together
form an auditable trail.

## 2. Archaeologist phase: extract `system.dx`

Load the [`archaeologist`](../../skills/archaeologist/SKILL.md) skill
and prompt:

> "Read every file under `impl_<source_lang>/`. Distill the program's
> *observable* behavior into a `system.dx` file at the project root.
> Follow the archaeologist skill exactly. When you must guess, log
> the guess as an `assumptions:` entry; do not embed it silently."

What the agent should produce:

- A `system.dx` with `system`, `intent`, `invariants`, `assumptions`,
  and (ideally) some `contracts:` entries derived from any tests the
  source ships with.
- An `unconstrained:` block listing the implementation choices that
  were arbitrary in the original (storage backend, internal
  threading model, output formatting nuances, …).

What to verify before moving on:

```bash
declare lint system.dx                # must exit 0
git add system.dx && git commit -m "Archaeologist: extract v0 spec from impl_<source_lang>/"
```

**Smell tests:**

- **Empty `assumptions:`** is almost always a lie. Real archaeology
  involves guesses. If the agent produced `assumptions: {}`, push
  back: "What did you have to infer? Restate those as assumptions."
- **Invariants that name a data structure or library** (`uses a
  Bloom filter`, `imports requests`) are not invariants — they're
  implementation details that leaked through. Send them back to be
  rewritten as black-box statements ("membership queries return in
  O(log n) time", "issues HTTP GET requests over TLS").
- **Contracts that reference internal state** are unverifiable. Push
  back: "Rewrite this `then` clause to reference only stdout / exit
  code / file system / HTTP response."

## 3. Architect phase: ratify and prune the spec

Load the [`architect`](../../skills/architect/SKILL.md) skill. **Do not
prompt with a single mega-instruction here** — current models tend to
hang or time out on broad "review every entry" prompts against a
realistic spec (5+ assumptions, 7+ invariants is enough to trigger
this). Drive the pass in three smaller, action-oriented turns instead:

**Turn 1 — assumption triage:**

> "Read `system.dx`. For each entry under `assumptions:`, recommend
> one of: promote (give a category-prefixed invariant ID), demote (to
> `unconstrained:`), or reject (delete; we'll fix the code instead).
> Output a numbered list; do not edit the file yet."

Read the recommendations, intervene where you disagree, then:

**Turn 2 — apply the triage:**

> "Edit `system.dx` to apply: promote A as `iface_x`, promote B as
> `perf_y`, demote C, leave D as an assumption. Then run
> `declare lint system.dx` and report the result."

**Turn 3 — invariant pruning pass:**

> "For each entry currently in `invariants:`, ask: would relaxing this
> change anything observable to a user of the system? If no, recommend
> demoting to `unconstrained:` or deleting. Output a numbered list;
> do not edit the file yet."

Apply the pruning recommendations the same way as turn 2. Repeat the
three-turn cycle until the spec settles.

This is the human's most important checkpoint. You are deciding what
the *next* implementer is allowed to assume, what they're allowed to
change, and what they cannot touch.

After each round of edits:

```bash
declare lint system.dx                       # must exit 0
declare diff HEAD:system.dx system.dx        # see what you changed semantically
git add system.dx && git commit -m "Architect: <describe the semantic change>"
```

You are done with this phase when:

- `assumptions:` contains only entries you *consciously* want to leave
  for the next implementer to handle.
- Every `invariants:` entry survives the pruning pass.
- Every `invariants:` entry is testable as a black box (the Judge
  needs to be able to verify it).
- `contracts:` covers the load-bearing behaviors. Five well-chosen
  contracts beat fifty redundant ones.

**Don't skip this phase.** A v0 spec straight from the archaeologist
is biased toward the original implementation's idiosyncrasies. The
architect pass is what makes the spec language-agnostic.

## 4. Implementer phase: synthesize the new code

Load the [`implementer`](../../skills/implementer/SKILL.md) skill,
open a fresh session if you can (so the agent has no memory of the
source code from earlier phases), and prompt:

> "Read only `system.dx`. Do **not** read anything under
> `impl_<source_lang>/`. Generate a complete implementation in
> `<target_language>` under `impl_<target_lang>/` that satisfies
> every entry in `invariants:` and every contract in `contracts:`.
> Use the language's native idioms; you do not need to mimic the
> original's structure. When the spec is ambiguous, append an
> `assumptions:` entry to `system.dx` *before* writing the code that
> makes the assumption."

The "do not read the source" instruction is doing real work here.
The whole point of the port is to prove the spec is sufficient. If
the implementer peeks, the new code inherits the original's
biases — and you've reduced the journey to a translation pass.

**Operational note (current gap):** No tool today *enforces* the
no-peeking rule. It is honor-system, mediated by the prompt and by
keeping `impl_<source_lang>/` out of the implementer session's open
files. A future `declare` may grow a sandboxing primitive
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
export TODO_FILE=/tmp/declare-port-scratch.json

# If it doesn't, override the default location at the OS level
# (containers, fakeroot, a chroot, or just `cd` into a clean dir for
# both implementations).
```

Treat this as part of the workspace setup in step 1 if your spec has
an `iface_*` invariant for an env-var override or a similar
configuration knob — apply it to both implementations during testing.

After the implementer finishes:

```bash
# Build the new implementation in its native toolchain.
cd impl_<target_lang> && <build command>

# Re-lint the spec — the implementer may have appended assumptions.
declare lint system.dx
declare diff HEAD:system.dx system.dx

git add . && git commit -m "Implementer: generate impl_<target_lang> from system.dx"
```

If the implementer logged new assumptions, you have two options:

- **Cheap and fast:** accept them and move on; the architect will
  ratify them in a follow-up pass.
- **Strict:** loop back to step 3, ratify each new assumption, and
  re-run the implementer. Better for high-stakes ports.

## 5. Judge phase: verify against the contracts

Load the [`judge`](../../skills/judge/SKILL.md) skill and prompt:

> "Walk every entry in `contracts:` against the new implementation
> under `impl_<target_lang>/`. For each contract: set up the
> `given`, trigger the `when`, observe the outcome, compare to the
> `then`. Report PASS/FAIL for each. For FAILs, classify as
> implementation bug, spec gap, or intent mismatch per the judge
> skill."

**Operational note (current gap):** there is no `declare verify`
command in v0.1.0 (deferred to v0.2 per
[SPEC §4](../../SPEC.md#4-verification-model)). The judge **is** the
contract executor today: an agent walks each contract by hand or via
its tool-use. This works fine for a handful of contracts; it does
not scale to dozens. The biggest priority gap in this journey is
mechanizing this step. See *Known gaps* below.

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

- `system.dx` — a language-agnostic, version-controlled spec for the
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
to `system.dx` first; the implementer then catches both
implementations up.

## Known gaps in this journey (priority TODOs)

The following are real, blocking-or-painful gaps in v0.1.0. Each one
is a candidate priority TODO for a future `declare` revision. They
are listed in roughly the order they bite an end-user trying to
follow this journey today.

### Gap 1 — No `declare verify` (high priority)

**Where it bites:** step 5 (Judge phase).

**Symptom:** you have 30 contracts, and the judge has to walk each
one by prose. By the time you've executed contract 30 you've forgotten
the setup for contract 1, and the audit trail is buried in chat
history.

**What's needed:** a `declare verify <system>.dx --impl <command>`
command that:

1. Parses `contracts:` into a structured execution plan.
2. For each contract, sets up `given` (via a small embedded grammar:
   env vars, files, args), runs `when` (executes `<command>` with the
   right args), evaluates `then` (matches stdout/stderr/exit code/file
   state).
3. Emits a deterministic pass/fail summary plus per-contract
   diagnostics.

This requires designing a contract grammar that's expressive enough
for real-world preconditions but constrained enough to stay
language-agnostic. SPEC §4 explicitly defers this to v0.2.

### Gap 2 — No mechanism to enforce "implementer must not read the source" (medium priority)

**Where it bites:** step 4 (Implementer phase).

**Symptom:** the no-peeking rule is honor-system. A diligent
agent honors it; a less diligent one quietly inherits the original's
quirks and the journey reduces to a translation pass.

**What's needed:** at minimum, a documented convention (e.g., a
`.declare-implementer-allowlist` file the agent runtime respects).
At maximum, a sandboxing primitive — though that crosses into the
agent-runtime layer, which is explicitly out of scope for the
`declare` binary itself.

## Working example

[`examples/weather_cli/`](../../examples/weather_cli/) in this repo
is a fully-worked instance of this journey:

- `impl_cpp/weather_cli.cc` — the source-language artifact (the
  archaeologist's input).
- `system.dx` — the spec extracted by the archaeologist and ratified
  by the architect.
- `impl_python/weather_cli.py` — the target-language artifact
  generated by the implementer from `system.dx` alone.

The README in that directory walks through which artifact corresponds
to which phase of this journey. Both implementations satisfy the
four directly-runnable contracts manually; the fifth
(`caches_repeat_queries`) needs `declare verify` to be checked
mechanically — a perfect illustration of Gap 1 above.

## Related reading

- [`AGENTS.md`](../../AGENTS.md) — the universal rules every agent
  follows in this repo, including the verification loop and the
  post-merge ritual.
- [`SPEC.md`](../../SPEC.md) — the normative `.dx` language reference.
- [`skills/declare-orchestrator/SKILL.md`](../../skills/declare-orchestrator/SKILL.md)
  — the meta-routing skill an agent loads on entering a
  `declare`-managed repo.
- The four role-skills referenced throughout this journey:
  [archaeologist](../../skills/archaeologist/SKILL.md),
  [architect](../../skills/architect/SKILL.md),
  [implementer](../../skills/implementer/SKILL.md),
  [judge](../../skills/judge/SKILL.md).
