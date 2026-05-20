package lint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsFilesystemPath(t *testing.T) {
	tests := []struct {
		input string
		want  bool
		why   string
	}{
		{"hello.md", true, "no colon, plain filename"},
		{"./hello.md", true, "leading ./"},
		{"../foo/bar.md", true, "leading ../"},
		{"/abs/path/hello.md", true, "absolute path"},
		{"-stdin.md", true, "leading - (flag-like)"},
		{"", true, "empty input"},
		{"C:\\Users\\foo\\hello.md", true, "Windows drive letter"},
		{"D:relative.md", true, "Windows drive letter without backslash"},

		{"HEAD:hello.md", false, "git ref + path"},
		{"HEAD~1:hello.md", false, "git ref with tilde"},
		{"main:examples/hello.md", false, "branch ref + nested path"},
		{"deadbeef:hello.md", false, "SHA-like ref"},
		{"v0.2.0:SPECIFICATION.md", false, "tag ref"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := isFilesystemPath(tc.input)
			if got != tc.want {
				t.Errorf("isFilesystemPath(%q) = %v, want %v (%s)",
					tc.input, got, tc.want, tc.why)
			}
		})
	}
}

func TestReadSource_PlainFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hello.md")
	want := []byte("# t\n")
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}

	got, displayPath, err := readSource(path)
	if err != nil {
		t.Fatalf("readSource: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("data: got %q, want %q", got, want)
	}
	if displayPath != path {
		t.Errorf("displayPath: got %q, want %q", displayPath, path)
	}
}

func TestReadSource_PlainFile_NotFound(t *testing.T) {
	_, _, err := readSource(filepath.Join(t.TempDir(), "missing.md"))
	if err == nil {
		t.Fatal("expected an error for a missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected an os.IsNotExist error, got %v", err)
	}
}

func TestReadSource_GitRev_HEAD(t *testing.T) {
	dir := newTempGitRepo(t)
	want := []byte("# head-version\n")
	gitWriteAndCommit(t, dir, "hello.md", want, "initial")

	t.Chdir(dir)
	got, displayPath, err := readSource("HEAD:hello.md")
	if err != nil {
		t.Fatalf("readSource: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("data: got %q, want %q", got, want)
	}
	if displayPath != "HEAD:hello.md" {
		t.Errorf("displayPath: got %q, want %q", displayPath, "HEAD:hello.md")
	}
}

func TestReadSource_GitRev_HEAD1(t *testing.T) {
	dir := newTempGitRepo(t)

	v1 := []byte("# v1\n")
	gitWriteAndCommit(t, dir, "hello.md", v1, "v1")

	v2 := []byte("# v2\n")
	gitWriteAndCommit(t, dir, "hello.md", v2, "v2")

	t.Chdir(dir)

	// HEAD~1 should give us v1.
	got, _, err := readSource("HEAD~1:hello.md")
	if err != nil {
		t.Fatalf("readSource HEAD~1: %v", err)
	}
	if string(got) != string(v1) {
		t.Errorf("HEAD~1 data: got %q, want %q", got, v1)
	}

	// HEAD should give us v2.
	got, _, err = readSource("HEAD:hello.md")
	if err != nil {
		t.Fatalf("readSource HEAD: %v", err)
	}
	if string(got) != string(v2) {
		t.Errorf("HEAD data: got %q, want %q", got, v2)
	}
}

func TestReadSource_GitRev_BadRev(t *testing.T) {
	dir := newTempGitRepo(t)
	gitWriteAndCommit(t, dir, "hello.md", []byte("# t\n"), "init")

	t.Chdir(dir)
	_, _, err := readSource("doesnotexist:hello.md")
	if err == nil {
		t.Fatal("expected an error for a missing revision")
	}
	if !strings.Contains(err.Error(), "git show") {
		t.Errorf("error should mention git show; got %v", err)
	}
}

func TestReadSource_GitRev_BadPath(t *testing.T) {
	dir := newTempGitRepo(t)
	gitWriteAndCommit(t, dir, "hello.md", []byte("# t\n"), "init")

	t.Chdir(dir)
	_, _, err := readSource("HEAD:nope.md")
	if err == nil {
		t.Fatal("expected an error for a missing path-in-rev")
	}
	if !strings.Contains(err.Error(), "git show") {
		t.Errorf("error should mention git show; got %v", err)
	}
}

func TestReadSource_GitRev_EmptyRev(t *testing.T) {
	_, _, err := readSource(":hello.md")
	if err == nil {
		t.Fatal("expected an error for empty revision")
	}
	if !strings.Contains(err.Error(), "empty revision") {
		t.Errorf("expected 'empty revision' diagnostic; got %v", err)
	}
}

func TestReadSource_GitRev_EmptyPath(t *testing.T) {
	_, _, err := readSource("HEAD:")
	if err == nil {
		t.Fatal("expected an error for empty path")
	}
	if !strings.Contains(err.Error(), "empty path") {
		t.Errorf("expected 'empty path' diagnostic; got %v", err)
	}
}

func TestLintSource_GitRev(t *testing.T) {
	dir := newTempGitRepo(t)
	gitWriteAndCommit(t, dir, "hello.md", []byte(minimalValid), "init")

	t.Chdir(dir)
	res, err := LintSource("HEAD:hello.md")
	if err != nil {
		t.Fatalf("LintSource: %v", err)
	}
	if !res.OK() {
		t.Errorf("expected zero issues, got: %v", res.Issues)
	}
}

// --- helpers ---

// newTempGitRepo creates an isolated git repository under t.TempDir
// with a dummy committer identity. Returns the absolute repo path.
func newTempGitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on $PATH; skipping git-rev tests")
	}
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

// gitWriteAndCommit writes content to dir/relpath, stages it, and
// commits with the given message. Fails the test on any error.
func gitWriteAndCommit(t *testing.T, dir, relpath string, content []byte, msg string) {
	t.Helper()
	full := filepath.Join(dir, relpath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", relpath},
		{"commit", "-q", "-m", msg},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}
