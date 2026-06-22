package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/schaemi85/spire/internal/manifest"
)

func TestRenderProjectFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, filepath.Join(dir, "readme.md"), "# [[ .slots.AppName ]]\nVersion: [[ .slots.Version ]]")
	writeTestFile(t, filepath.Join(dir, "plain.txt"), "no template delimiters here")

	rc := NewResolveContext()
	rc.Slots["AppName"] = "My App"
	rc.Slots["Version"] = "2.0.0"

	if err := RenderProjectFiles(dir, rc, nil); err != nil {
		t.Fatalf("RenderProjectFiles() error: %v", err)
	}

	rendered := readTestFile(t, filepath.Join(dir, "readme.md"))
	if rendered != "# My App\nVersion: 2.0.0" {
		t.Errorf("rendered = %q", rendered)
	}

	plain := readTestFile(t, filepath.Join(dir, "plain.txt"))
	if plain != "no template delimiters here" {
		t.Errorf("plain file modified: %q", plain)
	}
}

func TestRenderProjectFilesWithPipelines(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "config.yaml"), "name: [[ .slots.AppName | slugify ]]")

	rc := NewResolveContext()
	rc.Slots["AppName"] = "My Cool App"

	if err := RenderProjectFiles(dir, rc, nil); err != nil {
		t.Fatalf("RenderProjectFiles() error: %v", err)
	}

	got := readTestFile(t, filepath.Join(dir, "config.yaml"))
	if got != "name: my-cool-app" {
		t.Errorf("rendered = %q", got)
	}
}

func TestRenderProjectFilesIgnorePaths(t *testing.T) {
	dir := t.TempDir()

	ignoredDir := filepath.Join(dir, ".git")
	os.Mkdir(ignoredDir, 0755)
	writeTestFile(t, filepath.Join(ignoredDir, "config"), "[[ .slots.AppName ]]")

	writeTestFile(t, filepath.Join(dir, "main.go"), "package [[ .slots.Package ]]")

	rc := NewResolveContext()
	rc.Slots["AppName"] = "test"
	rc.Slots["Package"] = "main"

	if err := RenderProjectFiles(dir, rc, []string{".git"}); err != nil {
		t.Fatalf("RenderProjectFiles() error: %v", err)
	}

	got := readTestFile(t, filepath.Join(ignoredDir, "config"))
	if got != "[[ .slots.AppName ]]" {
		t.Errorf("ignored file was modified: %q", got)
	}

	got = readTestFile(t, filepath.Join(dir, "main.go"))
	if got != "package main" {
		t.Errorf("rendered = %q", got)
	}
}

func TestRenderProjectFilesInvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "bad.txt"), "[[ .slots.Name | nonExistentFunc999 ]]")

	rc := NewResolveContext()
	err := RenderProjectFiles(dir, rc, nil)
	if err == nil {
		t.Error("expected error for invalid template, got nil")
	}
}

func TestApplyPathRenames(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "projectname.txt"), "content")
	os.MkdirAll(filepath.Join(dir, "projectname", "sub"), 0755)
	writeTestFile(t, filepath.Join(dir, "projectname", "sub", "file.go"), "package sub")

	rc := NewResolveContext()
	rc.Slots["Slug"] = "my-app"

	renames := []manifest.PathRename{
		{Pattern: "projectname", Expression: "[[ .slots.Slug ]]"},
	}

	if err := ApplyPathRenames(dir, renames, rc); err != nil {
		t.Fatalf("ApplyPathRenames() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "my-app.txt")); err != nil {
		t.Error("expected my-app.txt to exist")
	}
	if _, err := os.Stat(filepath.Join(dir, "projectname.txt")); err == nil {
		t.Error("projectname.txt should not exist after rename")
	}

	if _, err := os.Stat(filepath.Join(dir, "my-app", "sub", "file.go")); err != nil {
		t.Error("expected my-app/sub/file.go to exist")
	}
}

func TestApplyPathRenamesEmptyExpression(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "placeholder.txt"), "content")

	rc := NewResolveContext()
	rc.Slots["Empty"] = ""

	renames := []manifest.PathRename{
		{Pattern: "placeholder", Expression: "[[ .slots.Empty ]]"},
	}

	if err := ApplyPathRenames(dir, renames, rc); err != nil {
		t.Fatalf("ApplyPathRenames() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "placeholder.txt")); err != nil {
		t.Error("file should still exist when replacement is empty")
	}
}

func TestRenderTemplateFiles(t *testing.T) {
	templateDir := t.TempDir()
	projectDir := t.TempDir()

	writeTestFile(t, filepath.Join(templateDir, "docker-compose.tmpl"), "services:\n  [[ .slots.ServiceName ]]:\n    image: [[ .slots.Image ]]")

	rc := NewResolveContext()
	rc.Slots["ServiceName"] = "web"
	rc.Slots["Image"] = "nginx:latest"

	files := []manifest.TemplateFile{
		{Source: "docker-compose.tmpl", Destination: "docker-compose.yaml"},
	}

	if err := RenderTemplateFiles(templateDir, projectDir, files, rc); err != nil {
		t.Fatalf("RenderTemplateFiles() error: %v", err)
	}

	got := readTestFile(t, filepath.Join(projectDir, "docker-compose.yaml"))
	expected := "services:\n  web:\n    image: nginx:latest"
	if got != expected {
		t.Errorf("rendered =\n%s\nwant:\n%s", got, expected)
	}
}

func TestRenderTemplateFilesMissingSource(t *testing.T) {
	templateDir := t.TempDir()
	projectDir := t.TempDir()

	rc := NewResolveContext()
	files := []manifest.TemplateFile{
		{Source: "nonexistent.tmpl", Destination: "out.yaml"},
	}

	err := RenderTemplateFiles(templateDir, projectDir, files, rc)
	if err == nil {
		t.Error("expected error for missing source file")
	}
}

func TestRenderTemplateFilesCreatesSubdirectories(t *testing.T) {
	templateDir := t.TempDir()
	projectDir := t.TempDir()

	writeTestFile(t, filepath.Join(templateDir, "config.tmpl"), "key: [[ .slots.Value ]]")

	rc := NewResolveContext()
	rc.Slots["Value"] = "test"

	files := []manifest.TemplateFile{
		{Source: "config.tmpl", Destination: "deep/nested/dir/config.yaml"},
	}

	if err := RenderTemplateFiles(templateDir, projectDir, files, rc); err != nil {
		t.Fatalf("RenderTemplateFiles() error: %v", err)
	}

	got := readTestFile(t, filepath.Join(projectDir, "deep", "nested", "dir", "config.yaml"))
	if got != "key: test" {
		t.Errorf("rendered = %q", got)
	}
}

func TestEvaluatePostHooksRemovePaths(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "to-remove.txt"), "content")
	writeTestFile(t, filepath.Join(dir, "keep.txt"), "content")

	rc := NewResolveContext()
	hooks := []manifest.PostHook{
		{RemovePaths: []string{"to-remove.txt"}},
	}

	if err := EvaluatePostHooks(dir, hooks, rc); err != nil {
		t.Fatalf("EvaluatePostHooks() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "to-remove.txt")); err == nil {
		t.Error("to-remove.txt should have been deleted")
	}
	if _, err := os.Stat(filepath.Join(dir, "keep.txt")); err != nil {
		t.Error("keep.txt should still exist")
	}
}

func TestEvaluatePostHooksWithCondition(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "conditional.txt"), "content")

	rc := NewResolveContext()
	rc.Slots["RemoveIt"] = "false"

	hooks := []manifest.PostHook{
		{
			Condition:   "[[ .slots.RemoveIt ]]",
			RemovePaths: []string{"conditional.txt"},
		},
	}

	if err := EvaluatePostHooks(dir, hooks, rc); err != nil {
		t.Fatalf("EvaluatePostHooks() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "conditional.txt")); err != nil {
		t.Error("file should not be removed when condition is false")
	}

	rc.Slots["RemoveIt"] = "true"
	if err := EvaluatePostHooks(dir, hooks, rc); err != nil {
		t.Fatalf("EvaluatePostHooks() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "conditional.txt")); err == nil {
		t.Error("file should be removed when condition is true")
	}
}

func TestEvaluatePostHooksEmptyConditionAlwaysRuns(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, filepath.Join(dir, "file.txt"), "content")

	rc := NewResolveContext()
	hooks := []manifest.PostHook{
		{Condition: "", RemovePaths: []string{"file.txt"}},
	}

	if err := EvaluatePostHooks(dir, hooks, rc); err != nil {
		t.Fatalf("EvaluatePostHooks() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "file.txt")); err == nil {
		t.Error("file should be removed when condition is empty (always runs)")
	}
}

// --- helpers ---

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}
