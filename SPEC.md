# Specification: The `.dx` Language

This document defines `.dx`, a declarative specification language
designed to hold the *idea* of a software system in a form that
humans can review, version, and argue about — and that AI agents can
consume, validate, and produce imperative code from.

The document is organized in three parts, in deliberate order of
permanence:

- **Part I — Concepts.** The philosophical position behind `.dx`,
  the operating principles that follow from it, the conceptual
  definitions of each block, and the workflow within which `.dx`
  files are authored, evolved, and verified. This part is universal:
  it would be true even if no toolchain or serialization existed.

- **Part II — Serialization (v0.1.0).** The concrete YAML 1.2 subset
  used to write `.dx` files in this version of the spec. The
  physical-format rules, the structural constraints, the schema for
  each block, the reserved field set, and the versioning policy.
  This part is replaceable in a future major revision without
  changing the concepts.

- **Part III — Reference Implementation.** Pointers to the toolchain,
  skills, journeys, and worked examples that ship in this repository.
  This part is one of many possible instantiations: it exists to
  support readers in working with `.dx` files today, but the
  language is independent of any particular implementation.

A reader who wants only the *language* should read Part I and Part II.
A reader who wants only the *concepts* — to evaluate, criticize, or
re-implement them — can read Part I alone. A reader who wants to use
`.dx` today can read Part III after Parts I and II to find the tools.

# Part I — Concepts

## 1. Philosophy

A program is two things at once: an artifact (the source code, in
some particular language) and an idea (what the system is supposed
to *be*). For most of computing history the two have been fused.
The idea exists only as it is encoded in the artifact; reading the
idea means reading the code. When the artifact changes, the idea
changes — and there is no separate place where the idea lives that
can be checked, versioned, reviewed, or argued about.

The intellectual position behind `.dx` is that this fusion is now
optional. In a world where AI writes the imperative artifact, the
*idea* of the system is the load-bearing thing humans should attend
to, and the artifact is a derived witness — one of many possible
implementations, all equally valid if they satisfy the idea.

A `.dx` file is meant to be that idea, written down. It does not
specify *how* the system computes anything; it specifies *what is
true* about the system's observable behavior. The implementation is
a witness that those truths hold. Two witnesses, in different
languages, both honest, are equivalent.

This is an old idea given new traction. **Formal-methods languages**
(Z, VDM, TLA+, B, Alloy) have made this same separation since the
1970s — designed for human formal proof, never widely adopted because
authoring them is expensive and verifying them requires expert
reviewers. **Knowledge-representation languages** (KIF, OWL, Cyc)
attempted a similar move for AI consumers — they failed for the
inverse reason: humans couldn't agree on shared semantics, and
pre-LLM machines couldn't reason fast enough to make the cost
worthwhile. **Design-by-contract** (Eiffel, JML, Code Contracts)
embedded the idea inside imperative languages as annotations — it
gained adherents but never became the primary artifact.
**Denotational semantics** (Strachey, Scott) provided the
philosophical foundation — meaning before mechanism — but kept the
meaning in the heads of theorists rather than version-control
systems.

Each of these traditions had the right insight and the wrong era.
They lacked a *consumer* that could draft, validate, and act on a
declarative specification at scale. The arrival of LLM-mediated
coding workflows changes both sides of that equation: drafting
becomes a conversation with an agent, consuming becomes a tool-use
loop, and the cost calculus inverts.

`.dx` is the minimum-viable declarative intermediate language for
this new operating context. It is not a successor to TLA+ or Z or
OWL; it inherits from all of them. What it adds is a *delivery
mechanism* — the agent that consumes the spec and produces (or
modifies, or verifies) the implementation — that none of the prior
traditions could assume. The spec is the source of truth; the agent
is the engine; the imperative code is downstream of both.

A few honest claims this position commits us to:

1. **The `.dx` file is primary.** When the spec and the code
   disagree, the spec wins. If a constraint is impossible to
   satisfy, the spec changes — not the code.
2. **Implementations are plural and replaceable.** A `.dx` should
   never name a language, library, or framework. If two
   implementations in two languages both satisfy every contract,
   both are correct, and the question of which is "real" is
   malformed.
3. **Heuristic leaps are first-class artifacts.** When an agent
   must guess, the guess is recorded in `assumptions:` *before*
   the code is written. Silent invention is the failure mode the
   entire system is designed to prevent.
4. **Verification is observational.** A contract describes what an
   outside observer would see, not what the code does internally.
   This is the discipline that lets implementations diverge
   stylistically while remaining equivalent semantically.

What we are *not* claiming:

- That `.dx` files are formal proofs. They are not — `.dx` does not
  imply a model checker or a theorem prover. Verification is
  performed by an agent walking each contract; future revisions of
  the language may add expressive forms that admit mechanical
  verification for the contracts that support it.
- That the spec language is final. v0.1.0 is deliberately minimal;
  whole categories of expression (temporal logic, refinement
  relations, cross-system contracts) are absent and will land or be
  rejected as the workflow's needs become clearer.
- That AI-mediated specification will replace human design. The
  opposite: it makes human design *more* central by giving humans a
  place to do design that isn't immediately swamped by syntax
  decisions.

This positioning is the language's intellectual moat. Everything
that follows in Parts I and II operationalizes it; the philosophy is
what makes the operational choices coherent.

## 2. Operating Principles

The philosophy in §1 is realized through three operating principles
that govern every concrete choice in the language and the workflow:

- **Decoupling.** The `.dx` file defines the *what* and the
  *constraints*. The generated imperative code defines the *how*.
  Neither side is permitted to invade the other.
- **Orthogonality.** The specification must never leak imperative
  logic or dictate internal architecture (e.g., unit tests, library
  choice, threading model). It focuses purely on observable system
  state and the constraints that bind it.
- **Formalized hallucination.** Agents are required to explicitly
  declare heuristic leaps in an `assumptions:` block, turning silent
  hallucinations into auditable, promotable workflow state.

## 3. The Six Blocks

A `.dx` file is organized into six top-level blocks. Each block
exists for a specific philosophical reason; this section defines
each conceptually. The YAML shape of each block is in Part II §9;
this part is concerned only with what each block *means*.

### 3a. `system`

A unique identifier naming the declaration. Acts as the namespace
for this `.dx` file; a multi-spec project distinguishes its specs
by this name. Required.

### 3b. `intent`

The high-level semantic purpose of the system. Operationalizes the
"the `.dx` file is the *idea* of the system, written down" position
in §1: a fresh implementer reading only `intent` should understand
what the system is *for*, even if they cannot yet build it.

The block has two parts: a single `primary` objective (what the
system exists to do, in one sentence) and an optional list of
`secondary` goals (supporting objectives or non-functional concerns).
Required.

### 3c. `invariants`

Non-negotiable observable constraints that the implementation must
satisfy. Operationalizes positions 1 (the `.dx` file is primary)
and 4 (verification is observational) in §1: each invariant is a
proposition about the system's externally-visible behavior that all
valid implementations must honor.

Invariants describe *what is true*, not *how to compute it*. A
well-formed invariant never names a language, library, framework,
or internal data structure — those are implementation choices, not
constraints. Required (the block must exist; it may be empty if
the system genuinely has no invariants beyond `intent`).

### 3d. `assumptions`

Heuristic choices an agent made because `intent` and `invariants`
did not uniquely determine the answer. Operationalizes position 3
(heuristic leaps as first-class artifacts) in §1: the entire
purpose of the block is to convert silent invention into auditable,
promotable workflow state.

Each entry is a *what was decided* paired with a *why it was the
most defensible choice given the ambiguity*. The architect later
promotes a ratified assumption to `invariants` (we are committing
to it), demotes one to `unconstrained` (we explicitly don't care),
or rejects it (rewrites the spec so the assumption becomes
unnecessary).

The block is required. An empty `assumptions:` is meaningful: it
asserts that the agent made no unrecorded heuristic choices. This
is distinct from omitting the block, which would be a structural
error.

### 3e. `contracts`

Black-box verification rules in given/when/then form.
Operationalizes position 4 (verification is observational) in §1:
a contract is a recipe an outside observer can run to confirm an
invariant holds.

Every clause must reference observable state — stdout, stderr,
exit code, file system state, HTTP response, log output, and so
on. Never internal program state. A contract that cannot be
expressed in observable terms is a signal that the underlying
invariant is not testable as a black box and may need rephrasing.

Optional: a `.dx` file with no contracts is well-formed but loses
the verification story that makes the spec checkable.

### 3f. `unconstrained`

Explicitly declared degrees of freedom. Operationalizes position 2
(implementations are plural and replaceable) in §1: each entry says
"this aspect of the system is *not* constrained by the spec; the
implementer may choose freely."

Without this block, every unspecified aspect is ambiguously either
an oversight or an intentional non-constraint. This block
disambiguates. Use it aggressively: over-specification is a defect.
If the spec did not intend to constrain something, the choice
belongs either in this block (named explicitly) or absent
altogether (left to the implementer's discretion).

Optional.

## 4. The Multi-Agent Workflow

`.dx` is designed to be operated by a workflow of specialized roles
coordinated through the file itself. The roles are conceptual; any
specific implementation may collapse multiple roles into one agent
or split one role across many. What matters is the pattern.

1. **The Archaeologist.** Distills existing imperative code into
   semantic intent and observable invariants, producing a base
   `.dx` file. Operates only when the system already exists in code
   form; greenfield projects skip this role.
2. **The Architect.** Owns the `.dx` file. Refines `intent`,
   adds or prunes `invariants`, promotes ratified `assumptions`,
   demotes overspecifications to `unconstrained`. The Architect is
   the only role permitted to modify `intent`, `invariants`,
   `contracts`, and `unconstrained`.
3. **The Implementer.** Reads the `.dx` file (and only the `.dx`
   file) and produces imperative code that satisfies every
   invariant and contract. The Implementer is the only role
   permitted to *append* to `assumptions:` during code generation,
   and is forbidden from modifying any other block.
4. **The Judge.** Executes the implementation against the
   `contracts` block via black-box testing. Classifies any failure
   as either an implementation bug, a spec gap, or an intent
   mismatch, and routes it to the appropriate role for correction.

The roles share a strict, machine-checkable boundary on which
blocks each may write. Only the Architect may modify the spec-
defining blocks; the Implementer may only append to `assumptions:`;
the Judge writes nothing. This separation is what makes the
workflow auditable: any change to the `.dx` file can be attributed
to a specific role acting under specific authority.

When work transitions between roles, the transition is announced
explicitly. The conventional form is a single line of the form:

```
HANDOFF: <from-role> → <to-role>: <one-sentence reason>
```

The handoff line is the workflow's audit trail. Together with the
`.dx` file's git history, it makes every architectural decision
traceable.

## 5. Verification

Verification of an implementation against the `.dx` file is the
Judge's responsibility. The Judge interprets each contract's
`given` / `when` / `then` clauses, sets up the precondition,
triggers the action, and observes the outcome.

Three classifications cover every failure:

- **Implementation bug.** The code is wrong; the spec is right.
  The contract's expected outcome did not occur, and the contract
  is unambiguous, and no other invariant or contract contradicts
  it. Route to the Implementer for correction.
- **Spec gap.** The spec is wrong; the code is at most
  accidentally right. The contract is ambiguous, contradicts an
  invariant, or under-specifies the situation. Route to the
  Architect to tighten the spec.
- **Intent mismatch.** The contract conflicts with `intent` or
  with another invariant. The spec contradicts itself. Route to
  the Architect to reconcile.

When in doubt between *implementation bug* and *spec gap*, default
to *spec gap*. The cost of an incorrect *spec gap* call is one
extra Architect/Implementer cycle; the cost of an incorrect
*implementation bug* call is the Implementer rewriting working
code to satisfy a broken spec.

`.dx` does not require that contracts be machine-executable. A
contract written in prose is a contract; the Judge interprets it.
Future revisions of the language may add expressive forms that
admit mechanical verification for the contracts that support it,
but the language does not assume any particular execution model.

## 6. Spec Evolution

`.dx` files are version-controlled like source code. The language
does not define a structural merge algorithm; concurrent edits
resolve through whatever VCS the project uses.

After any merge, the Architect must:

1. Validate the merged file against this spec (Part II §8 and §9
   define what "valid" means).
2. Compute the *semantic* delta between the merge base and the
   merge result. A clean text-merge can hide a semantic conflict —
   for example, one branch demoting an invariant to `unconstrained`
   while the other branch tightens it. The semantic delta surfaces
   such conflicts as first-class operations against the schema.
3. Reconcile any semantic conflict in the spec, not in the
   implementation. Per position 1 in §1, the `.dx` file leads.

How the semantic delta is computed is an implementation concern
(see Part III for the reference toolchain's `diff` command); what
matters at the language level is that the reconciliation happens
in the spec.

Future revisions of the language may introduce a CRDT-style
structural merge that operates over the AST directly and surfaces
semantic conflicts as first-class operations. The current spec
does not require it.

# Part II — Serialization (v0.1.0)

This part defines the concrete YAML 1.2 subset used to write `.dx`
files in v0.1.0. The conceptual definitions of each block are in
Part I §3; this part is concerned only with physical format and
schema. References from this part back to Part I are explicit
where they matter.

## 7. Physical Format

A `.dx` file MUST be valid YAML 1.2 (subject to the structural
constraints in §8). The canonical file extension is `.dx`.

YAML was chosen as the substrate after considering JSON, TOML,
HCL, and a custom DSL. The decision rests on four properties that
matter specifically for the LLM-mediated authoring context the
language was designed for:

- **Universal editor support.** Every modern editor highlights
  YAML out of the box. No plugin, no language-server install, no
  setup cost for a human reviewer of any background.
- **Multi-line ergonomics.** The literal block scalar (`|`)
  preserves human-authored bytes line-by-line, which matters when
  a contract's `then:` clause references observable output
  verbatim. JSON has no native multi-line story; TOML's is
  awkward; HCL's is fine but locks adoption to the HashiCorp
  ecosystem.
- **Comment support.** YAML allows `#` comments. This is essential
  for human authoring and review. JSON's lack of comments alone
  rules it out for a spec language meant to be read by both humans
  and machines.
- **Deterministic AST.** YAML 1.2 is well-specified and produces
  stable parse trees across implementations *when the strict-
  subset rules in §8 are applied*. Without those rules YAML is
  famously unpredictable; the §8 constraints exist precisely to
  recover the determinism that the broader YAML spec sacrifices.

A custom DSL was rejected because tree-sitter / syntax-highlighter
investment is a real cost, and no candidate DSL we considered
offered enough advantage over a strict YAML subset to justify it.

A future major revision of the language may select a different
serialization (or admit multiple serializations of the same
underlying schema). Such a change does not affect Part I.

## 8. Structural Constraints

To maintain a deterministic Abstract Syntax Tree (AST) and prevent
semantic drift during agent processing, the following restrictions
apply to every `.dx` file. All MUST and MUST NOT clauses are
enforceable structurally.

- **No Anchors / Aliases.** A `.dx` file MUST NOT use YAML anchors
  (`&name`) or aliases (`*name`). They introduce hidden state that
  breaks an agent's local reasoning over the document.
- **No Custom Tags.** A `.dx` file MUST NOT use explicit YAML tags
  outside the implicit core schema (`!!str`, `!!int`, `!!float`,
  `!!bool`, `!!null`, `!!seq`, `!!map`, `!!timestamp`). `!!binary`,
  `!!set`, and any user-defined `!foo` tags are rejected.
- **Literal Scalars Only.** Multi-line strings MUST use the literal
  block scalar (`|`). The folded scalar (`>`) is rejected because
  it collapses newlines into spaces in ways that vary subtly
  across YAML libraries and LLM tokenizers — the resulting decoded
  value is no longer reliably the bytes the human wrote.
- **Scalar Leaves.** Map values inside `invariants`, `assumptions`,
  and `unconstrained` MUST be scalar strings, not nested mappings
  or sequences. (See §10 for the v0.2 reserved field set, which
  anticipates relaxing this rule to allow a structured leaf shape.)
- **Root Key Ordering.** A `.dx` file SHOULD list its top-level
  keys in this order: `system`, `intent`, `invariants`,
  `assumptions`, `contracts`, `unconstrained`. A file that violates
  the SHOULD is structurally valid but is not in canonical form.
  (See Part III for the reference toolchain's `fmt` command, which
  enforces canonical form automatically.)

## 9. Schema

This section defines the YAML shape of each block. The conceptual
purpose of each block is in Part I §3; only the concrete schema
appears here.

### `system` (Required)

A string scalar in slug format (conventionally kebab-case, no
leading digit).

```yaml
system: hello-world
```

### `intent` (Required)

A mapping with two members:

- `primary` — a string scalar. The core objective of the system
  in one sentence. Required.
- `secondary` — a sequence of string scalars. Supporting
  objectives or non-functional goals. Optional. Order is
  significant and is preserved by canonical formatting.

```yaml
intent:
  primary: Greet a user by name on standard output.
  secondary:
    - Be friendly.
    - Exit cleanly.
```

### `invariants` (Required)

A mapping from string keys (invariant identifiers) to string
scalars (invariant bodies). The block must be present even when
empty; an empty block is written as `{}`.

Keys SHOULD carry a category prefix. Conventional prefixes
include `iface_`, `perf_`, `sec_`, `obs_`, `data_`, and `ux_`;
projects may define additional prefixes used consistently within
a single file.

The body is a string scalar describing the constraint in
black-box terms (see Part I §3c for the conceptual rule).

```yaml
invariants:
  iface_stdout: Writes a single UTF-8 line to stdout terminated by `\n`.
  perf_startup_ms: Cold-start latency must remain under 50ms on commodity hardware.
```

### `assumptions` (Required)

A mapping from string keys (assumption identifiers) to string
scalars (assumption bodies). The block must be present even when
empty; an empty block is written as `{}`.

The empty-block state is meaningful per Part I §3d: it asserts
"the agent made no unrecorded heuristic choices," distinct from
"the agent forgot to record any."

```yaml
assumptions:
  greeting.format: |
    The greeting is "Hello, <name>!" — the spec does not pin
    punctuation or word choice; this matches the canonical
    POSIX-tutorial form.
```

### `contracts` (Optional)

A mapping from string keys (contract names) to contract objects.
Each contract object is a mapping with three string-scalar fields:

- `given` — the precondition under which the contract applies.
- `when` — the triggering action or event.
- `then` — the observable outcome that must hold.

All three fields are conventionally present. A contract that
cannot express any one of them in observable terms is a signal
that the underlying invariant is not testable as a black box and
may need rephrasing (see Part I §3e).

```yaml
contracts:
  greets_named_user:
    given: The argument vector contains exactly one non-empty name.
    when: The binary is invoked.
    then: stdout contains "Hello, <name>!\n" and the exit code is 0.
```

### `unconstrained` (Optional)

A mapping from string keys (category names) to string scalars
(descriptions of the freedom granted). Both keys and values are
strings.

Common categories include `language`, `internal_data_structures`,
`cache_format`, `output_phrasing`, and `concurrency_model` —
anything the spec wants to leave open. The set is open-ended;
projects may invent categories as needed.

```yaml
unconstrained:
  language: Any language with a stable POSIX runtime is acceptable.
```

## 10. Reserved Field Names (Future Compatibility)

The following field names are **reserved** within `invariants:`,
`assumptions:`, `contracts:`, and `unconstrained:` map values.
v0.1.0 does not require them; a future revision may attach
normative semantics to each. Tooling MUST NOT use them for
unrelated purposes.

- `rule` — the constraint or assertion text (the body of a
  v0.1.0 leaf).
- `reason` — free-form prose explaining *why* the entry exists.
- `author` — the agent or human responsible for the most recent
  mutation (e.g., `agent-architect@cloudcode/2026-05-12`).
- `since` — the spec version or change identifier in which the
  entry first appeared.

In v0.1.0, leaves under `invariants:` / `assumptions:` /
`unconstrained:` are scalar strings (per §8). The reserved-field
set anticipates a v0.2 transition to a structured shape:

```yaml
# Forward-compatible v0.2 sketch -- NOT valid v0.1.0:
invariants:
  perf_cache_ttl:
    rule: Cache TTL must be strictly 600 seconds.
    reason: Upstream API documentation forbids polling faster than 10 minutes.
    author: agent-architect@cloudcode
    since: v0.1.0
```

Reserving the names now lets a future revision adopt the
structured form without colliding with field names already in
use.

## 11. Versioning

This part of the document describes v0.1.0 of the `.dx`
serialization. Future revisions will be released as
`v0.MAJOR.MINOR`:

- **Patch** (`v0.1.x`): clarifications, additional reserved names,
  additional structural checks that reject already-questionable
  input. No new required fields.
- **Minor** (`v0.x.0`): new optional blocks, structured forms of
  existing leaves (gated by the reserved-field discipline in §10),
  new conventions.
- **Major** (`v1.0.0`): commitment to long-term backward
  compatibility. May also be the point at which a different
  serialization is selected, while keeping Part I unchanged.

v0.1.0 does not include a top-level spec-version declaration. A
future revision will introduce one (likely a top-level `dx_spec:`
key); until then, `.dx` files have no in-band version marker and
are assumed to target the current released spec.

The conceptual content of Part I is independent of this
versioning scheme. A change to Part II's serialization may happen
without changing Part I; a change to Part I's concepts is a more
significant event and would coincide with a major release.

# Part III — Reference Implementation

The toolchain, skills, journeys, and worked examples that ship in
this repository are one possible instantiation of the language
defined in Parts I and II. They are reference material, not part
of the language definition. A different team could build a
completely different toolchain — in a different programming
language, with a different command structure, targeting a
different agent runtime — and `.dx` files written for one would
work with the other, provided both implementations conform to
this spec.

What this repository ships:

- A deterministic CLI (`declare`) implementing structural
  validation, canonical formatting, semantic diffing, AST export,
  and contract enumeration. The CLI contains no LLM — by design,
  per Part I §1: the binary is the referee, not a player.
  See [`README.md`](README.md) for the command surface and
  [`ROADMAP.md`](ROADMAP.md) for what is shipped, what is
  deferred, and why.
- Seven portable agent skills under [`skills/`](skills/) that
  teach a coding agent (Claude Code, Gemini CLI, Cursor, or any
  agent that consumes Markdown skills) how to operate as
  Archaeologist, Architect, Implementer, or Judge. See
  [`skills/README.md`](skills/README.md).
- End-to-end user journeys under [`docs/journeys/`](docs/journeys/)
  walking through real tasks (porting a program to another
  language, greenfield development, adding a feature, supporting
  multiple language implementations of the same spec).
- Worked examples under [`examples/`](examples/) — minimal
  fixtures plus a fully-worked weather CLI demonstrating the
  Archaeologist → Architect → Implementer → Judge loop end to end.

Behavioral expectations for any agent contributing to this
repository — independent of which toolchain or skills are loaded —
are in [`AGENTS.md`](AGENTS.md). Those expectations follow from
Part I of this document.
