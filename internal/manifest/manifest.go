package manifest

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/schaemi85/spire/internal/metadata"
	"gopkg.in/yaml.v3"
)

const FilePath = ".spire/manifest.yaml"

// LoadManifest reads the manifest from the current working directory.
// If the file does not exist, it creates a minimal default one.
func LoadManifest() (*SpireManifest, error) {
	fmt.Printf("\nLoading Spire Manifest from %s...\n", FilePath)
	manifest := &SpireManifest{}
	if _, err := os.Stat(FilePath); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("⚠️ Spire Manifest is missing let's create it\n")
		return createManifest()
	}

	yamlData, err := os.ReadFile(FilePath)
	if err != nil {
		return nil, fmt.Errorf("an error occurred while reading file %s: %w", FilePath, err)
	}
	if err = yaml.Unmarshal(yamlData, manifest); err != nil {
		return manifest, fmt.Errorf("failed to unmarshal Spire Manifest from YAML: %w", err)
	}

	fmt.Printf("Spire Manifest loaded successfully\n")
	return manifest, nil
}

// LoadManifestFrom reads a SpireManifest from an explicit path.
func LoadManifestFrom(path string) (*SpireManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest from %s: %w", path, err)
	}
	m := &SpireManifest{}
	if err := yaml.Unmarshal(data, m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest from %s: %w", path, err)
	}
	return m, nil
}

// SaveManifest marshals the manifest to YAML and writes it to FilePath.
// dst, when non-empty, prefixes the path (used to save into a template output directory).
func SaveManifest(m *SpireManifest, dst string) error {
	fmt.Printf("\nSaving Spire Manifest to %s...\n", FilePath)
	destination := FilePath
	if dst != "" {
		destination = filepath.Join(dst, FilePath)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	yamlData, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest to YAML: %w", err)
	}
	if err := os.WriteFile(destination, yamlData, 0755); err != nil {
		return fmt.Errorf("failed to write YAML file: %w", err)
	}
	fmt.Printf("Spire Manifest saved successfully\n")
	return nil
}

func createManifest() (*SpireManifest, error) {
	var repoUrl string
	fmt.Println("\nSearching for Git Remote Origin...")
	if repo, err := git.PlainOpen("."); err == nil {
		if remote, err := repo.Remote("origin"); err == nil {
			repoUrl = remote.Config().URLs[0]
		} else {
			fmt.Printf("❌ Failed to read repository origin: %+v\n", err)
		}
	} else {
		fmt.Printf("❌ Failed to open current Git repository: %+v\n", err)
	}

	fmt.Println("\nSpire Manifest Information")
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	m := &SpireManifest{
		SpireVersion:    metadata.VERSION,
		TemplateVersion: "v0.0.1",
		GitRepository:   repoUrl,
	}
	if err := SaveManifest(m, ""); err != nil {
		return m, fmt.Errorf("❌ could not save Spire Manifest: %v", err)
	}
	fmt.Printf("Spire Manifest created successfully\n")
	return m, nil
}
