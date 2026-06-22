package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetOrRemoveYAMLKeySetValue(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	writeYAML(t, f, "name: old\nversion: 1.0\n")

	newVal := "new-name"
	if err := SetOrRemoveYAMLKey(f, "name", &newVal); err != nil {
		t.Fatalf("SetOrRemoveYAMLKey() error: %v", err)
	}

	content := readFileContent(t, f)
	if !containsString(content, "new-name") {
		t.Errorf("expected 'new-name' in output, got:\n%s", content)
	}
}

func TestSetOrRemoveYAMLKeyRemoveValue(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	writeYAML(t, f, "name: value\nkeep: yes\n")

	if err := SetOrRemoveYAMLKey(f, "name", nil); err != nil {
		t.Fatalf("SetOrRemoveYAMLKey() error: %v", err)
	}

	content := readFileContent(t, f)
	if containsString(content, "name:") {
		t.Errorf("key 'name' should be removed, got:\n%s", content)
	}
	if !containsString(content, "keep:") {
		t.Errorf("'keep' key should still exist, got:\n%s", content)
	}
}

func TestSetOrRemoveYAMLKeyNestedKey(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	writeYAML(t, f, "parent:\n  child: old\n  other: keep\n")

	newVal := "updated"
	if err := SetOrRemoveYAMLKey(f, "parent.child", &newVal); err != nil {
		t.Fatalf("SetOrRemoveYAMLKey() error: %v", err)
	}

	content := readFileContent(t, f)
	if !containsString(content, "updated") {
		t.Errorf("expected 'updated' in output, got:\n%s", content)
	}
	if !containsString(content, "other:") {
		t.Errorf("sibling key should still exist, got:\n%s", content)
	}
}

func TestSetOrRemoveYAMLKeyRemoveNestedKey(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "config.yaml")
	writeYAML(t, f, "parent:\n  child: value\n  sibling: keep\n")

	if err := SetOrRemoveYAMLKey(f, "parent.child", nil); err != nil {
		t.Fatalf("SetOrRemoveYAMLKey() error: %v", err)
	}

	content := readFileContent(t, f)
	if containsString(content, "child:") {
		t.Errorf("nested key should be removed, got:\n%s", content)
	}
	if !containsString(content, "sibling:") {
		t.Errorf("sibling should still exist, got:\n%s", content)
	}
}

func TestSetOrRemoveYAMLKeyMissingFile(t *testing.T) {
	newVal := "value"
	err := SetOrRemoveYAMLKey("/nonexistent/file.yaml", "key", &newVal)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestSetOrRemoveYAMLKeyInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.yaml")
	// Use truly invalid YAML that cannot be parsed
	writeYAML(t, f, "key: [unclosed\n  - mixed: {nope")

	newVal := "value"
	err := SetOrRemoveYAMLKey(f, "key", &newVal)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestSetOrRemoveYAMLKeyDeeplyNested(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "deep.yaml")
	writeYAML(t, f, "a:\n  b:\n    c: deep-value\n")

	newVal := "new-deep"
	if err := SetOrRemoveYAMLKey(f, "a.b.c", &newVal); err != nil {
		t.Fatalf("SetOrRemoveYAMLKey() error: %v", err)
	}

	content := readFileContent(t, f)
	if !containsString(content, "new-deep") {
		t.Errorf("expected 'new-deep', got:\n%s", content)
	}
}

// --- helpers ---

func writeYAML(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
