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

### Greeting format

The greeting is "Hello, <name>!" — the spec does not pin the
punctuation or word choice, and this matches the canonical
POSIX-tutorial form.

## Contracts

### Greets a named user

**Given** the argument vector contains exactly one non-empty name.

**When** the binary is invoked.

**Then** stdout contains `Hello, <name>!\n` and the exit code is 0.

## Unconstrained

### Implementation language

Any language with a stable POSIX runtime is acceptable.
