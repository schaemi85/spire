package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsTextFile(t *testing.T) {
	dir := t.TempDir()

	// Text file
	textFile := filepath.Join(dir, "hello.txt")
	os.WriteFile(textFile, []byte("Hello, world!\n"), 0644)

	if !IsTextFile(textFile) {
		t.Error("expected hello.txt to be detected as text")
	}

	// Binary file (contains null bytes)
	binFile := filepath.Join(dir, "binary.bin")
	os.WriteFile(binFile, []byte{0x00, 0x01, 0x02, 0xFF}, 0644)

	if IsTextFile(binFile) {
		t.Error("expected binary.bin to be detected as binary")
	}

	// Non-existent file
	if IsTextFile(filepath.Join(dir, "missing.txt")) {
		t.Error("expected missing file to return false")
	}

	// Empty file (should be text)
	emptyFile := filepath.Join(dir, "empty.txt")
	os.WriteFile(emptyFile, []byte{}, 0644)

	if !IsTextFile(emptyFile) {
		t.Error("expected empty file to be detected as text")
	}
}

func TestReplaceInFiles(t *testing.T) {
	dir := t.TempDir()

	// Create files with placeholders
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package __PKG__\n\nfunc __FUNC__() {}"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# __PKG__\nWelcome to __PKG__"), 0644)

	replacements := []Replacement{
		{Placeholder: "__PKG__", Replacement: "myapp"},
		{Placeholder: "__FUNC__", Replacement: "Run"},
	}

	if err := ReplaceInFiles(dir, replacements, nil); err != nil {
		t.Fatalf("ReplaceInFiles() error: %v", err)
	}

	mainContent, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if string(mainContent) != "package myapp\n\nfunc Run() {}" {
		t.Errorf("main.go = %q", string(mainContent))
	}

	readmeContent, _ := os.ReadFile(filepath.Join(dir, "readme.md"))
	if string(readmeContent) != "# myapp\nWelcome to myapp" {
		t.Errorf("readme.md = %q", string(readmeContent))
	}
}

func TestReplaceInFilesIgnoresDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create an ignored directory with a file
	ignoredDir := filepath.Join(dir, "vendor")
	os.MkdirAll(ignoredDir, 0755)
	os.WriteFile(filepath.Join(ignoredDir, "lib.go"), []byte("package __PKG__"), 0644)

	// Create a non-ignored file
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package __PKG__"), 0644)

	replacements := []Replacement{
		{Placeholder: "__PKG__", Replacement: "myapp"},
	}

	if err := ReplaceInFiles(dir, replacements, []string{"vendor"}); err != nil {
		t.Fatalf("ReplaceInFiles() error: %v", err)
	}

	// Ignored file should be unchanged
	vendorContent, _ := os.ReadFile(filepath.Join(ignoredDir, "lib.go"))
	if string(vendorContent) != "package __PKG__" {
		t.Errorf("vendor file modified: %q", string(vendorContent))
	}

	// Non-ignored file should be changed
	mainContent, _ := os.ReadFile(filepath.Join(dir, "main.go"))
	if string(mainContent) != "package myapp" {
		t.Errorf("main.go = %q", string(mainContent))
	}
}

func TestReplaceInFilesSkipsBinary(t *testing.T) {
	dir := t.TempDir()

	// Binary file
	os.WriteFile(filepath.Join(dir, "image.bin"), []byte{0x00, 0x01, 0x02, 0xFF}, 0644)

	replacements := []Replacement{
		{Placeholder: "\x01", Replacement: "replaced"},
	}

	if err := ReplaceInFiles(dir, replacements, nil); err != nil {
		t.Fatalf("ReplaceInFiles() error: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "image.bin"))
	if len(content) != 4 || content[1] != 0x01 {
		t.Error("binary file should not be modified")
	}
}

func TestRenamePathsWithPlaceholder(t *testing.T) {
	dir := t.TempDir()

	// Create files and directories with placeholder
	os.WriteFile(filepath.Join(dir, "__name__-service.go"), []byte("content"), 0644)
	os.MkdirAll(filepath.Join(dir, "__name__-dir", "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "__name__-dir", "sub", "file.go"), []byte("pkg"), 0644)

	if err := RenamePathsWithPlaceholder(dir, "__name__", "user"); err != nil {
		t.Fatalf("RenamePathsWithPlaceholder() error: %v", err)
	}

	// Check file rename
	if _, err := os.Stat(filepath.Join(dir, "user-service.go")); err != nil {
		t.Error("expected user-service.go to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "__name__-service.go")); err == nil {
		t.Error("__name__-service.go should not exist after rename")
	}

	// Check directory rename (and nested files preserved)
	if _, err := os.Stat(filepath.Join(dir, "user-dir", "sub", "file.go")); err != nil {
		t.Error("expected user-dir/sub/file.go to exist")
	}
}

func TestRenamePathsWithPlaceholderNoMatch(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "normal.go"), []byte("content"), 0644)

	if err := RenamePathsWithPlaceholder(dir, "__name__", "user"); err != nil {
		t.Fatalf("RenamePathsWithPlaceholder() error: %v", err)
	}

	// File should remain unchanged
	if _, err := os.Stat(filepath.Join(dir, "normal.go")); err != nil {
		t.Error("normal.go should still exist")
	}
}

func TestRemoveFilesorDirs(t *testing.T) {
	dir := t.TempDir()

	f1 := filepath.Join(dir, "file1.txt")
	f2 := filepath.Join(dir, "file2.txt")
	d1 := filepath.Join(dir, "mydir")

	os.WriteFile(f1, []byte("content"), 0644)
	os.WriteFile(f2, []byte("content"), 0644)
	os.MkdirAll(filepath.Join(d1, "sub"), 0755)
	os.WriteFile(filepath.Join(d1, "sub", "nested.txt"), []byte("content"), 0644)

	if err := RemoveFilesorDirs(f1, d1); err != nil {
		t.Fatalf("RemoveFilesorDirs() error: %v", err)
	}

	if _, err := os.Stat(f1); err == nil {
		t.Error("file1.txt should be removed")
	}
	if _, err := os.Stat(d1); err == nil {
		t.Error("mydir should be removed")
	}
	// file2 should remain
	if _, err := os.Stat(f2); err != nil {
		t.Error("file2.txt should still exist")
	}
}

func TestRemoveFilesorDirsNonExistent(t *testing.T) {
	// RemoveAll on non-existent path returns nil, so this should succeed
	err := RemoveFilesorDirs("/nonexistent/path/abc123")
	if err != nil {
		t.Errorf("RemoveFilesorDirs on non-existent path should not error: %v", err)
	}
}

func TestRenamePathsWithPlaceholderDeepNesting(t *testing.T) {
	dir := t.TempDir()

	// Create deeply nested directories with placeholder at multiple levels
	deep := filepath.Join(dir, "__name__", "middle", "__name__-inner")
	os.MkdirAll(deep, 0755)
	os.WriteFile(filepath.Join(deep, "__name__-file.txt"), []byte("content"), 0644)

	if err := RenamePathsWithPlaceholder(dir, "__name__", "app"); err != nil {
		t.Fatalf("RenamePathsWithPlaceholder() error: %v", err)
	}

	expected := filepath.Join(dir, "app", "middle", "app-inner", "app-file.txt")
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected %s to exist after rename", expected)
	}
}
