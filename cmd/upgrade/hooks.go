package upgrade

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/schaemi85/spire/cmd/service"
	"github.com/schaemi85/spire/internal/engine"
	"github.com/schaemi85/spire/internal/manifest"
)

// runPostUpgradeHooks executes post-upgrade tasks to ensure the project is in a good state.
func runPostUpgradeHooks(TemplateVersion string) ([]string, error) {
	var warnings []string
	fmt.Println()
	fmt.Println("🔧 Running post-upgrade hooks...")
	fmt.Println()

	appManifest, err := manifest.LoadManifest()
	if err != nil {
		return nil, err
	}
	cloneDir, _ := os.Getwd()
	fmt.Println("Customizing application to match with your manifest...")

	rc := engine.BuildResolveContextFromManifest(appManifest)

	if err := engine.RenderProjectFiles(cloneDir, rc, appManifest.IgnorePaths); err != nil {
		return nil, fmt.Errorf("failed to render project files: %w", err)
	}
	if len(appManifest.PathRenames) > 0 {
		if err := engine.ApplyPathRenames(cloneDir, appManifest.PathRenames, rc); err != nil {
			return nil, fmt.Errorf("failed to apply path renames: %w", err)
		}
	}
	fmt.Println("Customizing services to match with your manifest...")
	entries, err := os.ReadDir(ServicesPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read services directory: %w", err)
	}

	for _, e := range entries {
		if !service.IsValidServiceDir(ServicesPath, e) {
			if e.IsDir() {
				fmt.Printf("  ⚠️  Skipping %s, go module not found\n", e.Name())
			}
			continue
		}

		sPath := filepath.Join(ServicesPath, e.Name())

		serviceRC := engine.NewResolveContext()
		for k, v := range rc.Slots {
			serviceRC.Slots[k] = v
		}
		serviceRC.Slots["ServiceName"] = e.Name()

		if err := engine.RenderProjectFiles(sPath, serviceRC, appManifest.IgnorePaths); err != nil {
			return nil, fmt.Errorf("failed to render service files for %s: %w", e.Name(), err)
		}
		if len(appManifest.ServiceConfig.PathRenames) > 0 {
			if err := engine.ApplyPathRenames(sPath, appManifest.ServiceConfig.PathRenames, serviceRC); err != nil {
				return nil, fmt.Errorf("failed to apply path renames for service %s: %w", e.Name(), err)
			}
		}
	}

	fmt.Println("Updating Spire Manifest with new scaffolding version...")
	appManifest.TemplateVersion = TemplateVersion
	if err := manifest.SaveManifest(appManifest, ""); err != nil {
		return nil, fmt.Errorf("failed to save Spire Manifest: %w", err)
	}

	for _, task := range []struct{ name, arg string }{
		{"🧹", "clean"},
		{"🔧", "proto:gen"},
		{"", "build"},
		{"", "tidy"},
		{"🎨", "format"},
	} {
		fmt.Printf("\n%s Running 'task %s'...\n", task.name, task.arg)
		cmd := exec.Command("task", task.arg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			warnMsg := fmt.Sprintf("task %s failed: %v", task.arg, err)
			fmt.Printf("  ⚠️  Warning: %s\n", warnMsg)
			warnings = append(warnings, warnMsg)
		}
	}

	fmt.Println()
	if len(warnings) > 0 {
		fmt.Println("Post-upgrade hooks completed with warnings")
	} else {
		fmt.Println("Post-upgrade hooks completed")
	}
	return warnings, nil
}
