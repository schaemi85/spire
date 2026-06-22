package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/engine"
	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/plugin"
	"github.com/schaemi85/spire/internal/tools"

	"github.com/spf13/cobra"
)

// AddService contains the main logic for adding a service. Extracted for testability.
// setValues holds pre-seeded service slot values in "Key=Value" form (from --set flags).
func AddService(nonInteractive bool, setValues []string) error {
	m, err := manifest.LoadManifest()
	if err != nil {
		return err
	}
	if m.ServiceConfig.OriginalPath == "" {
		return fmt.Errorf("❌ serviceConfig.originalPath is not set in the manifest")
	}
	if _, err := os.Stat(m.ServiceConfig.OriginalPath); err != nil {
		return fmt.Errorf("❌ service blueprint directory not found: %s", m.ServiceConfig.OriginalPath)
	}
	if len(m.ServiceConfig.ServicesSlots) == 0 {
		return fmt.Errorf("❌ No service slots defined in manifest.\n   The manifest must have serviceConfig.servicesSlots defined")
	}

	rc := engine.BuildResolveContextFromManifest(m)
	for _, sv := range setValues {
		parts := strings.SplitN(sv, "=", 2)
		if len(parts) == 2 {
			rc.Slots[parts[0]] = parts[1]
		}
	}

	slotsClone := make([]manifest.Slot, len(m.ServiceConfig.ServicesSlots))
	copy(slotsClone, m.ServiceConfig.ServicesSlots)
	if err := engine.ResolveSlots(slotsClone, rc, nonInteractive); err != nil {
		return fmt.Errorf("failed to resolve service slots: %w", err)
	}

	name := rc.Slots["ServiceName"]
	if name == "" {
		return fmt.Errorf("❌ ServiceName slot was not resolved")
	}

	workDir, _ := os.Getwd()
	currentSvc := buildServiceInfo(name, name, slotsClone)
	hookCtx := buildHookContext(m, rc, workDir, &currentSvc)

	fmt.Println("\nRunning before-add-service plugins...")
	if err := plugin.RunHook(plugin.HookBeforeAddService, hookCtx); err != nil {
		return fmt.Errorf("before-add-service plugin failed: %w", err)
	}

	templateDir := m.ServiceConfig.OriginalPath
	targetDir := filepath.Join("services", name)

	fmt.Printf("\n🔧 Creating new service: '%s'\n", name)
	fmt.Println()

	fmt.Println("Copying service template...")
	if err := tools.CopyDir(templateDir, targetDir); err != nil {
		return fmt.Errorf("❌ error copying template: %w", err)
	}
	fmt.Println("Template copied successfully")

	fmt.Println("\nCustomizing service files...")
	if err := engine.RenderProjectFiles(targetDir, rc, m.IgnorePaths); err != nil {
		return fmt.Errorf("failed to render service files: %w", err)
	}

	if len(m.ServiceConfig.PathRenames) > 0 {
		if err := engine.ApplyPathRenames(targetDir, m.ServiceConfig.PathRenames, rc); err != nil {
			return fmt.Errorf("failed to apply service path renames: %w", err)
		}
	}

	if len(m.ServiceConfig.PostHooks) > 0 {
		if err := engine.EvaluatePostHooks(targetDir, m.ServiceConfig.PostHooks, rc); err != nil {
			return fmt.Errorf("failed to evaluate post-hooks: %w", err)
		}
	}
	fmt.Println("Service customized successfully")

	svc := manifest.Service{
		Name:     name,
		SlugName: name,
		Slots:    slotsClone,
	}
	m.Services = append(m.Services, svc)
	engine.ClearSecretSlotValues(slotsClone)
	if err := manifest.SaveManifest(m, ""); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	appRC := engine.BuildResolveContextFromManifest(m)
	var serviceChangeFiles []manifest.TemplateFile
	for _, tf := range m.TemplateFiles {
		if tf.RegenerateOnServiceChange {
			serviceChangeFiles = append(serviceChangeFiles, tf)
		}
	}
	if len(serviceChangeFiles) > 0 {
		projectDir, _ := os.Getwd()
		fmt.Println("\nRegenerating app-level template files...")
		if err := engine.RenderTemplateFiles(projectDir, projectDir, serviceChangeFiles, appRC); err != nil {
			return fmt.Errorf("failed to regenerate app-level template files: %w", err)
		}
		fmt.Println("App-level template files updated")
	}

	hookCtx.Services = append(hookCtx.Services, currentSvc)
	fmt.Println("\nRunning after-add-service plugins...")
	if err := plugin.RunHook(plugin.HookAfterAddService, hookCtx); err != nil {
		return fmt.Errorf("after-add-service plugin failed: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Printf("Service '%s' created successfully!\n", name)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n📁 Location: %s\n", targetDir)
	absPath, _ := filepath.Abs(targetDir)
	fmt.Printf("   Absolute path: %s\n", absPath)
	fmt.Println(strings.Repeat("-", 60))
	fmt.Println("\nHappy coding!")

	return nil
}

func buildServiceInfo(name, slugName string, slots []manifest.Slot) plugin.ServiceInfo {
	svcSlots := make(map[string]string, len(slots))
	for _, s := range slots {
		svcSlots[s.Key] = s.Value
	}
	return plugin.ServiceInfo{Name: name, SlugName: slugName, Slots: svcSlots}
}

func buildHookContext(m *manifest.SpireManifest, rc *engine.ResolveContext, workDir string, current *plugin.ServiceInfo) plugin.HookContext {
	appSlots := make(map[string]string, len(m.AppSlots))
	for _, s := range m.AppSlots {
		appSlots[s.Key] = s.Value
	}
	for k, v := range rc.Slots {
		appSlots[k] = v
	}
	services := make([]plugin.ServiceInfo, 0, len(m.Services))
	for _, svc := range m.Services {
		services = append(services, buildServiceInfo(svc.Name, svc.SlugName, svc.Slots))
	}
	return plugin.HookContext{
		WorkDir:        workDir,
		Slots:          appSlots,
		Services:       services,
		CurrentService: current,
	}
}

var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new service from the scaffolding template",
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)
		nint, _ := cmd.Root().Flags().GetBool("non-interactive")
		setValues, _ := cmd.Flags().GetStringArray("set")
		if err := AddService(nint, setValues); err != nil {
			fmt.Printf("\n%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	AddCmd.Flags().StringArray("set", nil, "Set service slot values (repeatable): --set Key=Value")
}
