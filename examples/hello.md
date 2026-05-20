# hello-world

## Intent

- Greet a user by name on standard output.
- Be friendly.
- Exit cleanly.

## Invariants

### Cold-start latency

Cold-start latency must remain under 50ms on commodity hardware.

### Single line on stdout

Writes a single UTF-8 line to stdout terminated by `\n`.

## Assumptions

### Intent secondary shape

`intent.secondary` is modelled as a list of strings; see pkg/ast.

## Contracts

### Greets a named user

**Given:** The argument vector contains exactly one non-empty name.

**When:** The binary is invoked.

**Then:** stdout contains `Hello, <name>!\n` and the exit code is 0.

## Unconstrained

### Implementation language

Any language with a stable POSIX runtime is acceptable.
