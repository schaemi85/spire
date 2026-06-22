package manifest

import (
	"fmt"
	"os"
	"regexp"
	"text/template"

	commoncmds "github.com/schaemi85/spire/cmd/common"
	"github.com/schaemi85/spire/internal/engine"
	mfst "github.com/schaemi85/spire/internal/manifest"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type validationError struct {
	path    string
	message string
}

func (e validationError) String() string {
	return fmt.Sprintf("  %-50s %s", e.path+":", e.message)
}

var slotRefRe = regexp.MustCompile(`\.slots\.([A-Za-z][A-Za-z0-9_]*)`)

// ValidateManifest runs all structural checks on a parsed SpireManifest.
func ValidateManifest(m *mfst.SpireManifest) []validationError {
	var errs []validationError

	if m.SpireVersion == "" {
		errs = append(errs, validationError{"spireVersion", "required field is missing or empty"})
	}
	if m.TemplateVersion == "" {
		errs = append(errs, validationError{"templateVersion", "required field is missing or empty"})
	}

	appSlotKeys := make(map[string]bool)
	for i, slot := range m.AppSlots {
		errs = append(errs, validateSlot(slot, fmt.Sprintf("appSlots[%d]", i), appSlotKeys)...)
	}

	svcSlotKeys := make(map[string]bool)
	for i, slot := range m.ServiceConfig.ServicesSlots {
		errs = append(errs, validateSlot(slot, fmt.Sprintf("serviceConfig.servicesSlots[%d]", i), svcSlotKeys)...)
	}

	allKeys := make(map[string]bool)
	for k := range appSlotKeys {
		allKeys[k] = true
	}
	for k := range svcSlotKeys {
		allKeys[k] = true
	}

	for i, pr := range m.PathRenames {
		errs = append(errs, validatePathRename(pr, fmt.Sprintf("pathRenames[%d]", i), allKeys)...)
	}

	for i, tf := range m.TemplateFiles {
		path := fmt.Sprintf("templateFiles[%d]", i)
		if tf.Source == "" {
			errs = append(errs, validationError{path + ".source", "source is required"})
		}
		if tf.Destination == "" {
			errs = append(errs, validationError{path + ".destination", "destination is required"})
		}
	}

	for i, pr := range m.ServiceConfig.PathRenames {
		errs = append(errs, validatePathRename(pr, fmt.Sprintf("serviceConfig.pathRenames[%d]", i), allKeys)...)
	}

	for i, hook := range m.ServiceConfig.PostHooks {
		base := fmt.Sprintf("serviceConfig.postHooks[%d]", i)
		if hook.Condition != "" {
			errs = append(errs, validateExpr(hook.Condition, base+".condition", allKeys)...)
		}
		for j, pathExpr := range hook.RemovePaths {
			errs = append(errs, validateExpr(pathExpr, fmt.Sprintf("%s.removePaths[%d]", base, j), allKeys)...)
		}
		for j, pr := range hook.PathRenames {
			errs = append(errs, validatePathRename(pr, fmt.Sprintf("%s.pathRenames[%d]", base, j), allKeys)...)
		}
	}

	return errs
}

func validateSlot(slot mfst.Slot, path string, seenKeys map[string]bool) []validationError {
	var errs []validationError

	if slot.Key == "" {
		errs = append(errs, validationError{path + ".key", "key is required"})
	} else if seenKeys[slot.Key] {
		errs = append(errs, validationError{path + ".key", fmt.Sprintf("duplicate slot key %q", slot.Key)})
	} else {
		seenKeys[slot.Key] = true
	}

	if slot.Type == mfst.DynamicValue && slot.Expression == "" {
		errs = append(errs, validationError{path + ".expression", "DynamicValue slot must have an expression"})
	}

	if slot.Expression != "" {
		errs = append(errs, validateExpr(slot.Expression, path+".expression", nil)...)
	}

	if slot.Validation != "" {
		if _, err := engine.ParseValidationRule(slot.Validation); err != nil {
			errs = append(errs, validationError{path + ".validation", err.Error()})
		}
	}

	return errs
}

func validatePathRename(pr mfst.PathRename, path string, allKeys map[string]bool) []validationError {
	var errs []validationError
	if pr.Pattern == "" {
		errs = append(errs, validationError{path + ".pattern", "pattern is required"})
	}
	if pr.Expression == "" {
		errs = append(errs, validationError{path + ".expression", "expression is required"})
	} else {
		errs = append(errs, validateExpr(pr.Expression, path+".expression", allKeys)...)
	}
	return errs
}

func validateExpr(expr, path string, knownKeys map[string]bool) []validationError {
	var errs []validationError

	_, err := template.New("check").Delims("[[", "]]").Funcs(engine.PipelineFuncs()).Parse(expr)
	if err != nil {
		errs = append(errs, validationError{path, fmt.Sprintf("invalid Go template: %v", err)})
		return errs
	}

	if knownKeys != nil {
		for _, m := range slotRefRe.FindAllStringSubmatch(expr, -1) {
			key := m[1]
			if !knownKeys[key] {
				errs = append(errs, validationError{path, fmt.Sprintf("references unknown slot key %q", key)})
			}
		}
	}

	return errs
}

var ValidateManifestCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the .spire/manifest.yaml for structural errors",
	Long: `Parses and validates the Spire manifest, reporting:

  - Missing required fields (spireVersion, templateVersion)
  - Slot integrity: duplicate keys, DynamicValue without expression, invalid validation rules
  - Invalid Go template expressions in slots, pathRenames, and postHooks
  - Cross-references to undefined slot keys in pathRenames and postHooks

Exits with code 1 when errors are found — suitable for CI/CD pipelines.`,
	Run: func(cmd *cobra.Command, args []string) {
		commoncmds.SwitchToWorkdir(cmd)

		file, _ := cmd.Flags().GetString("file")
		if file == "" {
			file = mfst.FilePath
		}

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ Cannot read %s: %v\n", file, err)
			os.Exit(1)
		}

		m := &mfst.SpireManifest{}
		if err := yaml.Unmarshal(data, m); err != nil {
			fmt.Printf("❌ YAML parse error in %s:\n   %v\n", file, err)
			os.Exit(1)
		}

		fmt.Printf("Validating %s...\n\n", file)

		errs := ValidateManifest(m)
		if len(errs) == 0 {
			fmt.Println("Manifest is valid")
			return
		}

		fmt.Printf("❌ Found %d error(s):\n\n", len(errs))
		for _, e := range errs {
			fmt.Println(e)
		}
		fmt.Println()
		os.Exit(1)
	},
}

func init() {
	ValidateManifestCmd.Flags().String("file", "", fmt.Sprintf("Path to the manifest file (default: %s)", mfst.FilePath))
}
