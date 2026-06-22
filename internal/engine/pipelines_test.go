package engine

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"My Cool App", "my-cool-app"},
		{"hello world", "hello-world"},
		{"UPPER CASE", "upper-case"},
		{"already-slugged", "already-slugged"},
		{"  spaces  around  ", "spaces-around"},
		{"special!@#chars", "special-chars"},
		{"multiple---hyphens", "multiple-hyphens"},
		{"", ""},
		{"single", "single"},
		{"CamelCaseInput", "camelcaseinput"},
		{"dots.and.dots", "dots-and-dots"},
		{"under_scores", "under-scores"},
		{"mixed-Delimiters_and Spaces", "mixed-delimiters-and-spaces"},
		{"123numbers", "123numbers"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.expected {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-cool-app", "MyCoolApp"},
		{"hello world", "HelloWorld"},
		{"single", "Single"},
		{"already", "Already"},
		{"", ""},
		{"a-b-c", "ABC"},
		{"under_score_case", "UnderScoreCase"},
		{"UPPER-CASE", "UpperCase"},
		{"mixed-Under_score", "MixedUnderScore"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := PascalCase(tt.input)
			if got != tt.expected {
				t.Errorf("PascalCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-cool-app", "myCoolApp"},
		{"hello world", "helloWorld"},
		{"single", "single"},
		{"", ""},
		{"a-b-c", "aBC"},
		{"under_score_case", "underScoreCase"},
		{"UPPER-CASE", "upperCase"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := CamelCase(tt.input)
			if got != tt.expected {
				t.Errorf("CamelCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSnakeCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-cool-app", "my_cool_app"},
		{"hello world", "hello_world"},
		{"single", "single"},
		{"", ""},
		{"UPPER-CASE", "upper_case"},
		{"under_score_case", "under_score_case"},
		{"mixed-Under_score", "mixed_under_score"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SnakeCase(tt.input)
			if got != tt.expected {
				t.Errorf("SnakeCase(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestEnsureSuffix(t *testing.T) {
	tests := []struct {
		suffix   string
		input    string
		expected string
	}{
		{"-api", "my-service", "my-service-api"},
		{"-api", "my-service-api", "my-service-api"},
		{"/", "path/to/dir", "path/to/dir/"},
		{"/", "path/to/dir/", "path/to/dir/"},
		{"", "anything", "anything"},
		{".go", "main", "main.go"},
		{".go", "main.go", "main.go"},
	}
	for _, tt := range tests {
		t.Run(tt.input+"_"+tt.suffix, func(t *testing.T) {
			got := EnsureSuffix(tt.suffix, tt.input)
			if got != tt.expected {
				t.Errorf("EnsureSuffix(%q, %q) = %q, want %q", tt.suffix, tt.input, got, tt.expected)
			}
		})
	}
}

func TestPipelineFuncsRegistered(t *testing.T) {
	funcs := PipelineFuncs()
	expected := []string{
		"slugify", "upper", "lower", "title",
		"pascalCase", "camelCase", "snakeCase",
		"trimSuffix", "trimPrefix", "replace",
		"ensureSuffix", "default", "generatePassword",
		"join", "split", "contains",
		"hasPrefix", "hasSuffix", "repeat", "trimSpace",
	}
	for _, name := range expected {
		if funcs[name] == nil {
			t.Errorf("PipelineFuncs() missing expected function %q", name)
		}
	}
}

func TestGeneratePasswordLength(t *testing.T) {
	lengths := []int{0, 1, 8, 16, 32, 64}
	for _, l := range lengths {
		p := GeneratePassword(l)
		if len(p) != l {
			t.Errorf("GeneratePassword(%d) returned length %d", l, len(p))
		}
		ps := GeneratePassword(l, true)
		if len(ps) != l {
			t.Errorf("GeneratePassword(%d, true) returned length %d", l, len(ps))
		}
	}
}

func TestGeneratePasswordUniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		p := GeneratePassword(16)
		if seen[p] {
			t.Fatalf("GeneratePassword produced duplicate: %s", p)
		}
		seen[p] = true
	}
}

func TestGeneratePasswordSpecialChars(t *testing.T) {
	const length = 100
	const iterations = 50

	validAll := pipelineCharsetAlphanumeric + pipelineCharsetSpecial
	hasSpecial := false

	for i := 0; i < iterations; i++ {
		p := GeneratePassword(length, true)
		for _, c := range p {
			if !strings.ContainsRune(validAll, c) {
				t.Errorf("GeneratePassword(special=true) produced unexpected char: %c", c)
			}
		}
		if strings.ContainsAny(p, pipelineCharsetSpecial) {
			hasSpecial = true
		}
	}

	if !hasSpecial {
		t.Error("GeneratePassword(special=true) never produced a special character in 50 iterations")
	}
}

func TestGeneratePasswordNoSpecialByDefault(t *testing.T) {
	const length = 200
	const iterations = 20

	for i := 0; i < iterations; i++ {
		p := GeneratePassword(length)
		for _, c := range p {
			if strings.ContainsRune(pipelineCharsetSpecial, c) {
				t.Errorf("GeneratePassword (no special) produced special character: %c", c)
			}
		}
	}
}
