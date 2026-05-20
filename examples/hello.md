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

### ast.intent_secondary_shape

`intent.secondary` is modelled as a list of strings; see pkg/ast.

## Contracts

### greets_named_user

**Given:** The argument vector contains exactly one non-empty name.

**When:** The binary is invoked.

**Then:** stdout contains `Hello, <name>!\n` and the exit code is 0.

## Unconstrained

### language

Any language with a stable POSIX runtime is acceptable.
