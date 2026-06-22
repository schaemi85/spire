package upgrade

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type ReplaceItem struct {
	Path   string   `yaml:"path"`
	Keep   []string `yaml:"keep,omitempty"`
	Ignore []string `yaml:"ignore,omitempty"`
}

type UpgradeManifest struct {
	Application []ReplaceItem `yaml:"application"`
	Services    []ReplaceItem `yaml:"services"`
}

const (
	FilePath = ".spire/upgrade-manifest.yaml"
)

func createUpgradeManifest() (*UpgradeManifest, error) {
	fmt.Printf("\n📒 Creating default Upgrade Manifest to %s...\n", FilePath)
	upgradeManifest := &UpgradeManifest{
		Application: []ReplaceItem{
			{Path: ".devcontainer", Keep: []string{"postgres/init-user-db.sh", ".env"}},
			{Path: ".vscode"},
			{Path: "infra/tofu/azure", Keep: []string{"environments/**", "**/custom_*.tf"}},
			{Path: "infra/tofu/tufin", Keep: []string{"environments/**", "**/custom_*.tf"}},
			{Path: "pkg"},
			{Path: "templates"},
			{Path: ".dockerignore"},
			{Path: ".gitattributes"},
			{Path: ".gitignore"},
			{Path: ".gitlab-ci.cloud.yml"},
			{Path: ".gitlab-ci.onprem.yml"},
			{Path: ".gitlab-ci.yml"},
			{Path: ".golangci.yml"},
			{Path: "Taskfile.yml"},
		},
		Services: []ReplaceItem{
			{Path: "_apigw"},
			{Path: "_azure"},
			{Path: "_k8s"},
			{Path: "_ci"},
			{Path: "_e2e"},
		},
	}
	// Ensure directory exists
	dir := filepath.Dir(FilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(upgradeManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal configs to YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(FilePath, yamlData, 0755); err != nil {
		return nil, fmt.Errorf("failed to write YAML file: %w", err)
	}
	fmt.Printf("Spire Manifest saved successfully\n")
	return upgradeManifest, nil
}

func loadUpgradeManifest() (*UpgradeManifest, error) {
	fmt.Printf("\nLoading configuration from %s...\n", FilePath)
	var manifest UpgradeManifest
	// Ensure file exists
	if _, err := os.Stat(FilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("⚠️ Application Upgrade Manifest is missing let's create a default one\n")
		return createUpgradeManifest()
	}
	yamlData, err := os.ReadFile(FilePath)
	if err != nil {
		return &manifest, fmt.Errorf("an error occurred while reading file %s: %w", FilePath, err)
	}
	// Unmarshal from YAML
	err = yaml.Unmarshal(yamlData, &manifest)
	if err != nil {
		return &manifest, fmt.Errorf("failed to unmarshal Upgrade Manifest from YAML: %w", err)
	}

	fmt.Printf("Upgrade Manifest loaded successfully\n")
	return &manifest, nil
}
