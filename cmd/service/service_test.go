package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsValidServiceDir(t *testing.T) {
	dir := t.TempDir()

	// Valid service: directory with go.mod
	validSvc := filepath.Join(dir, "user-service")
	os.MkdirAll(validSvc, 0755)
	os.WriteFile(filepath.Join(validSvc, "go.mod"), []byte("module example/user-service"), 0644)

	// Invalid service: directory without go.mod
	noMod := filepath.Join(dir, "no-mod-svc")
	os.MkdirAll(noMod, 0755)

	// File (not a directory)
	os.WriteFile(filepath.Join(dir, "not-a-dir.txt"), []byte("content"), 0644)

	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		valid := IsValidServiceDir(dir, entry)
		switch entry.Name() {
		case "user-service":
			if !valid {
				t.Error("user-service should be valid (has go.mod)")
			}
		case "no-mod-svc":
			if valid {
				t.Error("no-mod-svc should be invalid (no go.mod)")
			}
		case "not-a-dir.txt":
			if valid {
				t.Error("not-a-dir.txt should be invalid (not a directory)")
			}
		}
	}
}

func TestReadServiceConfig(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantDB  bool
		wantAPI bool
		wantJob bool
		wantNil bool
	}{
		{
			name:    "all features enabled",
			yaml:    "db: true\napi: true\nconfigure_as_job: true\n",
			wantDB:  true,
			wantAPI: true,
			wantJob: true,
		},
		{
			name:    "only db",
			yaml:    "db: true\n",
			wantDB:  true,
			wantAPI: false,
			wantJob: false,
		},
		{
			name:    "only api",
			yaml:    "api: true\n",
			wantDB:  false,
			wantAPI: true,
			wantJob: false,
		},
		{
			name:    "only job",
			yaml:    "configure_as_job: true\n",
			wantDB:  false,
			wantAPI: false,
			wantJob: true,
		},
		{
			name:    "no features",
			yaml:    "name: minimal\n",
			wantDB:  false,
			wantAPI: false,
			wantJob: false,
		},
		{
			name:    "empty config",
			yaml:    "",
			wantDB:  false,
			wantAPI: false,
			wantJob: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			svcDir := filepath.Join(dir, "my-service")
			os.MkdirAll(svcDir, 0755)
			os.WriteFile(filepath.Join(svcDir, "config.yaml"), []byte(tt.yaml), 0644)

			svc := ReadServiceConfig(svcDir)

			if tt.wantNil {
				if svc != nil {
					t.Errorf("expected nil, got %+v", svc)
				}
				return
			}
			if svc == nil {
				t.Fatal("expected non-nil service config")
			}
			if svc.Name != "my-service" {
				t.Errorf("Name = %q, want %q", svc.Name, "my-service")
			}
			if svc.WithDB != tt.wantDB {
				t.Errorf("WithDB = %v, want %v", svc.WithDB, tt.wantDB)
			}
			if svc.WithAPI != tt.wantAPI {
				t.Errorf("WithAPI = %v, want %v", svc.WithAPI, tt.wantAPI)
			}
			if svc.WithJob != tt.wantJob {
				t.Errorf("WithJob = %v, want %v", svc.WithJob, tt.wantJob)
			}
		})
	}
}

func TestReadServiceConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	svc := ReadServiceConfig(filepath.Join(dir, "nonexistent"))
	if svc != nil {
		t.Errorf("expected nil for missing config, got %+v", svc)
	}
}

func TestReadServiceConfigInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	svcDir := filepath.Join(dir, "bad-service")
	os.MkdirAll(svcDir, 0755)
	os.WriteFile(filepath.Join(svcDir, "config.yaml"), []byte(":::invalid\n{{{"), 0644)

	svc := ReadServiceConfig(svcDir)
	if svc != nil {
		t.Errorf("expected nil for invalid YAML, got %+v", svc)
	}
}

func TestReadServicesConfigs(t *testing.T) {
	// Save and restore working directory
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)

	servicesDir := filepath.Join(dir, "services")
	os.MkdirAll(servicesDir, 0755)

	// Valid service with config
	svc1 := filepath.Join(servicesDir, "svc1")
	os.MkdirAll(svc1, 0755)
	os.WriteFile(filepath.Join(svc1, "go.mod"), []byte("module svc1"), 0644)
	os.WriteFile(filepath.Join(svc1, "config.yaml"), []byte("db: true\napi: true\n"), 0644)

	// Valid service without config.yaml (ReadServiceConfig returns nil)
	svc2 := filepath.Join(servicesDir, "svc2")
	os.MkdirAll(svc2, 0755)
	os.WriteFile(filepath.Join(svc2, "go.mod"), []byte("module svc2"), 0644)

	// Invalid: file, not a directory
	os.WriteFile(filepath.Join(servicesDir, "not-a-dir"), []byte("content"), 0644)

	// Invalid: directory without go.mod
	noMod := filepath.Join(servicesDir, "no-mod")
	os.MkdirAll(noMod, 0755)

	result, err := ReadServicesConfigs()
	if err != nil {
		t.Fatalf("ReadServicesConfigs() error: %v", err)
	}

	// Only svc1 should be in results (svc2 has no config.yaml so ReadServiceConfig returns nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 service, got %d", len(result))
	}
	if result[0].Name != "svc1" {
		t.Errorf("service name = %q, want %q", result[0].Name, "svc1")
	}
	if !result[0].WithDB || !result[0].WithAPI {
		t.Errorf("service flags incorrect: %+v", result[0])
	}
}

func TestAddServiceNonInteractiveWithSet(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)

	// Service blueprint: a directory with a file referencing the ServiceName slot.
	blueprint := filepath.Join(dir, "services", "sampleservice")
	os.MkdirAll(blueprint, 0755)
	os.WriteFile(filepath.Join(blueprint, "main.go"),
		[]byte("package main\n\n// service: [[ .slots.ServiceName ]]\n"), 0644)

	// Minimal manifest with a single mandatory service slot.
	manifestYAML := `spireVersion: v0.0.1
templateVersion: v1.0.0
serviceConfig:
  originalPath: services/sampleservice
  servicesSlots:
    - key: ServiceName
      label: Service Name
      type: PromptMandatory
`
	os.MkdirAll(filepath.Join(dir, ".spire"), 0755)
	if err := os.WriteFile(filepath.Join(dir, ".spire", "manifest.yaml"), []byte(manifestYAML), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	// Non-interactive: the value must come from --set, not a prompt.
	if err := AddService(true, []string{"ServiceName=payments"}); err != nil {
		t.Fatalf("AddService() error: %v", err)
	}

	// The service directory should have been created and rendered.
	rendered, err := os.ReadFile(filepath.Join(dir, "services", "payments", "main.go"))
	if err != nil {
		t.Fatalf("expected rendered service file: %v", err)
	}
	if got := string(rendered); !strings.Contains(got, "service: payments") {
		t.Errorf("rendered file = %q, want it to contain %q", got, "service: payments")
	}
}

func TestAddServiceNonInteractiveMissingSetFails(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)

	blueprint := filepath.Join(dir, "services", "sampleservice")
	os.MkdirAll(blueprint, 0755)
	os.WriteFile(filepath.Join(blueprint, "main.go"), []byte("package main\n"), 0644)

	manifestYAML := `spireVersion: v0.0.1
templateVersion: v1.0.0
serviceConfig:
  originalPath: services/sampleservice
  servicesSlots:
    - key: ServiceName
      type: PromptMandatory
`
	os.MkdirAll(filepath.Join(dir, ".spire"), 0755)
	os.WriteFile(filepath.Join(dir, ".spire", "manifest.yaml"), []byte(manifestYAML), 0644)

	// No --set value for a mandatory slot in non-interactive mode must fail.
	if err := AddService(true, nil); err == nil {
		t.Error("expected error when a mandatory slot has no --set value in non-interactive mode")
	}
}

func TestReadServicesConfigsMissingDir(t *testing.T) {
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	dir := t.TempDir()
	os.Chdir(dir)
	// No "services" directory

	_, err := ReadServicesConfigs()
	if err == nil {
		t.Error("expected error for missing services directory")
	}
}
