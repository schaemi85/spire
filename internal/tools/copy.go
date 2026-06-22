package tools

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type GlobMode int
type globMatch struct {
	root string
	path string
}

const (
	Flatten GlobMode = iota
	PreserveFromRoot
	PreserveFullPath
)

func stat(path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)
	return info, err == nil
}

func Copy(src, dst string) error {
	return CopyWithMode(src, dst, PreserveFromRoot)
}

func CopyWithMode(src, dst string, mode GlobMode) error {
	matches, err := expand(src)
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return fmt.Errorf("source not found: %s", src)
	}

	dstInfo, dstExists := stat(dst)
	if !dstExists && len(matches) > 1 {
		if err := os.MkdirAll(dst, 0755); err != nil {
			return err
		}
	} else if len(matches) > 1 && dstExists && !dstInfo.IsDir() {
		return fmt.Errorf("multiple sources require destination to be a directory")
	}

	for _, m := range matches {
		info, err := os.Lstat(m.path)
		if err != nil {
			return err
		}

		var target string
		switch mode {
		case Flatten:
			target = filepath.Join(dst, filepath.Base(m.path))

		case PreserveFromRoot:
			rel, err := filepath.Rel(m.root, m.path)
			if err != nil {
				return err
			}
			target = filepath.Join(dst, rel)

		case PreserveFullPath:
			target = filepath.Join(dst, filepath.Clean(m.path))
		}

		if info.IsDir() {
			if err := CopyDir(m.path, target); err != nil {
				return err
			}
			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if err := CopyFile(m.path, target, info); err != nil {
			return err
		}
	}
	return nil
}

func expand(pattern string) ([]globMatch, error) {
	root := GlobRoot(pattern)

	var matches []globMatch
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ok, err := matchPattern(pattern, path)
		if err != nil {
			return err
		}
		if ok {
			matches = append(matches, globMatch{
				root: root,
				path: path,
			})
		}
		return nil
	})
	return matches, err
}

// matchPattern matches a path against a pattern that may contain ** for recursive directory matching.
func matchPattern(pattern, path string) (bool, error) {
	// If pattern doesn't contain **, use standard filepath.Match
	if !strings.Contains(pattern, "**") {
		return filepath.Match(pattern, path)
	}

	// Split pattern into segments
	patternParts := strings.Split(filepath.Clean(pattern), string(filepath.Separator))
	pathParts := strings.Split(filepath.Clean(path), string(filepath.Separator))

	return matchParts(patternParts, pathParts)
}

// matchParts recursively matches pattern parts against path parts, handling **.
func matchParts(pattern, path []string) (bool, error) {
	// Both exhausted - match
	if len(pattern) == 0 && len(path) == 0 {
		return true, nil
	}

	// Pattern exhausted but path remains - no match
	if len(pattern) == 0 {
		return false, nil
	}

	// Path exhausted but pattern remains - only match if remaining pattern is just **
	if len(path) == 0 {
		return len(pattern) == 1 && pattern[0] == "**", nil
	}

	// Handle ** - it can match zero or more path segments
	if pattern[0] == "**" {
		// Try matching ** with zero segments (skip it)
		if match, err := matchParts(pattern[1:], path); err != nil || match {
			return match, err
		}
		// Try matching ** with one or more segments (consume path segment)
		return matchParts(pattern, path[1:])
	}

	// Regular pattern matching for current segment
	match, err := filepath.Match(pattern[0], path[0])
	if err != nil || !match {
		return false, err
	}

	// Current segment matched, continue with remaining segments
	return matchParts(pattern[1:], path[1:])
}

func hasGlobMeta(p string) bool {
	for i := 0; i < len(p); i++ {
		switch p[i] {
		case '*', '?', '[':
			// check if escaped
			if i == 0 || p[i-1] != '\\' {
				return true
			}
		}
	}
	return false
}

func GlobRoot(pattern string) string {
	clean := filepath.Clean(pattern)

	// Windows: extract volume ("C:", "\\server\share"), Unix: ""
	volume := filepath.VolumeName(clean)

	// Remove volume for processing
	path := clean[len(volume):]

	// Detect absolute path
	isAbs := strings.HasPrefix(path, string(filepath.Separator))

	// Trim leading separator to avoid empty split
	path = strings.TrimPrefix(path, string(filepath.Separator))

	parts := strings.Split(path, string(filepath.Separator))

	var rootParts []string
	for _, p := range parts {
		if hasGlobMeta(p) {
			break
		}
		rootParts = append(rootParts, p)
	}

	// Rebuild root path
	root := filepath.Join(rootParts...)

	switch {
	case volume != "" && isAbs:
		if root == "" {
			return volume + string(filepath.Separator)
		}
		return volume + string(filepath.Separator) + root

	case isAbs:
		if root == "" {
			return string(filepath.Separator)
		}
		return string(filepath.Separator) + root

	default:
		if root == "" {
			return "."
		}
		return root
	}
}

func CopyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		return CopyFile(path, target, info)
	})
}

func CopyFile(src, dst string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode())
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
