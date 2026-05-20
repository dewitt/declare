package ast

import "testing"

func TestSlug(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		// Basic prose.
		{"Greets a named user", "greets_a_named_user"},
		{"Cold-start latency", "cold_start_latency"},
		{"Weather CLI", "weather_cli"},

		// Category-prefix convention.
		{"Interface: single line on stdout", "interface_single_line_on_stdout"},
		{"Performance: cold-start latency", "performance_cold_start_latency"},

		// Already-slug-shaped inputs round-trip.
		{"already_a_slug", "already_a_slug"},
		{"snake_case", "snake_case"},
		{"camelCase", "camelcase"},

		// Punctuation and special characters collapse to one
		// underscore.
		{"Hello, world!", "hello_world"},
		{"foo--bar", "foo_bar"},
		{"foo   bar", "foo_bar"},
		{"foo___bar", "foo_bar"},

		// Digits are alphanumeric.
		{"perf p99 50ms", "perf_p99_50ms"},
		{"v0.2.0", "v0_2_0"},

		// Leading/trailing punctuation trimmed.
		{"  leading whitespace", "leading_whitespace"},
		{"trailing dots...", "trailing_dots"},
		{"!!!shout!!!", "shout"},

		// Empty / pathological inputs.
		{"", ""},
		{"!!!", ""},
		{"   ", ""},

		// Non-ASCII letters are treated as non-alphanumeric and
		// collapse (the slug is ASCII-only by design; humans can
		// still use Unicode in the heading text itself).
		{"naïve résumé", "na_ve_r_sum"},

		// Backticks inside a heading (a likely real case for
		// invariants that mention symbols).
		{"Reads `WEATHER_API_KEY`", "reads_weather_api_key"},
	}

	for _, tc := range cases {
		got := Slug(tc.in)
		if got != tc.want {
			t.Errorf("Slug(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
