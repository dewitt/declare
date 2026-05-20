package lint

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Source-resolution helpers for `dx lint` and `dx diff`.
//
// We accept two forms of input on the command line:
//
//  1. A filesystem path -- the historical default. Read from disk.
//  2. A `<rev>:<path>` pair, mirroring `git show` syntax. We resolve
//     the file content by shelling out to `git show <rev>:<path>`,
//     which works against any reachable git object: a SHA, a tag, a
//     branch, or a relative ref like `HEAD~1`.
//
// The two forms are distinguished syntactically so that a plain file
// like `./examples/hello.md` is never confused for a git ref. Any
// input that begins with `./` `/` or `-`, or that lacks a colon, is
// treated as a filesystem path.

// readSource resolves an input string to (data, displayPath, error).
//
// The returned displayPath is what should appear in diagnostic
// messages -- callers should use it instead of the input verbatim
// because it is canonical (e.g., we trim leading `./`).
func readSource(input string) ([]byte, string, error) {
	if isFilesystemPath(input) {
		data, err := os.ReadFile(input)
		if err != nil {
			return nil, input, err
		}
		return data, input, nil
	}

	// Parse <rev>:<path>. We've already established the colon exists
	// via isFilesystemPath returning false.
	idx := strings.IndexByte(input, ':')
	rev, path := input[:idx], input[idx+1:]
	if rev == "" {
		return nil, input, fmt.Errorf("invalid revision spec %q: empty revision", input)
	}
	if path == "" {
		return nil, input, fmt.Errorf("invalid revision spec %q: empty path", input)
	}

	// Shell out to `git show`. This implicitly requires the caller to
	// be inside a git working tree; we surface git's stderr verbatim
	// when it isn't, which is more useful than wrapping it.
	cmd := exec.Command("git", "show", input)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		if stderr != "" {
			return nil, input, fmt.Errorf("git show %s: %s", input, stderr)
		}
		return nil, input, fmt.Errorf("git show %s: %w", input, err)
	}
	// displayPath includes the rev so diagnostics tell the user
	// which revision the issue came from.
	return out, input, nil
}

// isFilesystemPath reports whether input should be treated as a
// filesystem path rather than a git revision spec.
//
// Rules (first match wins):
//   - Inputs with no colon are filesystem paths.
//   - Inputs starting with `./`, `../`, `/`, or `-` are filesystem
//     paths even if they contain a colon (e.g., a flag-like name or
//     an unusual filename).
//   - Inputs whose pre-colon segment is a single character are
//     treated as filesystem paths so Windows-style `C:\foo` is not
//     mistaken for a git ref. Real git revs are ~never one character.
//   - Everything else is a `<rev>:<path>` git spec.
func isFilesystemPath(input string) bool {
	if input == "" {
		return true
	}
	if !strings.ContainsRune(input, ':') {
		return true
	}
	if strings.HasPrefix(input, "./") ||
		strings.HasPrefix(input, "../") ||
		strings.HasPrefix(input, "/") ||
		strings.HasPrefix(input, "-") {
		return true
	}
	if idx := strings.IndexByte(input, ':'); idx == 1 {
		return true
	}
	return false
}
