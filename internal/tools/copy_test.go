package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestGlobModes(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a", "test1.go"), "test1")
	writeFile(t, filepath.Join(dir, "a", "b", "test2.go"), "test2")
	writeFile(t, filepath.Join(dir, "a", "b", "c", "test3.go"), "test3")

	dst := filepath.Join(dir, "out")
	pattern := filepath.Join(dir, "**", "test*.go")

	tests := []struct {
		name  string
		mode  GlobMode
		files []string
	}{
		{
			"flatten",
			Flatten,
			[]string{
				"test1.go",
				"test2.go",
			},
		},
		{
			"preserve-from-root",
			PreserveFromRoot,
			[]string{
				"a/test1.go",
				"a/b/test2.go",
				"a/b/c/test3.go",
			},
		},
		{
			"preserve-full-path",
			PreserveFullPath,
			[]string{
				filepath.Join(dir, "a", "test1.go"),
				filepath.Join(dir, "a", "b", "test2.go"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.RemoveAll(dst)
			if err := os.MkdirAll(dst, 0755); err != nil {
				t.Fatal(err)
			}
			if err := CopyWithMode(pattern, dst, tt.mode); err != nil {
				t.Fatal(err)
			}

			for _, f := range tt.files {
				if _, err := os.Stat(filepath.Join(dst, f)); err != nil {
					t.Fatalf("missing %s", f)
				}
			}
		})
	}
}

func TestCopySingleFile(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "b.txt")
	writeFile(t, src, "hello")

	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if got := readFile(t, dst); got != "hello" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestCopyDirectory(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "src")
	dst := filepath.Join(dir, "dst")

	writeFile(t, filepath.Join(src, "a.txt"), "a")
	writeFile(t, filepath.Join(src, "sub", "b.txt"), "b")

	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "a.txt")); got != "a" {
		t.Fatalf("unexpected content: %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "sub", "b.txt")); got != "b" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestGlobCopy(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a.go"), "a")
	writeFile(t, filepath.Join(dir, "b.go"), "b")
	writeFile(t, filepath.Join(dir, "c.txt"), "c")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	if err := Copy(filepath.Join(dir, "*.go"), dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if got := readFile(t, filepath.Join(dst, "a.go")); got != "a" {
		t.Fatalf("unexpected content: %q", got)
	}
	if got := readFile(t, filepath.Join(dst, "b.go")); got != "b" {
		t.Fatalf("unexpected content: %q", got)
	}

	if _, err := os.Stat(filepath.Join(dst, "c.txt")); !os.IsNotExist(err) {
		t.Fatalf("unexpected file copied")
	}
}

func TestRecursiveGlob(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a", "test_one.go"), "1")
	writeFile(t, filepath.Join(dir, "a", "b", "test_two.go"), "2")
	writeFile(t, filepath.Join(dir, "a", "b", "ignore.txt"), "x")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "**", "test_*.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	files := []string{
		filepath.Join(dst, "a", "test_one.go"),
		filepath.Join(dst, "a", "b", "test_two.go"),
	}

	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}
}

func TestSkipGitDir(t *testing.T) {
	dir := t.TempDir()

	src := filepath.Join(dir, "src")
	writeFile(t, filepath.Join(src, ".git", "config"), "secret")
	writeFile(t, filepath.Join(src, "file.txt"), "ok")

	dst := filepath.Join(dir, "dst")
	if err := Copy(src, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dst, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git directory should be skipped")
	}

	if got := readFile(t, filepath.Join(dst, "file.txt")); got != "ok" {
		t.Fatalf("unexpected content: %q", got)
	}
}

func TestMultipleSourcesToFileFails(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a.go"), "a")
	writeFile(t, filepath.Join(dir, "b.go"), "b")

	// Create an existing file as destination
	dst := filepath.Join(dir, "out.go")
	writeFile(t, dst, "existing")

	err := Copy(filepath.Join(dir, "*.go"), dst)

	if err == nil || !strings.Contains(err.Error(), "multiple sources") {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestSourceNotFound(t *testing.T) {
	dir := t.TempDir()
	err := Copy(filepath.Join(dir, "nope"), filepath.Join(dir, "out"))

	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDoubleStarAtBeginning(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a", "file.go"), "1")
	writeFile(t, filepath.Join(dir, "a", "b", "file.go"), "2")
	writeFile(t, filepath.Join(dir, "a", "b", "c", "file.go"), "3")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "**", "file.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	files := []string{
		filepath.Join(dst, "a", "file.go"),
		filepath.Join(dst, "a", "b", "file.go"),
		filepath.Join(dst, "a", "b", "c", "file.go"),
	}

	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}
}

func TestDoubleStarInMiddle(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "src", "main.go"), "1")
	writeFile(t, filepath.Join(dir, "src", "pkg", "lib.go"), "2")
	writeFile(t, filepath.Join(dir, "src", "pkg", "sub", "util.go"), "3")
	writeFile(t, filepath.Join(dir, "other", "ignore.go"), "x")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "src", "**", "*.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "main.go"),
		filepath.Join(dst, "pkg", "lib.go"),
		filepath.Join(dst, "pkg", "sub", "util.go"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}

	// Should NOT copy files from other/
	if _, err := os.Stat(filepath.Join(dst, "other", "ignore.go")); !os.IsNotExist(err) {
		t.Fatalf("unexpected file copied from other/")
	}
}

func TestDoubleStarAtEnd(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "docs", "README.md"), "1")
	writeFile(t, filepath.Join(dir, "docs", "guide", "intro.md"), "2")
	writeFile(t, filepath.Join(dir, "docs", "guide", "advanced", "tips.md"), "3")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "docs", "**")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "README.md"),
		filepath.Join(dst, "guide", "intro.md"),
		filepath.Join(dst, "guide", "advanced", "tips.md"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}
}

func TestMultipleDoubleStars(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "a", "x", "test.go"), "1")
	writeFile(t, filepath.Join(dir, "a", "b", "y", "test.go"), "2")
	writeFile(t, filepath.Join(dir, "a", "b", "c", "z", "test.go"), "3")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "a", "**", "**", "test.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "x", "test.go"),
		filepath.Join(dst, "b", "y", "test.go"),
		filepath.Join(dst, "b", "c", "z", "test.go"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}
}

func TestQuestionMarkGlob(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "file1.go"), "1")
	writeFile(t, filepath.Join(dir, "file2.go"), "2")
	writeFile(t, filepath.Join(dir, "file10.go"), "10")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "file?.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "file1.go"),
		filepath.Join(dst, "file2.go"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}

	// file10.go should NOT match (two digits)
	if _, err := os.Stat(filepath.Join(dst, "file10.go")); !os.IsNotExist(err) {
		t.Fatalf("file10.go should not match file?.go")
	}
}

func TestCharacterClassGlob(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "test_a.txt"), "a")
	writeFile(t, filepath.Join(dir, "test_b.txt"), "b")
	writeFile(t, filepath.Join(dir, "test_c.txt"), "c")
	writeFile(t, filepath.Join(dir, "test_x.txt"), "x")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "test_[abc].txt")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "test_a.txt"),
		filepath.Join(dst, "test_b.txt"),
		filepath.Join(dst, "test_c.txt"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}

	// test_x.txt should NOT match
	if _, err := os.Stat(filepath.Join(dst, "test_x.txt")); !os.IsNotExist(err) {
		t.Fatalf("test_x.txt should not match test_[abc].txt")
	}
}

func TestDoubleStarWithMultipleWildcards(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "src", "component_test.go"), "1")
	writeFile(t, filepath.Join(dir, "src", "pkg", "module_test.go"), "2")
	writeFile(t, filepath.Join(dir, "src", "pkg", "deep", "service_test.go"), "3")
	writeFile(t, filepath.Join(dir, "src", "main.go"), "x")
	writeFile(t, filepath.Join(dir, "src", "pkg", "lib.go"), "y")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "src", "**", "*_test.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "component_test.go"),
		filepath.Join(dst, "pkg", "module_test.go"),
		filepath.Join(dst, "pkg", "deep", "service_test.go"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}

	// Non-test files should NOT be copied
	notExpected := []string{
		filepath.Join(dst, "main.go"),
		filepath.Join(dst, "pkg", "lib.go"),
	}

	for _, f := range notExpected {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("unexpected file copied: %s", f)
		}
	}
}

func TestDoubleStarMatchesZeroDirectories(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "src", "file.go"), "1")
	writeFile(t, filepath.Join(dir, "src", "sub", "file.go"), "2")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	// ** should match both 0 and N directories
	pattern := filepath.Join(dir, "src", "**", "file.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	// Should match both direct file and nested file
	expected := []string{
		filepath.Join(dst, "file.go"),
		filepath.Join(dst, "sub", "file.go"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}
}

func TestCombinedGlobPatterns(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "tests", "unit", "user_test.go"), "1")
	writeFile(t, filepath.Join(dir, "tests", "unit", "auth_test.go"), "2")
	writeFile(t, filepath.Join(dir, "tests", "integration", "api_test.go"), "3")
	writeFile(t, filepath.Join(dir, "tests", "e2e", "flow_test.go"), "4")
	writeFile(t, filepath.Join(dir, "tests", "unit", "helper.go"), "x")

	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(dst, 0755); err != nil {
		t.Fatal(err)
	}
	pattern := filepath.Join(dir, "tests", "**", "[ua]*_test.go")

	if err := Copy(pattern, dst); err != nil {
		t.Fatalf("Copy failed: %v", err)
	}

	expected := []string{
		filepath.Join(dst, "unit", "user_test.go"),
		filepath.Join(dst, "unit", "auth_test.go"),
		filepath.Join(dst, "integration", "api_test.go"),
	}

	for _, f := range expected {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("expected file missing: %s", f)
		}
	}

	notExpected := []string{
		filepath.Join(dst, "e2e", "flow_test.go"),
		filepath.Join(dst, "unit", "helper.go"),
	}

	for _, f := range notExpected {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("unexpected file copied: %s", f)
		}
	}
}
