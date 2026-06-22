package engine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/tools"
)

const (
	SpireDelimLeft  = "[["
	SpireDelimRight = "]]"
)

// RenderProjectFiles walks all text files under dir and renders each one in-place
// as a Go template. Only files containing Spire delimiters ([[ ]]) are rendered.
func RenderProjectFiles(dir string, rc *ResolveContext, ignorePaths []string) error {
	data := rc.templateData()
	funcs := PipelineFuncs()

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if slices.Contains(ignorePaths, name) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !tools.IsTextFile(path) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(content), SpireDelimLeft) {
			return nil
		}
		tmpl, err := template.New(filepath.Base(path)).Delims(SpireDelimLeft, SpireDelimRight).Funcs(funcs).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to render %s: %w", path, err)
		}
		return os.WriteFile(path, buf.Bytes(), info.Mode())
	})
}

// ApplyPathRenames evaluates each PathRename expression and renames matching
// files and directories. Directories are renamed deepest-first to avoid path conflicts.
func ApplyPathRenames(dir string, renames []manifest.PathRename, rc *ResolveContext) error {
	for _, r := range renames {
		replacement, err := EvaluateExpression(r.Expression, rc)
		if err != nil {
			return fmt.Errorf("failed to evaluate path rename expression %q: %w", r.Expression, err)
		}
		if replacement == "" {
			continue
		}
		if err := tools.RenamePathsWithPlaceholder(dir, r.Pattern, replacement); err != nil {
			return fmt.Errorf("failed to rename paths for pattern %q: %w", r.Pattern, err)
		}
	}
	return nil
}

// RenderTemplateFiles processes each TemplateFile entry: renders the Go template
// source file and writes the result to the destination.
func RenderTemplateFiles(templateDir, projectDir string, files []manifest.TemplateFile, rc *ResolveContext) error {
	data := rc.templateData()
	funcs := PipelineFuncs()

	for _, f := range files {
		src := filepath.Join(templateDir, f.Source)
		dst := filepath.Join(projectDir, f.Destination)

		content, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read template file %s: %w", f.Source, err)
		}

		tmpl, err := template.New(filepath.Base(f.Source)).Delims(SpireDelimLeft, SpireDelimRight).Funcs(funcs).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template file %s: %w", f.Source, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to render template file %s: %w", f.Source, err)
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", f.Destination, err)
		}

		if err := os.WriteFile(dst, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write rendered file %s: %w", f.Destination, err)
		}
	}
	return nil
}

// EvaluatePostHooks processes post-generation hooks against a target directory.
func EvaluatePostHooks(targetDir string, hooks []manifest.PostHook, rc *ResolveContext) error {
	for _, hook := range hooks {
		if hook.Condition != "" {
			result, err := EvaluateExpression(hook.Condition, rc)
			if err != nil {
				return fmt.Errorf("failed to evaluate post-hook condition %q: %w", hook.Condition, err)
			}
			if result != "true" {
				continue
			}
		}

		for _, pathExpr := range hook.RemovePaths {
			resolved, err := EvaluateExpression(pathExpr, rc)
			if err != nil {
				return fmt.Errorf("failed to evaluate removePath %q: %w", pathExpr, err)
			}
			target := filepath.Join(targetDir, resolved)
			if err := os.RemoveAll(target); err != nil {
				return fmt.Errorf("failed to remove %s: %w", target, err)
			}
		}

		if len(hook.PathRenames) > 0 {
			if err := ApplyPathRenames(targetDir, hook.PathRenames, rc); err != nil {
				return fmt.Errorf("failed to apply post-hook path renames: %w", err)
			}
		}
	}
	return nil
}
