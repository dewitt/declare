# The dx Specification Language

> **Status:** Experimental. This document specifies an experimental
> language and is not on any standards track. It is published to
> invite review, criticism, and revision. The language and this
> document may change incompatibly in future revisions.

## Abstract

The dx specification language defines a serialization-independent
form for declaring the intent and constraints of a software system,
intended for use by AI agents that consume the specification and
produce conforming imperative implementations. A `.dx` file declares
what a system is required to do in terms an outside observer can
verify, and explicitly enumerates the heuristic choices an agent
made on the human author's behalf so those choices can be reviewed
and ratified. The language is descriptive of behavior, not
prescriptive of implementation: a single specification may admit
many valid implementations across many programming languages.

This document defines the conceptual model in Section 3 and a
concrete YAML 1.2 serialization in Section 4. Other serializations
are permitted and may be specified in future revisions without
changing the conceptual model.

## 1. Introduction

A program is two things at once: an artifact written in some
programming language, and the idea of what that program is for. In
conventional software development the two are fused; the idea exists
only as encoded in the artifact. In a workflow where an AI agent
produces the artifact from a human-supplied prompt, the absence of
a separate, durable record of the idea becomes a defect. The agent
fills gaps in the prompt with silent heuristic choices; subsequent
edits drift away from the original intent; and the human reviewer
has no place to look that is not the imperative code.

The dx language addresses this by giving the idea its own artifact:
a `.dx` file. The file is authored, read, and modified by humans
and agents alike — typically jointly, with the human supplying
intent and the agent surfacing the constraints and assumptions
that intent implies — and version-controlled as the source of
truth for what the system is required to do. The imperative code
becomes a witness that the requirements hold; multiple equally
valid implementations may exist for the same `.dx` file.

### 1.1. Goals

The dx language aims to:

1. Provide a durable, version-controllable record of a system's
   intent and constraints, separate from any implementation.
2. Make the choices an agent makes on the human's behalf explicit
   and reviewable, by requiring those choices to be recorded as
   first-class artifacts before the corresponding code is written.
3. Permit multiple implementations in different programming
   languages from a single specification, with conformance
   verifiable from outside the implementation.
4. Be small enough to read end-to-end and use today, with a
   trajectory toward greater expressive power in future revisions.
5. Be portable across agent runtimes by avoiding any dependency
   on the particulars of a specific tool or model.

### 1.2. Non-Goals

The dx language does not aim to:

1. Provide formal proof that an implementation satisfies a
   specification. The conformance check defined in Section 3.8 is
   observational, not deductive.
2. Replace human design judgment. The language gives humans a
   place to record decisions; it does not make decisions.
3. Be the only or final form of declarative specification for
   AI-mediated software development. Future work may supersede or
   subsume parts of this language.

### 1.3. Historical Antecedents

The separation of specification from implementation has a long
history. Formal specification languages including Z [Z], VDM [VDM],
TLA+ [TLA], B [B], and Alloy [Alloy] address the same separation
for the purpose of human formal proof. Knowledge-representation
languages including KIF [KIF] and OWL [OWL] address it for
machine-consumable knowledge bases. Design-by-contract systems
including Eiffel [Eiffel] and JML [JML] embed declarative
constraints inside imperative languages as annotations. The
denotational-semantics tradition [Strachey] [Scott] provides the
mathematical framework for the underlying separation between a
program's meaning and its mechanism.

The dx language inherits the conceptual move common to all of these
prior efforts. It differs in addressing AI-mediated software
development specifically: the consumer of a `.dx` file is normally
an agent, the producer of the implementation is normally an agent,
and the mechanism by which heuristic choices are made explicit is
designed for that workflow. See Section 7 for citations.

### 1.4. Document Conventions

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT",
"SHOULD", "SHOULD NOT", "RECOMMENDED", "MAY", and "OPTIONAL" in
this document are to be interpreted as described in BCP 14
[RFC2119] [RFC8174] when, and only when, they appear in all
capitals, as shown here.

## 2. Terminology

This section defines terms used throughout the document. Terms are
listed in dependency order: later definitions may rely on earlier
ones.

**Implementation.** A program, in some programming language,
intended to satisfy a declaration. A single declaration may have
zero, one, or many implementations.

**Declaration.** The contents of a `.dx` file. Comprises a system
identifier, an intent, a set of invariants, a set of assumptions,
optionally a set of contracts, and optionally a set of
unconstrained degrees of freedom. The declaration is the source of
truth for what the implementation is required to do.

**Observable.** A property of a running implementation that can be
determined from outside the implementation: standard input, output,
and error streams; exit codes; file system state; network traffic;
and similar externally-visible effects. Internal program state,
data structures, and call graphs are not observable in this sense.

**System identifier.** A short string that names a declaration.
Acts as the namespace for the declaration when multiple
declarations coexist in a single project.

**Intent.** A short, human-readable statement of what the system
exists to do. Comprises a primary objective and an optional list
of secondary objectives.

**Invariant.** A non-negotiable observable property the
implementation MUST maintain. Identified by a category-prefixed
slug. Each invariant constrains every valid implementation.

**Assumption.** A heuristic choice an agent made because the intent
and invariants did not uniquely determine the answer. Identified
by a slug. Each assumption pairs the choice that was made with the
reason it was the most defensible choice given the ambiguity.

**Contract.** A black-box verification rule expressed in
given/when/then form. Each contract describes a precondition, a
triggering action, and an observable outcome that MUST hold.
Contracts are how invariants are checked.

**Unconstrained degree of freedom.** An aspect of the system the
declaration explicitly leaves to the discretion of whoever
produces the implementation. Identified by a category name.

**Agent.** A program, typically incorporating a large language
model, that consumes or produces declarations and implementations.
The dx language does not specify which agent is used or how it is
invoked, nor does it require that one be used at all; declarations
MAY be authored and implemented entirely by humans.

**Conformance.** An implementation conforms to a declaration if
and only if every contract in the declaration holds against that
implementation, as defined in Section 3.8.

## 3. Concepts

This section defines the conceptual model of the dx language
independent of any concrete serialization. Section 4 defines the
v0.1.0 YAML serialization of the model defined here.

### 3.1. Declarations

A declaration is the unit of specification in the dx language. A
declaration comprises six components, of which four are REQUIRED
and two are OPTIONAL:

| Component | Status |
|---|---|
| System identifier | REQUIRED |
| Intent | REQUIRED |
| Invariants | REQUIRED |
| Assumptions | REQUIRED |
| Contracts | OPTIONAL |
| Unconstrained degrees of freedom | OPTIONAL |

A REQUIRED component MUST be present, though some REQUIRED
components MAY be empty (see the per-component definitions in
Sections 3.2 through 3.7).

Implementations MUST treat the declaration as the source of truth.
When the behavior of an implementation conflicts with the
declaration, the implementation is at fault by definition; if a
constraint cannot be satisfied, the declaration MUST be modified
rather than the implementation made non-conforming.

### 3.2. The System Identifier

A declaration MUST carry a system identifier: a short string that
names the declaration and acts as its namespace within a project
that contains multiple declarations.

### 3.3. Intent

A declaration MUST carry an intent. The intent comprises:

- A REQUIRED primary objective: a short statement of what the
  system exists to do, sufficient that a fresh reader of the intent
  understands the system's purpose.
- An OPTIONAL ordered list of secondary objectives: supporting
  goals or non-functional concerns. Order is significant.

The intent describes purpose, not mechanism. An intent that names
a programming language, a library, a framework, or an internal
implementation strategy is malformed.

### 3.4. Invariants

A declaration MUST carry an invariants block. The block is a
collection of invariants, each identified by a unique category-
prefixed slug. An invariant is an observable property the
implementation MUST maintain.

Invariants describe what is true of the implementation as observed
from outside, not how the implementation is constructed. An
invariant MUST NOT name a programming language, a library, a
framework, or an internal data structure. The set of valid
implementations of a declaration is defined as the set of programs
for which every invariant holds.

The invariants block MAY be empty. An empty invariants block
asserts that the system has no constraints beyond its intent.

Conventional category prefixes include `iface_` (interface
behavior), `perf_` (performance), `sec_` (security), `obs_`
(observability), `data_` (data shape and persistence), and `ux_`
(user experience). Implementations MAY define additional prefixes
provided they are used consistently within a single declaration.

### 3.5. Assumptions

A declaration MUST carry an assumptions block. The block is a
collection of assumptions, each identified by a unique slug. An
assumption is a heuristic choice made because the intent and
invariants did not uniquely determine the answer.

Each assumption MUST record both the choice that was made and the
reason it was the most defensible choice given the ambiguity. An
assumption that records only the choice is malformed; the rationale
is not optional.

When any party (human or agent) producing or modifying an
implementation determines behavior that is not specified by the
intent or by an invariant, that party MUST record the
determination as an assumption before the corresponding code is
written. Silent determination is forbidden by this specification.

The assumptions block MAY be empty. An empty assumptions block
asserts that no unrecorded heuristic choices were made in producing
the implementation. This assertion is distinguishable from omitting
the block, which is a structural error.

Assumptions have a lifecycle. Over time, an assumption SHOULD be
reviewed and resolved into one of three outcomes: ratified as an
invariant (when the choice has been confirmed as a binding
constraint), demoted to an unconstrained degree of freedom (when
the choice is confirmed as not constraining any future
implementation), or rejected by modifying the declaration so that
the assumption becomes unnecessary. The dx language does not
specify who performs this review.

### 3.6. Contracts

A declaration MAY carry a contracts block. The block is a
collection of contracts, each identified by a name. A contract is
a black-box verification rule comprising three clauses:

- A `given` clause stating the precondition under which the
  contract applies.
- A `when` clause stating the triggering action or event.
- A `then` clause stating the observable outcome that MUST hold
  if the precondition was met and the trigger occurred.

Every clause of every contract MUST reference observable state.
Internal program state, intermediate computations, and
implementation details are out of scope for contracts. A contract
that cannot be expressed in observable terms indicates that the
underlying invariant is not testable as a black box and SHOULD be
rephrased.

A declaration with no contracts is well-formed but loses the
verification story that makes the declaration checkable.
Conformance (Section 3.8) cannot be determined for an
implementation against a declaration that contains no contracts.

### 3.7. Unconstrained Degrees of Freedom

A declaration MAY carry an unconstrained block. The block is a
collection of categories, each paired with a description, that
the declaration explicitly leaves to the discretion of whoever
produces the implementation.

Without an unconstrained block, every aspect of the system not
specified by an invariant is ambiguously either an oversight or
an intentional non-constraint. The unconstrained block
disambiguates: an entry asserts that the named aspect was
considered and intentionally left open.

Common categories include `language` (which programming language
to implement in), `internal_data_structures`, `cache_format`,
`output_phrasing`, and `concurrency_model`. The set is open-ended
and projects MAY define their own categories.

A declaration SHOULD use the unconstrained block aggressively.
Over-specification is a defect: an aspect that the declaration
does not intend to constrain MUST be either named in the
unconstrained block or absent altogether.

### 3.8. Conformance

An implementation conforms to a declaration if and only if every
contract in the declaration's contracts block holds against that
implementation. A contract holds when, given its precondition and
its triggering action, the observable outcome matches its `then`
clause.

Conformance is observational. Whether a contract holds is
determined by inspecting the implementation's externally-visible
behavior, not its internal state (per Section 3.6). This
specification does not require that contracts be machine-
executable; a contract written in prose is a contract. Who
performs the conformance check, and how, are operational concerns
beyond the scope of this specification.

### 3.9. Spec Evolution

Declarations are version-controlled. Two revisions of a
declaration admit a *semantic delta*: an enumeration of the
operations against the schema (additions, removals, mutations,
promotions, demotions, renames of invariants, assumptions,
contracts, and unconstrained entries) that distinguish them.
The semantic delta is well-defined regardless of how the two
revisions came to differ.

A semantic delta is distinct from a textual diff over the
serialization. Two revisions that produce no semantic delta are
equivalent under this specification; two revisions that produce
no textual diff but differ semantically are malformed (a
structural error in at least one of them).

When concurrent modifications to a declaration are reconciled,
the reconciliation MUST occur in the declaration itself, per
Section 3.1 (declarations are the source of truth). How the
semantic delta is computed and how reconciliation is performed
are operational concerns beyond the scope of this specification.

## 4. Serialization (v0.1.0)

This section defines a concrete YAML 1.2 serialization of the
conceptual model defined in Section 3. The serialization is
versioned independently of the conceptual model: a future
revision MAY define additional or alternative serializations
without changing Section 3.

### 4.1. Physical Format

A `.dx` file MUST be a valid YAML 1.2 [YAML] document, subject to
the structural constraints in Section 4.2. The canonical file
extension is `.dx`.

YAML was selected as the v0.1.0 serialization for four reasons:

- **Universal editor support.** Every modern editor highlights
  YAML out of the box, with no plugin required.
- **Multi-line ergonomics.** The literal block scalar (`|`)
  preserves authored bytes line by line, which is necessary when
  contract clauses reference observable output verbatim.
- **Comment support.** YAML permits `#` comments, which is
  essential for the human review that follows agent-authored
  drafts.
- **Deterministic AST.** YAML 1.2, when constrained as in Section
  4.2, produces stable parse trees across implementations.

JSON, TOML, HCL, and a custom DSL were considered and rejected:
JSON for absence of comments and weak multi-line support; TOML
for awkward multi-line strings; HCL for ecosystem coupling to a
single vendor; a custom DSL for tree-sitter and editor-tooling
costs not justified by the marginal gain over a constrained YAML
subset.

### 4.2. Structural Constraints

To preserve a deterministic Abstract Syntax Tree (AST) and to
prevent ambiguity that could affect agent processing, the
following constraints apply to every `.dx` file. The constraints
are normative and enforceable structurally.

- **Anchors and aliases.** A `.dx` file MUST NOT use YAML
  anchors (`&name`) or aliases (`*name`). They introduce hidden
  state that breaks local reasoning over the document.
- **Custom tags.** A `.dx` file MUST NOT use explicit YAML tags
  outside the implicit core schema (`!!str`, `!!int`, `!!float`,
  `!!bool`, `!!null`, `!!seq`, `!!map`, `!!timestamp`).
  Application-defined tags such as `!!binary`, `!!set`, or
  user-defined `!foo` are rejected.
- **Multi-line scalars.** Multi-line strings MUST use the literal
  block scalar (`|`). The folded block scalar (`>`) is rejected
  because folding behavior varies subtly across YAML libraries
  and across LLM tokenizers; the decoded value is no longer
  reliably the bytes that were authored.
- **Scalar leaves.** Map values inside `invariants`,
  `assumptions`, and `unconstrained` MUST be scalar strings, not
  nested mappings or sequences. (See Section 4.4 for the v0.2
  reserved field set, which anticipates relaxing this rule for
  certain reserved field names.)
- **Top-level key ordering.** A `.dx` file SHOULD list its
  top-level keys in the order: `system`, `intent`, `invariants`,
  `assumptions`, `contracts`, `unconstrained`. A file that
  violates the SHOULD is structurally valid but is not in
  canonical form.

### 4.3. Schema

This section defines the YAML shape of each component of a
declaration. The conceptual purpose of each component is in
Section 3; only the concrete schema is defined here. See Appendix
A for fully-worked examples.

#### 4.3.1. system

The value of `system` MUST be a string scalar in slug format
(conventionally kebab-case, with no leading digit).

#### 4.3.2. intent

The value of `intent` MUST be a mapping with the following
members:

- `primary`: REQUIRED. A string scalar.
- `secondary`: OPTIONAL. A sequence of string scalars. Order is
  significant and MUST be preserved by canonical formatting.

#### 4.3.3. invariants

The value of `invariants` MUST be a mapping from string keys
(invariant identifiers) to string scalars (invariant bodies).
The mapping MAY be empty; if empty, it MUST be encoded as `{}`.

Each key SHOULD carry a category prefix per Section 3.4.

#### 4.3.4. assumptions

The value of `assumptions` MUST be a mapping from string keys
(assumption identifiers) to string scalars (assumption bodies).
The mapping MAY be empty; if empty, it MUST be encoded as `{}`.

The empty-mapping form is semantically meaningful per Section
3.5: it asserts that no unrecorded heuristic choices were made.

#### 4.3.5. contracts

The value of `contracts`, if present, MUST be a mapping from
string keys (contract names) to contract objects. Each contract
object MUST be a mapping with three string-scalar fields:

- `given`: the precondition under which the contract applies.
- `when`: the triggering action or event.
- `then`: the observable outcome that MUST hold.

A contract object that omits any of the three fields is
malformed.

#### 4.3.6. unconstrained

The value of `unconstrained`, if present, MUST be a mapping from
string keys (category names) to string scalars (descriptions of
the freedom granted).

### 4.4. Reserved Field Names

The following field names are reserved within `invariants`,
`assumptions`, `contracts`, and `unconstrained` map values.
v0.1.0 does not require them; future revisions of this
serialization MAY attach normative semantics to each.
Implementations MUST NOT use these names for unrelated
purposes.

- `rule`: the constraint or assertion text (the body of a
  v0.1.0 leaf).
- `reason`: free-form prose explaining why the entry exists.
- `author`: the agent or human responsible for the most recent
  modification of the entry.
- `since`: the spec version or change identifier in which the
  entry first appeared.

In v0.1.0, leaves under `invariants`, `assumptions`, and
`unconstrained` are scalar strings per Section 4.2. The
reserved-field set anticipates a v0.2 transition to a structured
leaf shape; reserving the names now permits that transition
without colliding with field names already in use. Appendix A.5
shows a forward-compatible v0.2 sketch.

### 4.5. Versioning

This document specifies v0.1.0 of the dx serialization. Future
revisions will be released as `vMAJOR.MINOR.PATCH` per the
following rules:

- **Patch** (`v0.1.x`): clarifications, additional reserved
  names, additional structural checks that reject already-
  questionable input. No new required fields.
- **Minor** (`v0.x.0`): new optional blocks, structured forms
  of existing leaves (gated by the reserved-field discipline in
  Section 4.4), additional conventions.
- **Major** (`v1.0.0` and later): commitment to long-term
  backward compatibility. A major revision MAY also be the point
  at which an alternative serialization is selected, with
  Section 3 unchanged.

v0.1.0 does not include an in-band serialization-version
declaration. A future revision is expected to introduce one
(likely a top-level `dx_spec` key); until then, `.dx` files
have no in-band version marker and are assumed to target the
current released version of this serialization.

The conceptual content of Section 3 is independent of this
versioning scheme. A change to Section 4 may occur without a
change to Section 3; a change to Section 3 is a more significant
event and would coincide with at least a minor release.

## 5. Security Considerations

A `.dx` file is consumed by an agent that produces or modifies
imperative code. An adversarial declaration could therefore
direct the agent to produce code with security vulnerabilities,
backdoors, or unauthorized network or filesystem access.
Mitigation is the responsibility of the agent runtime and the
human reviewer; this specification cannot defend against
adversarial declarations on its own.

Three observations on the threat model are worth recording:

- A declaration can specify only what the implementation MUST
  do, not what it MUST NOT do. The `unconstrained` block (Section
  3.7) declares aspects intentionally left open, but it is not a
  capability list. An agent reading a declaration MAY produce
  behavior that no invariant or contract addresses; whether that
  behavior is permitted is a question for the agent runtime, not
  for the specification.
- The conformance model defined in Section 3.8 is observational
  and prose-driven. Whoever performs the conformance check can be
  deceived by ambiguous prose; a check that does not exercise
  every contract on every implementation can miss regressions.
  Operators relying on dx-mediated workflows SHOULD treat
  conformance checking as a gating step, not as an advisory one.
- The reserved field set in Section 4.4 anticipates an `author`
  field. An `author` value is self-asserted; the specification
  does not define an authentication mechanism. Treat author
  values as advisory metadata, not as cryptographic provenance.

Future revisions of this specification MAY define explicit
mechanisms for capability listing, signed authorship, or
machine-verifiable contract execution. The present specification
addresses none of these.

## 6. References

[RFC2119] Bradner, S., "Key words for use in RFCs to Indicate
  Requirement Levels", BCP 14, RFC 2119, March 1997,
  <https://www.rfc-editor.org/info/rfc2119>.

[RFC8174] Leiba, B., "Ambiguity of Uppercase vs Lowercase in
  RFC 2119 Key Words", BCP 14, RFC 8174, May 2017,
  <https://www.rfc-editor.org/info/rfc8174>.

[YAML] Ben-Kiki, O., Evans, C., and I. döt Net, "YAML Ain't
  Markup Language (YAML™) Version 1.2", 3rd Edition, October
  2009, <https://yaml.org/spec/1.2.2/>.

[Z] Spivey, J. M., "The Z Notation: A Reference Manual",
  Prentice Hall, 1989.

[VDM] Jones, C. B., "Systematic Software Development Using
  VDM", 2nd Edition, Prentice Hall, 1990.

[TLA] Lamport, L., "Specifying Systems: The TLA+ Language and
  Tools for Hardware and Software Engineers", Addison-Wesley,
  2002.

[B] Abrial, J.-R., "The B-Book: Assigning Programs to
  Meanings", Cambridge University Press, 1996.

[Alloy] Jackson, D., "Software Abstractions: Logic, Language,
  and Analysis", Revised Edition, MIT Press, 2012.

[KIF] Genesereth, M. R. and R. E. Fikes, "Knowledge Interchange
  Format Version 3.0 Reference Manual", Stanford University
  Computer Science Department, 1992.

[OWL] W3C OWL Working Group, "OWL 2 Web Ontology Language
  Document Overview (Second Edition)", W3C Recommendation,
  December 2012, <https://www.w3.org/TR/owl2-overview/>.

[Eiffel] Meyer, B., "Object-Oriented Software Construction",
  2nd Edition, Prentice Hall, 1997.

[JML] Leavens, G. T., Baker, A. L., and C. Ruby, "Preliminary
  Design of JML: A Behavioral Interface Specification Language
  for Java", ACM SIGSOFT Software Engineering Notes 31(3),
  2006.

[Strachey] Strachey, C., "Fundamental Concepts in Programming
  Languages", lecture notes from the International Summer
  School in Computer Programming, Copenhagen, August 1967;
  reprinted in Higher-Order and Symbolic Computation 13,
  2000.

[Scott] Scott, D. S. and C. Strachey, "Toward a Mathematical
  Semantics for Computer Languages", Programming Research
  Group Technical Monograph PRG-6, Oxford University, 1971.

## Appendix A. Examples

This appendix provides fully-worked examples of the v0.1.0 YAML
serialization defined in Section 4. Examples are illustrative and
informative; they do not extend or override the normative content
of the preceding sections.

### A.1. A minimal declaration

The smallest well-formed `.dx` file declares a system identifier,
an intent with a primary objective, and the two REQUIRED but
possibly empty maps for invariants and assumptions.

```yaml
system: empty
intent:
  primary: A placeholder declaration with no constraints.
invariants: {}
assumptions: {}
```

### A.2. A complete declaration with all blocks

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
    The greeting is "Hello, <name>!" — the spec does not pin
    punctuation or word choice; this matches the canonical
    POSIX-tutorial form.
contracts:
  greets_named_user:
    given: The argument vector contains exactly one non-empty name.
    when: The binary is invoked.
    then: stdout contains "Hello, <name>!\n" and the exit code is 0.
unconstrained:
  language: Any language with a stable POSIX runtime is acceptable.
```

### A.3. Multi-line scalars

Single-line bodies use plain scalars. Multi-line bodies use the
literal block scalar (`|`); the folded block scalar (`>`) is
prohibited per Section 4.2.

```yaml
invariants:
  iface_simple: A single-line invariant body uses a plain scalar.
  iface_complex: |
    A multi-line invariant body uses the literal block scalar.
    Subsequent lines are preserved verbatim, line by line. Use
    this form when the constraint genuinely requires more than
    one sentence to express.
```

### A.4. The empty-block contract

An empty `assumptions` block is semantically meaningful per
Section 3.5. The two examples below are not equivalent:

```yaml
# Asserts: no unrecorded heuristic choices were made.
assumptions: {}
```

```yaml
# Malformed: the REQUIRED `assumptions` block is missing.
# (assumptions key omitted entirely)
```

### A.5. Forward-compatible v0.2 sketch (NOT valid in v0.1.0)

The reserved field set in Section 4.4 anticipates a v0.2
transition to a structured leaf shape. The following is not
valid v0.1.0; it is shown only to clarify the intended direction.

```yaml
invariants:
  perf_cache_ttl:
    rule: Cache TTL must be strictly 600 seconds.
    reason: Upstream API documentation forbids polling faster than 10 minutes.
    author: agent-architect@cloudcode
    since: v0.1.0
```
