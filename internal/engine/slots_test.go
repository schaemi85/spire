package engine

import (
	"testing"

	"github.com/schaemi85/spire/internal/manifest"
)

func TestEvaluateExpression(t *testing.T) {
	rc := NewResolveContext()
	rc.Slots["AppName"] = "My Cool App"
	rc.Slots["Version"] = "1.0.0"

	tests := []struct {
		name     string
		expr     string
		expected string
		wantErr  bool
	}{
		{"simple slot reference", "[[ .slots.AppName ]]", "My Cool App", false},
		{"slot with pipeline", "[[ .slots.AppName | slugify ]]", "my-cool-app", false},
		{"chained pipelines", "[[ .slots.AppName | slugify | upper ]]", "MY-COOL-APP", false},
		{"empty expression", "", "", false},
		{"invalid template syntax", "[[ .slots.AppName | badFunc123 ]]", "", true},
		{"non-existent slot", "[[ .slots.Missing ]]", "<no value>", false},
		{"literal text", "static-value", "static-value", false},
		{"slot with pascalCase", "[[ .slots.AppName | slugify | pascalCase ]]", "MyCoolApp", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expr, rc)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("EvaluateExpression(%q) = %q, want %q", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestApplyPipelines(t *testing.T) {
	rc := NewResolveContext()

	tests := []struct {
		name      string
		value     string
		pipelines []string
		expected  string
		wantErr   bool
	}{
		{"no pipelines", "hello", nil, "hello", false},
		{"empty pipelines", "hello", []string{}, "hello", false},
		{"single pipeline", "Hello World", []string{"slugify"}, "hello-world", false},
		{"multiple pipelines", "Hello World", []string{"slugify", "upper"}, "HELLO-WORLD", false},
		{"trimSpace", "  hello  ", []string{"trimSpace"}, "hello", false},
		{"invalid pipeline", "hello", []string{"nonExistentFunc"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := applyPipelines(tt.value, tt.pipelines, rc)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("applyPipelines(%q, %v) = %q, want %q", tt.value, tt.pipelines, got, tt.expected)
			}
		})
	}
}

func TestResolveSlotsNonInteractiveDynamic(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "AppName", Type: manifest.PromptMandatory},
		{Key: "Slug", Type: manifest.DynamicValue, Expression: "[[ .slots.AppName | slugify ]]"},
		{Key: "Pascal", Type: manifest.DynamicValue, Expression: "[[ .slots.Slug | pascalCase ]]"},
	}

	rc := NewResolveContext()
	rc.Slots["AppName"] = "My Cool App"

	err := ResolveSlots(slots, rc, true)
	if err != nil {
		t.Fatalf("ResolveSlots() error: %v", err)
	}

	if slots[0].Value != "My Cool App" {
		t.Errorf("AppName = %q, want %q", slots[0].Value, "My Cool App")
	}
	if slots[1].Value != "my-cool-app" {
		t.Errorf("Slug = %q, want %q", slots[1].Value, "my-cool-app")
	}
	if slots[2].Value != "MyCoolApp" {
		t.Errorf("Pascal = %q, want %q", slots[2].Value, "MyCoolApp")
	}
}

func TestResolveSlotsNonInteractiveMissingMandatory(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "AppName", Type: manifest.PromptMandatory},
	}
	rc := NewResolveContext()

	err := ResolveSlots(slots, rc, true)
	if err == nil {
		t.Error("expected error for missing mandatory slot in non-interactive mode")
	}
}

func TestResolveSlotsNonInteractiveMissingSecret(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Token", Type: manifest.PromptSecret},
	}
	rc := NewResolveContext()

	err := ResolveSlots(slots, rc, true)
	if err == nil {
		t.Error("expected error for missing secret slot in non-interactive mode")
	}
}

func TestResolveSlotsNonInteractiveOptionalDefault(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Port", Type: manifest.PromptOptional, DefaultValue: "8080"},
	}
	rc := NewResolveContext()

	err := ResolveSlots(slots, rc, true)
	if err != nil {
		t.Fatalf("ResolveSlots() error: %v", err)
	}
	if slots[0].Value != "8080" {
		t.Errorf("Port = %q, want %q", slots[0].Value, "8080")
	}
}

func TestResolveSlotsWithPipelines(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Name", Type: manifest.PromptMandatory},
		{Key: "Slug", Type: manifest.DynamicValue, Expression: "[[ .slots.Name | slugify ]]"},
	}
	rc := NewResolveContext()
	rc.Slots["Name"] = "Hello World"

	err := ResolveSlots(slots, rc, true)
	if err != nil {
		t.Fatalf("ResolveSlots() error: %v", err)
	}
	if slots[1].Value != "hello-world" {
		t.Errorf("Slug = %q, want %q", slots[1].Value, "hello-world")
	}
}

func TestResolveSlotsPresetSkipsPrompting(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Name", Type: manifest.PromptMandatory},
		{Key: "Env", Type: manifest.PromptOptional, DefaultValue: "dev"},
	}
	rc := NewResolveContext()
	rc.Slots["Name"] = "preset-value"
	rc.Slots["Env"] = "prod"

	err := ResolveSlots(slots, rc, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if slots[0].Value != "preset-value" {
		t.Errorf("Name = %q, want %q", slots[0].Value, "preset-value")
	}
	if slots[1].Value != "prod" {
		t.Errorf("Env = %q, want %q", slots[1].Value, "prod")
	}
}

func TestResolveDynamicSlots(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Name", Type: manifest.PromptMandatory, Value: "My App"},
		{Key: "Slug", Type: manifest.DynamicValue, Expression: "[[ .slots.Name | slugify ]]"},
	}
	rc := NewResolveContext()
	rc.Slots["Name"] = "Updated App"

	err := ResolveDynamicSlots(slots, rc)
	if err != nil {
		t.Fatalf("ResolveDynamicSlots() error: %v", err)
	}
	if slots[0].Value != "My App" {
		t.Errorf("non-dynamic slot changed: %q", slots[0].Value)
	}
	if slots[1].Value != "updated-app" {
		t.Errorf("Slug = %q, want %q", slots[1].Value, "updated-app")
	}
}

func TestResolveContextTemplateData(t *testing.T) {
	rc := NewResolveContext()
	rc.Slots["key"] = "value"
	rc.Services = []manifest.Service{{Name: "svc1"}}

	data := rc.templateData()

	slots, ok := data["slots"].(map[string]string)
	if !ok {
		t.Fatal("templateData missing slots map")
	}
	if slots["key"] != "value" {
		t.Errorf("slots[key] = %q", slots["key"])
	}

	services, ok := data["services"].([]manifest.Service)
	if !ok {
		t.Fatal("templateData missing services slice")
	}
	if len(services) != 1 {
		t.Errorf("services length = %d", len(services))
	}
}

func TestResolveSlotsUnknownType(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Bad", Type: manifest.SlotType(99)},
	}
	rc := NewResolveContext()

	err := ResolveSlots(slots, rc, true)
	if err == nil {
		t.Error("expected error for unknown slot type")
	}
}
