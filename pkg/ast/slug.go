package ast

import "strings"

// Slug reduces an arbitrary CommonMark heading body to the stable
// machine name defined by SPEC §4.2 ("Key headings"):
//
//   - The result is lowercase.
//   - ASCII letters and digits are preserved verbatim.
//   - Any run of characters outside that set (whitespace, punctuation,
//     non-ASCII letters, etc.) collapses to a single underscore.
//   - Leading and trailing underscores are trimmed.
//
// The function is total: every input produces some output. An input
// of all-non-alphanumerics produces the empty string, which the lint
// pass treats as a structural error ("empty slug").
//
// Examples:
//
//	Slug("Greets a named user")                  == "greets_a_named_user"
//	Slug("Interface: single line on stdout")     == "interface_single_line_on_stdout"
//	Slug("Cold-start latency")                   == "cold_start_latency"
//	Slug("**already** slugged_form")             == "already_slugged_form"
//	Slug("Weather CLI")                          == "weather_cli"
func Slug(heading string) string {
	var b strings.Builder
	b.Grow(len(heading))
	prevUnderscore := false
	for _, r := range heading {
		if isSlugAlphanumeric(r) {
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	out := b.String()
	// Trim trailing underscore if the loop appended one.
	for strings.HasSuffix(out, "_") {
		out = out[:len(out)-1]
	}
	return out
}

func isSlugAlphanumeric(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	}
	return false
}
