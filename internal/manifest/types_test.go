package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSlotTypeMarshalYAML(t *testing.T) {
	tests := []struct {
		st       SlotType
		expected string
	}{
		{PromptOptional, "PromptOptional"},
		{PromptMandatory, "PromptMandatory"},
		{PromptSecret, "PromptSecret"},
		{DynamicValue, "DynamicValue"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got, err := tt.st.MarshalYAML()
			if err != nil {
				t.Fatalf("MarshalYAML() error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("MarshalYAML() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSlotTypeMarshalYAMLUnknown(t *testing.T) {
	unknown := SlotType(99)
	_, err := unknown.MarshalYAML()
	if err == nil {
		t.Error("expected error for unknown SlotType, got nil")
	}
}

func TestSlotTypeUnmarshalYAML(t *testing.T) {
	tests := []struct {
		input    string
		expected SlotType
	}{
		{"PromptOptional", PromptOptional},
		{"PromptMandatory", PromptMandatory},
		{"PromptSecret", PromptSecret},
		{"DynamicValue", DynamicValue},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var st SlotType
			err := yaml.Unmarshal([]byte(tt.input), &st)
			if err != nil {
				t.Fatalf("UnmarshalYAML(%q) error: %v", tt.input, err)
			}
			if st != tt.expected {
				t.Errorf("UnmarshalYAML(%q) = %v, want %v", tt.input, st, tt.expected)
			}
		})
	}
}

func TestSlotTypeUnmarshalYAMLUnknown(t *testing.T) {
	var st SlotType
	err := yaml.Unmarshal([]byte("InvalidType"), &st)
	if err == nil {
		t.Error("expected error for unknown SlotType string, got nil")
	}
}

func TestSlotTypeString(t *testing.T) {
	if PromptOptional.String() != "PromptOptional" {
		t.Errorf("PromptOptional.String() = %q", PromptOptional.String())
	}
	unknown := SlotType(99)
	if unknown.String() != "Unknown" {
		t.Errorf("SlotType(99).String() = %q, want %q", unknown.String(), "Unknown")
	}
}

func TestGetSlotValue(t *testing.T) {
	m := &SpireManifest{
		AppSlots: []Slot{
			{Key: "ProjectName", Value: "my-app"},
			{Key: "Version", Value: "1.0.0"},
		},
	}

	if v := m.GetSlotValue("ProjectName"); v != "my-app" {
		t.Errorf("GetSlotValue(ProjectName) = %q, want %q", v, "my-app")
	}
	if v := m.GetSlotValue("Version"); v != "1.0.0" {
		t.Errorf("GetSlotValue(Version) = %q, want %q", v, "1.0.0")
	}
	if v := m.GetSlotValue("NonExistent"); v != "" {
		t.Errorf("GetSlotValue(NonExistent) = %q, want empty", v)
	}
}

func TestSetSlotValue(t *testing.T) {
	m := &SpireManifest{
		AppSlots: []Slot{
			{Key: "ProjectName", Value: "old-name"},
		},
	}

	if ok := m.SetSlotValue("ProjectName", "new-name"); !ok {
		t.Error("SetSlotValue returned false for existing slot")
	}
	if m.AppSlots[0].Value != "new-name" {
		t.Errorf("slot value = %q, want %q", m.AppSlots[0].Value, "new-name")
	}
	if ok := m.SetSlotValue("NonExistent", "value"); ok {
		t.Error("SetSlotValue returned true for non-existent slot")
	}
}

func TestManifestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	original := &SpireManifest{
		SpireVersion:    "1.0.0",
		TemplateVersion: "v2.1.0",
		GitRepository:   "https://example.com/repo.git",
		AppSlots: []Slot{
			{Key: "AppName", Label: "Application Name", Type: PromptMandatory, Value: "my-app"},
			{Key: "Slug", Type: DynamicValue, Expression: "[[ .slots.AppName | slugify ]]", Value: "my-app"},
		},
		Services: []Service{
			{Name: "user-service", SlugName: "user-service", Slots: []Slot{
				{Key: "ServiceName", Value: "user-service"},
			}},
		},
		TemplateFiles: []TemplateFile{
			{Source: "tmpl/docker-compose.yaml", Destination: "docker-compose.yaml", RegenerateOnServiceChange: true},
		},
		PathRenames: []PathRename{
			{Pattern: "projectname", Expression: "[[ .slots.Slug ]]"},
		},
		IgnorePaths: []string{".git", "vendor"},
	}

	if err := SaveManifest(original, dir); err != nil {
		t.Fatalf("SaveManifest() error: %v", err)
	}

	manifestPath := filepath.Join(dir, FilePath)
	loaded, err := LoadManifestFrom(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifestFrom() error: %v", err)
	}

	if loaded.SpireVersion != original.SpireVersion {
		t.Errorf("SpireVersion = %q, want %q", loaded.SpireVersion, original.SpireVersion)
	}
	if loaded.TemplateVersion != original.TemplateVersion {
		t.Errorf("TemplateVersion = %q, want %q", loaded.TemplateVersion, original.TemplateVersion)
	}
	if loaded.GitRepository != original.GitRepository {
		t.Errorf("GitRepository = %q, want %q", loaded.GitRepository, original.GitRepository)
	}
	if len(loaded.AppSlots) != len(original.AppSlots) {
		t.Fatalf("AppSlots length = %d, want %d", len(loaded.AppSlots), len(original.AppSlots))
	}
	if loaded.AppSlots[0].Key != "AppName" || loaded.AppSlots[0].Value != "my-app" {
		t.Errorf("AppSlots[0] = %+v", loaded.AppSlots[0])
	}
	if loaded.AppSlots[1].Type != DynamicValue {
		t.Errorf("AppSlots[1].Type = %v, want DynamicValue", loaded.AppSlots[1].Type)
	}
	if len(loaded.Services) != 1 || loaded.Services[0].Name != "user-service" {
		t.Errorf("Services = %+v", loaded.Services)
	}
	if len(loaded.TemplateFiles) != 1 || !loaded.TemplateFiles[0].RegenerateOnServiceChange {
		t.Errorf("TemplateFiles = %+v", loaded.TemplateFiles)
	}
	if len(loaded.PathRenames) != 1 || loaded.PathRenames[0].Pattern != "projectname" {
		t.Errorf("PathRenames = %+v", loaded.PathRenames)
	}
	if len(loaded.IgnorePaths) != 2 {
		t.Errorf("IgnorePaths = %v", loaded.IgnorePaths)
	}
}

func TestLoadManifestFromMissing(t *testing.T) {
	_, err := LoadManifestFrom("/nonexistent/manifest.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadManifestFromInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "bad.yaml")
	os.WriteFile(f, []byte(":::invalid yaml:::{{{\n\t\t[[["), 0644)

	_, err := LoadManifestFrom(f)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}
