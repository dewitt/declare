# The dx Specification Language

> **Status:** Experimental. This document specifies an experimental
> language and is not intended for production use. It is published to
> invite review, criticism, and revision. The language and this
> document may change incompatibly in future revisions.

## Abstract

The dx specification language defines a serialization-independent
form for declaring the intent and constraints of a software system,
intended for use by AI agents that consume the specification and
produce conforming imperative implementations. A dx declaration
declares what a system is required to do in terms an outside
observer can verify, and explicitly enumerates the heuristic
choices an agent made on the human author's behalf so those choices
can be reviewed and ratified. The language is descriptive of behavior, not
prescriptive of implementation: a single specification may admit
many valid implementations across many programming languages.

This document defines the conceptual model in Section 3 and a
concrete CommonMark serialization in Section 4. Other serializations
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
a `.md` file written in a small, strictly-defined subset of
CommonMark. The file is authored, read, and modified by humans
and agents alike — typically jointly, with the human supplying
intent and the agent surfacing the constraints and assumptions
that intent implies — and version-controlled as the source of
truth for what the system is required to do. The imperative code
becomes a witness that the requirements hold; multiple equally
valid implementations may exist for the same declaration.

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
history. Formal specification languages including Z, VDM, TLA+, B,
and Alloy address the same separation for the purpose of human
formal proof. Knowledge-representation languages including KIF and
OWL address it for machine-consumable knowledge bases.
Design-by-contract systems including Eiffel and JML embed
declarative constraints inside imperative languages as
annotations. The denotational-semantics tradition descending from
Strachey and Scott provides the mathematical framework for the
underlying separation between a program's meaning and its
mechanism. Section 6 cites these works.

The dx language inherits the conceptual move common to all of these
prior efforts. It differs in addressing AI-mediated software
development specifically: the consumer of a declaration is normally
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

**Declaration.** The contents of a dx declaration file (canonical
extension `.md`, per Section 4.1). Comprises a system
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
v0.2.0 CommonMark serialization of the model defined here.

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

An invariant MAY be phrased as a positive requirement (the
implementation MUST do X) or as a negative requirement (the
implementation MUST NOT do Y). Both forms describe observable
properties; the language draws no structural distinction between
them. Negative invariants are how a declaration captures
forbidden behaviors that complement what the system is required
to do. For example:

```markdown
## Invariants

### iface_returns_token

On successful authentication, returns a bearer token in the
Authorization header.

### sec_no_credential_logs

Credentials, tokens, and any other secret material MUST NOT be
written to any log stream or diagnostic output.
```

Both entries above are observable in principle. The first is
straightforward to verify; the second is harder (it requires
checking the absence of a property, not the presence of one) and
is constrained by the limits of observational verification. See
Section 5 for the security implications.

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

### 3.10. Future Directions

This subsection records concepts that are deliberately out of
scope for v0.1.0 but are real candidates for a future revision.
They are described here so that an implementer or contributor can
work around their absence with the right framing, and so that a
later revision can address them without inventing them from
scratch.

**Composition.** v0.1.0 assumes a single declaration per system.
Real-world projects (microservices, monorepos with shared
libraries, cross-language SDKs of one library) routinely have
multiple declarations that share invariants or contracts. A
future revision is expected to define a composition mechanism —
likely a top-level block that allows one declaration to import
named invariants and contracts from another — so that a shared
API contract has exactly one canonical definition. Until then,
the workaround is to duplicate shared invariants across files
and accept the manual reconciliation cost.

**Contract grouping.** v0.1.0 requires every contract to carry
its own `given`, `when`, and `then` clauses. Sets of contracts
that share a precondition (a `given` clause that appears
identically across many entries) currently must repeat that
clause. A future revision MAY introduce a structural grouping
form that lets several `when`/`then` cases share a parent
`given`. Until then, duplication is the workaround; the
serialization deliberately offers no cross-reference mechanism
(Section 4.2) and that limitation is not relaxed for this case.

**Assumption lifecycle states.** Section 3.5 describes a
lifecycle for assumptions (ratify, demote, reject) but provides
no in-band representation of where an assumption sits in that
lifecycle. The current convention relies on the assumption's
location (still in the `Assumptions` block, or moved out) plus
git history. A future revision MAY introduce explicit lifecycle
state, likely as part of the structured-leaf shape sketched in
Section 4.4. Until then, the workaround is to use commit
messages and review comments to track lifecycle state.

These three are not the only directions a future revision might
take. They are the ones that working with the language has
surfaced as real ergonomic limits, and naming them here is meant
to focus future design on changes the existing user base has
actually needed.

## 4. Serialization (v0.2.0)

This section defines a concrete CommonMark serialization of the
conceptual model defined in Section 3. The serialization is
versioned independently of the conceptual model: a future
revision MAY define additional or alternative serializations
without changing Section 3.

This is v0.2.0 of the serialization. v0.1.0 specified a YAML
1.2 serialization; v0.2.0 replaces it. Section 4.8 (Migration)
records the rationale and the hard-cutover policy. v0.1.0 YAML
files are not valid v0.2.0 input.

### 4.1. Physical Format

A declaration MUST be a valid CommonMark [CommonMark] document,
subject to the structural constraints in Section 4.2. The
canonical file extension is `.md`.

CommonMark was selected as the v0.2.0 serialization for four
reasons:

- **Prose-first ergonomics.** Declarations are dominated by
  natural-language sentences (intent statements, invariant
  bodies, contract clauses). CommonMark leaves prose unadorned:
  no surrounding quotes, no block-scalar sigil, no
  indentation-strip rule, no escapes for backticks or quoted
  strings.
- **Universal rendering.** Every code-review surface, issue
  tracker, IDE, and chat client renders CommonMark out of the
  box. A reviewer reading a declaration sees the rendered form
  the way it was meant to be read, with no `.dx` mode required.
- **Expressive leaves.** CommonMark permits inline links, fenced
  code blocks, tables, and emphasis inside the leaf prose of an
  invariant or contract. A URL to an upstream RFC, a code block
  showing exact expected output, or a comparison table fits
  naturally where it belongs.
- **Stable structural subset.** The structural rules in Section
  4.2 use only a small, stable subset of CommonMark (ATX
  headings at levels 1–3 plus one paragraph-leading bold
  pattern), which is unambiguous across CommonMark
  implementations.

YAML 1.2, JSON, TOML, HCL, and a custom DSL were considered and
rejected: YAML for the cognitive tax of block scalars and the
opacity of escapes when prose contains backticks or quoted
strings (the original v0.1.0 selection, replaced because prose
ergonomics dominate); JSON for absence of comments and weak
multi-line support; TOML for awkward multi-line strings; HCL for
ecosystem coupling to a single vendor; a custom DSL for
tree-sitter and editor-tooling costs not justified by the
marginal gain over a constrained CommonMark subset.

### 4.2. Structural Constraints

A declaration uses a two-tier grammar: a strict **structural
layer** that maps CommonMark headings onto the components
defined in Section 3, and an opaque **leaf layer** that captures
the prose between structural boundaries as the value of a
component. The structural layer is enforceable mechanically; the
leaf layer is delegated to CommonMark and is not interpreted
beyond its own rules.

The following constraints define the structural layer. All are
normative.

- **System heading.** A declaration MUST contain exactly one
  ATX heading at level 1 (`#`), and it MUST be the first
  non-blank, non-comment block in the document. Its body is the
  system identifier per Section 3.2.
- **Block headings.** A declaration MUST express each
  conceptual block (Section 3.1) as an ATX heading at level 2
  (`##`) whose body is one of the seven canonical block names,
  spelled in Title Case: `Intent`, `Invariants`, `Assumptions`,
  `Contracts`, `Unconstrained`. The REQUIRED blocks (`Intent`,
  `Invariants`, `Assumptions`) MUST be present; the OPTIONAL
  blocks (`Contracts`, `Unconstrained`) MAY be present. A
  declaration MUST NOT contain more than one `##` heading with
  the same body. A `##` heading whose body is outside this
  vocabulary is a structural error.
- **Key headings.** Within `Invariants`, `Assumptions`,
  `Contracts`, and `Unconstrained`, each entry is introduced by
  an ATX heading at level 3 (`###`) whose body is the entry's
  identifier. The identifier MUST be a slug (conventionally
  snake_case for invariants and assumptions, dot-separated
  segments for assumptions when grouping, kebab- or snake-case
  for contracts and unconstrained). A `###` heading that does
  not appear inside a recognized `##` block is a structural
  error.
- **Intent has no key headings.** The `Intent` block uses a
  fixed sub-structure (Section 4.3.2) rather than free-form
  `###` keys.
- **Contract sub-fields.** Inside a contract's `###` section,
  the three sub-fields are introduced as paragraph-leading
  bolds, exactly: `**Given:**`, `**When:**`, `**Then:**`. The
  sub-field body is the remainder of the paragraph plus any
  contiguous prose up to the next `**Given:**`, `**When:**`,
  `**Then:**`, or `###`/`##`/`#` heading. All three sub-fields
  MUST be present and MUST appear at most once per contract.
- **Reserved heading levels.** ATX heading levels 1, 2, and 3
  are reserved for the structural layer. Heading levels 4 and
  deeper (`####`, `#####`, `######`) are NOT structural and
  MUST be treated as leaf content.
- **Empty blocks.** A `##` block heading immediately followed
  by no `###` children encodes the semantically-meaningful
  empty form from Section 3.5: the block was considered and
  intentionally left empty. The absence of the heading
  altogether is a structural error for REQUIRED blocks.
- **Block ordering.** A declaration SHOULD order its `##`
  block headings as: `Intent`, `Invariants`, `Assumptions`,
  `Contracts`, `Unconstrained`. A file that violates the SHOULD
  is structurally valid but is not in canonical form.
- **Comments.** A declaration MAY contain HTML comments
  (`<!-- ... -->`) at any position. Comments are non-semantic:
  they MUST be preserved by canonical formatting where
  possible, but they MUST NOT affect parsing or carry meaning
  the structural layer interprets.

The following constraints define the leaf layer.

- **Opaque CommonMark.** The body of a `###` key heading (or
  of an `Intent` sub-field, or of a contract sub-field) is
  captured as the verbatim CommonMark text between structural
  boundaries, with leading and trailing blank lines trimmed.
  The text is not further interpreted by this specification.
- **Escapes are the author's responsibility.** When prose
  contains characters CommonMark interprets (asterisks,
  underscores, backticks, square brackets, angle brackets), the
  author MUST escape them per CommonMark rules to render
  literally. Tooling MUST NOT silently transform leaf content.

### 4.3. Schema

This section defines the CommonMark shape of each component of
a declaration. The conceptual purpose of each component is in
Section 3; only the concrete shape is defined here. See
Appendix A for fully-worked examples.

#### 4.3.1. System

The `#` level-1 heading body is the system identifier. It MUST
be a slug (conventionally kebab-case, with no leading digit).

#### 4.3.2. Intent

The `## Intent` block contains exactly one or two sub-sections,
each introduced by a paragraph-leading bold label:

- `**Primary:**` REQUIRED. The body is the primary objective
  per Section 3.3.
- `**Secondary:**` OPTIONAL. The body is an unordered
  CommonMark list (`-`-prefixed items). Order is significant
  and MUST be preserved by canonical formatting.

The Intent block does NOT use `###` key headings. Its structure
is fixed.

#### 4.3.3. Invariants

The `## Invariants` block contains zero or more `###` key
sections. Each key body is the invariant identifier; the
content under the heading (prose, optionally with leaf-layer
CommonMark) is the invariant body.

Each key SHOULD carry a category prefix per Section 3.4.

A `## Invariants` block with zero `###` children is valid; it
asserts that the system has no invariants beyond its intent.

#### 4.3.4. Assumptions

The `## Assumptions` block contains zero or more `###` key
sections. Each key body is the assumption identifier; the
content under the heading is the assumption body, which MUST
cover both the choice that was made and the reason it was
defensible (per Section 3.5).

A `## Assumptions` block with zero `###` children is
semantically meaningful per Section 3.5: it asserts that no
unrecorded heuristic choices were made. This assertion is
distinguishable from omitting the block, which is a structural
error.

#### 4.3.5. Contracts

The `## Contracts` block, if present, contains zero or more
`###` key sections. Each key body is the contract name. The
section MUST contain the three sub-fields `**Given:**`,
`**When:**`, and `**Then:**` per Section 4.2; a contract
missing any sub-field is malformed.

#### 4.3.6. Unconstrained

The `## Unconstrained` block, if present, contains zero or more
`###` key sections. Each key body is the category name; the
content under the heading is the description of the freedom
granted.

### 4.4. Reserved Sub-Field Names

The following sub-field names are reserved within entries of
`Invariants`, `Assumptions`, `Contracts`, and `Unconstrained`.
v0.2.0 does not require them; future revisions of this
serialization MAY attach normative semantics to each.
Implementations MUST NOT use these names as paragraph-leading
bolds for unrelated purposes.

- `**Rule:**` the constraint or assertion text (the body of a
  v0.2.0 leaf, if the author wishes to mark it explicitly).
- `**Reason:**` free-form prose explaining why the entry
  exists.
- `**Author:**` the agent or human responsible for the most
  recent modification of the entry.
- `**Since:**` the spec version or change identifier in which
  the entry first appeared.

In v0.2.0, leaves under `Invariants`, `Assumptions`, and
`Unconstrained` are opaque CommonMark per Section 4.2. The
reserved-sub-field set anticipates a v0.3 transition to a
structured leaf shape that gives these names normative meaning;
reserving them now permits that transition without colliding
with conventions already in use. Appendix A.5 shows a
forward-compatible sketch.

`**Given:**`, `**When:**`, and `**Then:**` are additionally
reserved within `Contracts` entries; their normative meaning is
defined in Section 4.3.5 and is not deferred to a future
revision.

### 4.5. Canonical Form

A canonical declaration satisfies all the SHOULDs in Section
4.2 plus the following formatting rules. Tooling that emits a
canonical form (`dx fmt`) MUST apply all of them; tooling that
reads a declaration MUST accept input that violates them.

- The `#` system heading appears on the first line of the
  document.
- One blank line separates every structural heading from the
  preceding content.
- `##` block headings appear in the canonical order from
  Section 4.2.
- Within each `##` block, `###` key headings appear in
  ascending lexicographic order by identifier.
- Within each `Contracts` entry, the sub-fields appear in the
  fixed order `**Given:**`, `**When:**`, `**Then:**`,
  regardless of the order the author wrote them.
- The document ends with exactly one trailing newline.
- No line carries trailing whitespace.

Canonical form is byte-stable: two declarations that parse to
the same AST MUST produce byte-identical canonical output.

### 4.6. Versioning

This document specifies v0.2.0 of the dx serialization. Future
revisions will be released as `vMAJOR.MINOR.PATCH` per the
following rules:

- **Patch** (`v0.2.x`): clarifications, additional reserved
  names, additional structural checks that reject already-
  questionable input. No new required fields.
- **Minor** (`v0.x.0`): new optional blocks, structured forms
  of existing leaves (gated by the reserved-sub-field
  discipline in Section 4.4), additional conventions.
- **Major** (`v1.0.0` and later): commitment to long-term
  backward compatibility. A major revision MAY also be the
  point at which an alternative serialization is selected,
  with Section 3 unchanged.

v0.2.0 does not include an in-band serialization-version
declaration. A future revision is expected to introduce one
(likely a reserved HTML comment such as `<!-- dx-spec: v0.2 -->`
at the top of the file); until then, declarations have no
in-band version marker and are assumed to target the current
released version of this serialization.

The conceptual content of Section 3 is independent of this
versioning scheme. A change to Section 4 may occur without a
change to Section 3; a change to Section 3 is a more
significant event and would coincide with at least a minor
release.

### 4.7. Relationship to v0.1.0

v0.1.0 of this serialization specified YAML 1.2 as the
canonical physical format. v0.2.0 replaces YAML with CommonMark
in full. The two formats are not compatible at the byte level:
a v0.1.0 file is not valid v0.2.0 input, and a v0.2.0 file is
not valid v0.1.0 input.

The conceptual model in Section 3 is unchanged across the
revision. A v0.1.0 declaration and a v0.2.0 declaration that
encode the same system identifier, intent, invariants,
assumptions, contracts, and unconstrained degrees of freedom
are semantically equivalent under Section 3, and the semantic
delta defined in Section 3.9 produces no operations between
them.

### 4.8. Migration from v0.1.0

The transition from v0.1.0 YAML to v0.2.0 CommonMark is a hard
cutover, not a deprecation. v0.2.0 tooling does not read v0.1.0
input; v0.1.0 tooling does not read v0.2.0 input. The cutover
policy reflects three observations:

- The two serializations encode the same AST, so a one-shot
  conversion is mechanical.
- Maintaining two parsers and two test matrices in tooling
  for an extended deprecation window is a recurring cost paid
  by every contributor, in exchange for benefit only to
  not-yet-migrated downstream files.
- The v0.1.0 file extension (`.dx`) and the v0.2.0 file
  extension (`.md`) are distinct, so the two formats cannot
  collide in a single working tree.

Operators with existing v0.1.0 declarations MUST convert them
before adopting v0.2.0 tooling. A conversion tool is not part
of this specification; operators MAY write a one-shot script
against the v0.1.0 schema (Section 4.3 of the v0.1.0 spec) and
the v0.2.0 schema (Section 4.3 above), or MAY rewrite each
declaration by hand.

## 5. Security Considerations

A declaration is consumed by an agent that produces or modifies
imperative code. An adversarial declaration could therefore
direct the agent to produce code with security vulnerabilities,
backdoors, or unauthorized network or filesystem access.
Mitigation is the responsibility of the agent runtime and the
human reviewer; this specification cannot defend against
adversarial declarations on its own.

Four observations on the threat model are worth recording:

- A declaration enumerates required and forbidden behaviors via
  positive and negative invariants (Section 3.4), but it cannot
  enumerate the universe of unspecified behaviors. The
  `Unconstrained` block (Section 3.7) names aspects intentionally
  left open, but it is not a capability list. An agent reading a
  declaration MAY produce behavior that no invariant or contract
  addresses; whether that behavior is permitted is a question for
  the agent runtime, not for the specification.
- Negative invariants strengthen the threat model but do not close
  it. The conformance model is observational (Section 3.8): a
  judge can verify that a forbidden output does not appear on
  stdout, but cannot verify that a forbidden internal property
  (an unencrypted password in memory, a secret written to a
  database column) is absent. Negative invariants describing
  non-observable properties are valid and worth stating, but
  their verification falls outside what the language guarantees.
- The conformance model is prose-driven. Whoever performs the
  conformance check can be deceived by ambiguous prose; a check
  that does not exercise every contract on every implementation
  can miss regressions. Operators relying on dx-mediated
  workflows SHOULD treat conformance checking as a gating step,
  not as an advisory one.
- The reserved sub-field set in Section 4.4 anticipates an
  `**Author:**` sub-field. An author value is self-asserted; the
  specification
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

[CommonMark] MacFarlane, J., et al., "CommonMark Spec",
  Version 0.31.2, January 2024,
  <https://spec.commonmark.org/0.31.2/>.

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

This appendix provides fully-worked examples of the v0.2.0
CommonMark serialization defined in Section 4. Examples are
illustrative and informative; they do not extend or override the
normative content of the preceding sections.

### A.1. A minimal declaration

The smallest well-formed declaration declares a system
identifier, an intent with a primary objective, and the two
REQUIRED but possibly empty blocks for invariants and
assumptions.

````markdown
# empty

## Intent

**Primary:** A placeholder declaration with no constraints.

## Invariants

## Assumptions
````

### A.2. A complete declaration with all blocks

````markdown
# hello-world

## Intent

**Primary:** Greet a user by name on standard output.

**Secondary:**

- Be friendly.
- Exit cleanly.

## Invariants

### iface_stdout

Writes a single UTF-8 line to stdout terminated by `\n`.

### perf_startup_ms

Cold-start latency must remain under 50ms on commodity hardware.

## Assumptions

### greeting.format

The greeting is "Hello, <name>!" — the spec does not pin
punctuation or word choice; this matches the canonical
POSIX-tutorial form.

## Contracts

### greets_named_user

**Given:** The argument vector contains exactly one non-empty
name.

**When:** The binary is invoked.

**Then:** stdout contains `Hello, <name>!\n` and the exit code
is 0.

## Unconstrained

### language

Any language with a stable POSIX runtime is acceptable.
````

### A.3. Multi-paragraph leaves

Leaf bodies are opaque CommonMark per Section 4.2. A single
paragraph is the common case; multiple paragraphs, fenced code
blocks, and inline links are all permitted.

````markdown
## Invariants

### iface_simple

A single-paragraph invariant body reads as ordinary prose.

### iface_complex

A multi-paragraph invariant body uses one or more paragraphs of
ordinary CommonMark. Subsequent paragraphs are preserved
verbatim. Use additional paragraphs when the constraint
genuinely requires more than one sentence to express.

Where useful, leaf prose MAY include a fenced code block to
pin observable output exactly:

```
expected stdout line one
expected stdout line two
```

It MAY also link to an authoritative reference, e.g. [RFC 7231
§6.5.1](https://www.rfc-editor.org/rfc/rfc7231#section-6.5.1).
````

### A.4. The empty-block contract

An empty `## Assumptions` block is semantically meaningful per
Section 3.5. The two cases below are not equivalent:

````markdown
## Assumptions
````

The block above is present with zero `###` children. Per
Section 3.5, this asserts that no unrecorded heuristic choices
were made.

````markdown
<!-- (no ## Assumptions block at all is malformed) -->
````

Omitting the `## Assumptions` heading entirely is a structural
error for the REQUIRED block.

### A.5. Forward-compatible v0.3 sketch (NOT normative in v0.2.0)

The reserved sub-field set in Section 4.4 anticipates a v0.3
transition to a structured leaf shape. The following is
informative only; v0.2.0 treats the bolds below as ordinary leaf
prose.

````markdown
## Invariants

### perf_cache_ttl

**Rule:** Cache TTL must be strictly 600 seconds.

**Reason:** Upstream API documentation forbids polling faster
than 10 minutes.

**Author:** agent-architect@example.org

**Since:** v0.2.0
````
