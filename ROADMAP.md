# Roadmap

Forward-looking work in `declare`, organized by priority. This file
is an **index, not a source of truth** — each item links to the
authoritative location where the gap is described in detail. Update
the linked location when an item moves; this index follows.

The convention in this repo is that gaps live next to the surface
they affect: per-document "Known gaps" sections, per-skill anti-pattern
lists, and SPEC.md "future direction" notes. This file aggregates
those into a single prioritized view so we don't lose them.

## Priority 1: gaps that block end-user journeys today

After several rounds of v0.1.x work, only two practical gaps remain.
Both need design before code, which is the right state to be in.

### `declare verify` (the contract executor)

**Bites:** the [judge phase](docs/journeys/port-to-another-language.md#5-judge-phase-verify-against-the-contracts)
of every journey. Today an agent walks each contract by prose; this
doesn't scale past a handful of contracts and is the single biggest
practical limitation in v0.1.0.

**Source of truth:** [SPEC.md §4](SPEC.md#4-verification-model)
(deferred to v0.2 with rationale) and the
["Gap 1" entry](docs/journeys/port-to-another-language.md#gap-1--no-declare-verify-high-priority)
in the port journey.

### Implementer "no peeking at source" enforcement

**Bites:** the [implementer phase](docs/journeys/port-to-another-language.md#4-implementer-phase-synthesize-the-new-code)
of the port journey. The discipline is honor-system today; an
undisciplined agent inherits the original implementation's biases
and reduces the journey to a translation pass.

**Source of truth:**
[Gap 2](docs/journeys/port-to-another-language.md#gap-2--no-mechanism-to-enforce-implementer-must-not-read-the-source-medium-priority)
in the port journey. May require a convention (an allowlist file the
agent runtime respects) rather than a CLI feature; it crosses into
agent-runtime territory.

## Spec-level future directions

These are explicit forward-references in SPEC.md — design choices
deferred to v0.2 by deliberate scope-cutting in v0.1.0.

| Direction | Source of truth |
| --------- | --------------- |
| Audit-trail leaf shape (`rule:`, `reason:`, `author:`, `since:`) | [SPEC.md §6](SPEC.md#6-reserved-field-names-future-compatibility) |
| Structural merge tool (`declare merge`) | [SPEC.md §5](SPEC.md#5-concurrent-edit-conflict-resolution) |
| In-band spec-version declaration (likely `dx_spec:`) | [SPEC.md §7](SPEC.md#7-versioning) |

## Documentation backlog

The four user journeys below are the canonical end-to-end use cases
for `declare`. All four are documented; the first has been
clean-room-tested end-to-end, the other three have a `> **TODO:**`
worked-example slot pending the same treatment.

**Source of truth:** [`docs/journeys/README.md`](docs/journeys/README.md).

- [port-to-another-language](docs/journeys/port-to-another-language.md) — documented + clean-room-tested.
- [greenfield-development](docs/journeys/greenfield-development.md) — documented; worked example pending.
- [add-a-feature](docs/journeys/add-a-feature.md) — documented; worked example pending.
- [add-a-feature-to-multiple-implementations](docs/journeys/add-a-feature-to-multiple-implementations.md) — documented; worked example pending.

Possible future journeys (not currently planned, but plausible
candidates if there's demand) are listed in the journeys index.

## How to add a new item

1. Decide where the *authoritative* description belongs:
   - Tooling gap → "Known gaps" section of the relevant journey doc
     (or create one if no journey exposes it yet).
   - Spec-level → a new SPEC.md "future direction" subsection.
   - Documentation gap → the docs/journeys index.
2. Add an entry to the appropriate priority section here, linking
   to the authoritative location.
3. Do **not** duplicate the description here. This file is the index;
   the cited location is the source of truth.
