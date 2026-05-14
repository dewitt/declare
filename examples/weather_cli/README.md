# weather-cli — the canonical working example

This directory is the round-trip demonstration that motivated the
`dx` design discussion. It exercises every block of the `.dx`
schema and is small enough to read end-to-end.

```
weather_cli/
├── system.dx              # The single source of truth.
├── impl_cpp/
│   └── weather_cli.cc     # Legacy artifact (the archaeologist's input).
└── impl_python/
    └── weather_cli.py     # Agentic synthesis (the implementer's output).
```

## The narrative

1. **Archaeologist:** reads `impl_cpp/weather_cli.cc`, writes `system.dx`.
   It captures the *observable* behavior (env-var handling, cache TTL by
   mtime, zip-code argument, `--json` flag, exit codes) and explicitly
   logs every guess as an `assumptions:` entry.
2. **Architect:** refines `system.dx` — promotes assumptions to
   invariants where appropriate, prunes over-specifications into
   `unconstrained:`, and adds matching `contracts:`.
3. **Implementer:** reads only `system.dx` (never `impl_cpp/`) and
   produces `impl_python/weather_cli.py`. The C++ implementation is
   never imported as ground truth — only the spec is.
4. **Judge:** runs every entry in `contracts:` against the
   implementation as a black-box test. Both the C++ and Python
   implementations should pass identically.

## Why this example, not "hello world"

A toy `print("Hello")` lacks state, boundaries, and ambiguity. Weather
fetching has all three: network I/O (mocked here for offline use),
caching with a TTL, environment-variable secrets, and human-vs-machine
output modes. That surface area is the minimum needed to stress-test
the `.dx` syntax — and it is exactly what was discussed in the
project's design conversation.

## Running the implementations

These are illustrative reference implementations, not part of the
`dx` build:

```bash
# C++
c++ -std=c++17 impl_cpp/weather_cli.cc -o /tmp/weather_cli_cpp
WEATHER_API_KEY=any-non-empty /tmp/weather_cli_cpp 98101
WEATHER_API_KEY=any-non-empty /tmp/weather_cli_cpp 98101 --json

# Python
WEATHER_API_KEY=any-non-empty python3 impl_python/weather_cli.py 98101
WEATHER_API_KEY=any-non-empty python3 impl_python/weather_cli.py 98101 --json
```

Both fetches are mocked, so the network is not contacted.

## What this example deliberately doesn't show (yet)

- **Automated `dx verify`.** The `contracts:` block is currently
  human-runnable prose, not a machine-executable harness. Closing that
  gap is a planned `dx verify` command; until then, the `judge`
  skill describes how a coding agent walks each contract by hand.
- **Real network calls.** Mocked to keep the example offline-safe and
  deterministic.
- **A `dx diff` between an old and new spec.** Once we have a
  meaningful v0 → v1 evolution of `system.dx`, this directory is a
  natural place to demonstrate it.
