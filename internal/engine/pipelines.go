package engine

import (
	"crypto/rand"
	"math/big"
	"strings"
	"text/template"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// titleCaser title-cases words using Unicode-aware rules (replacing the
// deprecated strings.Title).
var titleCaser = cases.Title(language.Und)

const (
	pipelineCharsetAlphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	pipelineCharsetSpecial      = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

// PipelineFuncs returns the set of built-in template functions available in
// slot expressions and template file rendering.
func PipelineFuncs() template.FuncMap {
	return template.FuncMap{
		"slugify":          Slugify,
		"upper":            strings.ToUpper,
		"lower":            strings.ToLower,
		"title":            titleCaser.String,
		"pascalCase":       PascalCase,
		"camelCase":        CamelCase,
		"snakeCase":        SnakeCase,
		"trimSuffix":       strings.TrimSuffix,
		"trimPrefix":       strings.TrimPrefix,
		"replace":          pipelineReplace,
		"ensureSuffix":     EnsureSuffix,
		"default":          pipelineDefault,
		"generatePassword": func(length int, special ...bool) string { return GeneratePassword(length, special...) },
		"join":             strings.Join,
		"split":            strings.Split,
		"contains":         strings.Contains,
		"hasPrefix":        strings.HasPrefix,
		"hasSuffix":        strings.HasSuffix,
		"repeat":           strings.Repeat,
		"trimSpace":        strings.TrimSpace,
	}
}

// Slugify converts a string to a lowercase slug (hyphens, no spaces).
// Any character that is not a letter or digit becomes a hyphen; consecutive hyphens are collapsed.
func Slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	// Collapse consecutive hyphens
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	return result
}

// PascalCase converts a string to PascalCase.
func PascalCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, "")
}

// CamelCase converts a string to camelCase.
func CamelCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		if len(w) == 0 {
			continue
		}
		if i == 0 {
			words[i] = strings.ToLower(w)
		} else {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, "")
}

// SnakeCase converts a string to snake_case.
func SnakeCase(s string) string {
	words := splitWords(s)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	return strings.Join(words, "_")
}

// EnsureSuffix returns s with the suffix appended if not already present.
// Argument order matches Go template piping: value | ensureSuffix "-api" calls EnsureSuffix("-api", value).
func EnsureSuffix(suffix, s string) string {
	if strings.HasSuffix(s, suffix) {
		return s
	}
	return s + suffix
}

// GeneratePassword produces a random alphanumeric password of the given length.
// Pass true as the second argument to include special characters.
func GeneratePassword(length int, special ...bool) string {
	charset := pipelineCharsetAlphanumeric
	if len(special) > 0 && special[0] {
		charset += pipelineCharsetSpecial
	}
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

func pipelineReplace(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func pipelineDefault(def, val string) string {
	if val == "" {
		return def
	}
	return val
}

// splitWords splits a string on hyphens, underscores, spaces, and case boundaries.
func splitWords(s string) []string {
	var words []string
	var current strings.Builder
	for i, r := range s {
		if r == '-' || r == '_' || r == ' ' {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			continue
		}
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(s[i-1])) {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}
