package tools

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// IsTextFile does a simple check to see if a file is likely text.
func IsTextFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}
	return true
}

func RemoveFilesorDirs(filesOrDirs ...string) error {
	for _, fileOrDir := range filesOrDirs {
		if err := os.RemoveAll(fileOrDir); err != nil {
			return fmt.Errorf("error removing %s: %w", fileOrDir, err)
		}
	}
	return nil
}

type Replacement struct {
	Placeholder string
	Replacement string
}

func ReplaceInFiles(dir string, replacements []Replacement, ignoreFilesOrDirs []string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if slices.Contains(ignoreFilesOrDirs, name) {
			if info.IsDir() {
				// Skip the entire directory subtree
				return filepath.SkipDir
			}
			// Skip this file
			fmt.Printf("Skipping: %s at %s\n", name, path)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !IsTextFile(path) {
			return nil
		}
		fileContent, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, r := range replacements {
			fileContent = []byte(strings.ReplaceAll(string(fileContent), r.Placeholder, r.Replacement))
		}
		if err := os.WriteFile(path, fileContent, info.Mode()); err != nil {
			return err
		}
		return nil
	})
}

// RenamePathsWithPlaceholder renames files and directories whose base name
// contains the placeholder string. Files are renamed during the walk; directories
// are collected and renamed deepest-first in a second pass so that child paths
// are still valid when their parents are renamed.
func RenamePathsWithPlaceholder(dir, placeholder, replacement string) error {
	var dirs []string

	// First pass: rename files in-place, collect matching directories.
	if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		base := filepath.Base(path)
		if !strings.Contains(base, placeholder) {
			return nil
		}
		if info.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		// Rename file immediately — safe because it doesn't affect other walk paths.
		parent := filepath.Dir(path)
		newPath := filepath.Join(parent, strings.ReplaceAll(base, placeholder, replacement))
		return os.Rename(path, newPath)
	}); err != nil {
		return err
	}

	// Second pass: rename directories deepest-first.
	slices.SortFunc(dirs, func(a, b string) int {
		depthA := strings.Count(a, string(os.PathSeparator))
		depthB := strings.Count(b, string(os.PathSeparator))
		return depthB - depthA // deeper first
	})
	for _, d := range dirs {
		parent := filepath.Dir(d)
		base := filepath.Base(d)
		newPath := filepath.Join(parent, strings.ReplaceAll(base, placeholder, replacement))
		if err := os.Rename(d, newPath); err != nil {
			return fmt.Errorf("failed to rename %s to %s: %w", d, newPath, err)
		}
	}
	return nil
}

// TODO: works only on Unix-like systems. For Windows, we may need to use PowerShell or a Go library to get directory size.
// calculateDirSize calculates the size of a directory.
func CalculateDirSize(path string) string {
	cmd := exec.Command("du", "-sh", path)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	parts := strings.Fields(string(output))
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}
