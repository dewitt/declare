# weather-cli

## Intent

**Primary:** Fetch and display current weather conditions for a given
US zip code on the command line.

**Secondary:**

- Cache results locally to prevent rate-limiting from the upstream provider.
- Be friendly to scripts as well as humans (offer a JSON output mode).

## Invariants

### iface_json_flag

When the `--json` flag is passed, stdout must contain a single
machine-parseable JSON object describing the weather. When the
flag is absent, stdout must contain a human-readable single-line
summary.

### iface_zipcode_arg

The implementation must accept a US zip code as the first
positional argument. Invocation without a zip code must fail
with a non-zero exit code and a usage message on stderr.

### perf_no_redundant_fetch

For repeated invocations with the same zip code within 600
seconds, the implementation must not issue a second upstream
network request.

### sec_api_key_env

The upstream API key must be read from the `WEATHER_API_KEY`
environment variable. The implementation must not embed any API
key as source-code literal, build-time constant, or persisted
configuration.

### sec_api_key_required

If `WEATHER_API_KEY` is unset or empty, the implementation must
fail with a non-zero exit code and an error message on stderr,
without making any upstream network request.

## Assumptions

### cache.location

The cache is stored at `~/.weather_cache.json` because no cache
location was specified. A future invariant may pin this to an
XDG-compliant path; until then, the home-directory default
matches the original C++ implementation extracted by the
archaeologist.

### cache.ttl_seconds

The 600-second cache TTL is encoded in the `perf_no_redundant_fetch`
invariant as a hard number. It was extracted from the legacy C++
`CACHE_TTL = 600` and retained because the upstream documentation
suggests poll intervals on this order of magnitude. A future
architect pass may relax this to an SLO ("the system must not
exceed N upstream calls per minute under steady-state load").

### network.provider

The upstream weather provider is OpenMeteo. The intent did not
pin a provider; OpenMeteo was chosen because it does not require
an API key for basic queries and matches the worked example from
the project's design discussion. (Note: this slightly contradicts
the `sec_api_key_required` invariant for the OpenMeteo case,
which is why the invariant is phrased to fail closed regardless
of provider.)

## Contracts

### caches_repeat_queries

**Given:** `WEATHER_API_KEY` is set and the binary has just
successfully returned weather for zip code "98101".

**When:** The binary is invoked again with the same zip code
within 600 seconds, with the upstream network blocked at the OS
level.

**Then:** Exit code is 0 and stdout contains the same weather
data as the previous invocation.

### emits_human_text_by_default

**Given:** A valid zip code "98101" is supplied as the first
argument and `WEATHER_API_KEY` is set to any non-empty value.

**When:** The binary is invoked with no `--json` flag.

**Then:** Exit code is 0 and stdout contains a single
human-readable line that mentions the zip code and at least a
temperature or condition description.

### emits_json_with_flag

**Given:** A valid zip code "98101" is supplied as the first
argument, `WEATHER_API_KEY` is set, and `--json` is passed.

**When:** The binary is invoked.

**Then:** Exit code is 0 and stdout is a single line of valid
JSON whose top-level object includes a temperature and a
condition description.

### rejects_missing_api_key

**Given:** A zip code is supplied as the first argument and the
`WEATHER_API_KEY` environment variable is unset.

**When:** The binary is invoked.

**Then:** Exit code is non-zero, stderr names the missing
environment variable, and no upstream network request was made.

### rejects_missing_zipcode

**Given:** No positional arguments are supplied and
`WEATHER_API_KEY` is set.

**When:** The binary is invoked.

**Then:** Exit code is non-zero and stderr contains the
substring "Usage" or a zip-code-related error message.

## Unconstrained

### cache_format

The on-disk cache format is unconstrained provided the
`perf_no_redundant_fetch` invariant holds. The C++ reference
uses flat JSON; the Python reference uses structured JSON; both
satisfy the contract.

### internal_data_structures

All internal data structures (in-memory cache, request batching,
HTTP client choice, JSON library) are unconstrained.

### language

Implementation language is unconstrained. The reference
implementations under `impl_cpp/` and `impl_python/` are
illustrative, not normative.

### output_phrasing

The exact wording of the human-readable output is unconstrained
provided it includes the zip code and at least one weather
attribute, per `emits_human_text_by_default`.
