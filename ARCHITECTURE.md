# Architecture: The `declare` Paradigm

`declare` is a Heuristic Intermediate Representation (HIR) and toolchain designed for the agentic AI era. It acts as the boundary layer between human intent and machine-generated implementation.

## 1. Core Philosophy

In a paradigm where LLMs write the majority of imperative code, the human role shifts from syntax authoring to system design and constraint definition. However, current "vibe coding" suffers from silent AI hallucinations and semantic drift. 

`declare` solves this by forcing a formal, version-controllable design phase via `.dx` files. 
- **Decoupling:** The `.dx` file defines the *what* and the *constraints*. The generated imperative code defines the *how*.
- **Orthogonality:** The specification must never leak imperative logic or dictate internal architecture (e.g., unit tests). It focuses purely on observable system state and Service Level Objectives (SLOs).
- **Formalized Hallucination:** Agents are required to explicitly declare heuristic leaps in an `assumptions` block, turning silent hallucinations into auditable, promotable workflow states.

## 2. The `.dx` Artifact

`.dx` files are written in a strictly constrained subset of YAML. This provides syntax highlighting out-of-the-box, ergonomic multi-line string support for humans, and a deterministic Abstract Syntax Tree (AST) for machine parsing.

### Core Blocks
*   **`intent`**: High-level semantic goals and business logic.
*   **`invariants`**: Non-negotiable physical, systemic, or performance constraints.
*   **`assumptions`**: Heuristics the agent filled in, waiting for human promotion or rejection.
*   **`contracts`**: Black-box verification rules (state transitions, standard I/O, exit codes) used to prove the implementation satisfies the invariants.
*   **`unconstrained`**: Explicitly declared degrees of freedom, preventing over-specification and guiding agent restraint.

## 3. The Multi-Agent Loop

The `declare` ecosystem is designed to be operated by a swarm of specialized agents, coordinated via the `.dx` file.

1.  **The Archaeologist (Extraction):** Distills legacy imperative code into semantic intent and observable invariants, outputting a base `.dx` file.
2.  **The Architect (Refinement):** Modernizes the `.dx` file, applies system-wide constraints, and flags semantic gaps as `assumptions`.
3.  **The Implementer (Coding):** Reads the `.dx` file (often via an AST-compiled JSON export) and generates the imperative code. It operates strictly within the defined invariants.
4.  **The Judge (Verification):** Executes the implementation against the `contracts` block using black-box testing. It cross-references failures with the `.dx` file and issues specific correction prompts to the Implementer.

## 4. The CLI Toolchain

The `declare` binary contains **no LLM**. It is a blindingly fast, deterministic toolchain built to enforce the `.dx` specification.

The v0.1.0 command set, with implementation status:

*   `declare lint` (implemented): Enforces SPEC §2 (no anchors/aliases, no folded scalars, no custom tags, scalar leaves under `invariants`/`assumptions`/`unconstrained`) and SPEC §3 (required-key presence). Walks the retained YAML node graph for the physical-rule checks; strict-decodes the AST for the structural-decode pass.
*   `declare fmt` (implemented): Reformats a `.dx` file to its canonical representation: top-level keys in SPEC §2 order, alphabetized maps, literal block scalars for multi-line strings, no trailing whitespace. Idempotent (`fmt(fmt(x)) == fmt(x)`) and AST-preserving. Prints to stdout by default; `--write` overwrites in place.
*   `declare diff` (implemented): Parses two `.dx` files and outputs a semantic ledger of operations (`[ADDED]`, `[REMOVED]`, `[MUTATED]`, `[PROMOTED]`, `[DEMOTED]`, `[RENAMED]`), rather than a noisy text diff.
*   `declare export` (implemented): Emits the AST as canonical YAML (default) or compact JSON, with comments stripped, for ingestion by agent context windows. Byte-stable for the same AST so two agents can agree on hashes.
*   `declare contracts list` (implemented): Enumerates contract identifiers in alphabetical order. Defaults to plain text (one ID per line, shell-friendly); `--verbose` adds an indented one-line preview of given/when/then; `--format=json` emits a structured object with full bodies. Used by the [`judge`](skills/judge/SKILL.md) skill to drive a deterministic walk over each contract.
*   `declare verify` (deferred to v0.2 per SPEC §4): Will run the `contracts:` block as a black-box test harness. Until it ships, contract execution is performed by an agent under the [`judge`](skills/judge/SKILL.md) skill, typically driven by `declare contracts list` to enumerate what to check.

See [`skills/declare-toolchain/SKILL.md`](skills/declare-toolchain/SKILL.md) for invocation details, exit codes, and the post-merge ritual.

## 5. Security & Safety

Agents modify `.dx` files directly, but rely on `declare lint` in their event loop to catch structural entropy. Semantic conflicts are resolved through the `declare diff` tool, which produces a deterministic ledger of how the spec evolved between two revisions.

The auditable design-decision ledger envisioned for the long term — where each invariant carries a `reason:`, an `author:`, and a `since:` field — is reserved as a future-compatibility shape in SPEC §6 but is **not** yet expressible in v0.1.0 (where leaves under `invariants:` / `assumptions:` / `unconstrained:` are scalar strings). Today, the audit trail lives in git: the commit that introduced an invariant is its provenance, and `declare diff` is the lens for reading the change.
