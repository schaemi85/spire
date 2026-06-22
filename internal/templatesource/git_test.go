package templatesource

import (
	"testing"
)

func TestParseSemverParts(t *testing.T) {
	tests := []struct {
		input    string
		expected []int
	}{
		{"v1.2.3", []int{1, 2, 3}},
		{"1.2.3", []int{1, 2, 3}},
		{"v0.0.1", []int{0, 0, 1}},
		{"v1.0.0-beta", []int{1, 0, 0}},
		{"v2.1.0-rc.1", []int{2, 1, 0}},
		{"10.20.30", []int{10, 20, 30}},
		{"v1.0", []int{1, 0}},
		{"v1", []int{1}},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseSemverParts(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("parseSemverParts(%q) = %v, want %v", tt.input, got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("parseSemverParts(%q)[%d] = %d, want %d", tt.input, i, got[i], tt.expected[i])
				}
			}
		})
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a, b string
		want int // >0 if a > b, <0 if a < b, 0 if equal
	}{
		{"v1.0.0", "v0.9.0", 1},
		{"v0.9.0", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0", 0},
		{"v1.2.3", "v1.2.2", 1},
		{"v1.2.3", "v1.3.0", -1},
		{"v2.0.0", "v1.99.99", 1},
		{"v10.0.0", "v9.0.0", 1},
		{"v1.0.0-beta", "v1.0.0-alpha", 0}, // pre-release stripped, both are 1.0.0
		{"v1.0", "v1.0.0", -1},             // fewer parts = smaller
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := compareSemver(tt.a, tt.b)
			switch {
			case tt.want > 0 && got <= 0:
				t.Errorf("compareSemver(%q, %q) = %d, expected > 0", tt.a, tt.b, got)
			case tt.want < 0 && got >= 0:
				t.Errorf("compareSemver(%q, %q) = %d, expected < 0", tt.a, tt.b, got)
			case tt.want == 0 && got != 0:
				t.Errorf("compareSemver(%q, %q) = %d, expected 0", tt.a, tt.b, got)
			}
		})
	}
}

func TestCompareSemverSortOrder(t *testing.T) {
	// Ensure a list sorted with compareSemver is in descending order (newest first)
	tags := []string{"v0.1.0", "v1.0.0", "v0.2.0", "v2.0.0", "v1.1.0"}

	// Simple bubble sort using compareSemver (descending)
	for i := 0; i < len(tags); i++ {
		for j := i + 1; j < len(tags); j++ {
			if compareSemver(tags[i], tags[j]) < 0 {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}

	expected := []string{"v2.0.0", "v1.1.0", "v1.0.0", "v0.2.0", "v0.1.0"}
	for i, tag := range tags {
		if tag != expected[i] {
			t.Errorf("sorted[%d] = %q, want %q", i, tag, expected[i])
		}
	}
}

func TestNewGitSourceFields(t *testing.T) {
	src := NewGitSource("https://github.com/example/repo.git")
	if src.repoURL != "https://github.com/example/repo.git" {
		t.Errorf("repoURL = %q", src.repoURL)
	}
	if src.cloneDir != "" {
		t.Errorf("cloneDir should be empty initially, got %q", src.cloneDir)
	}
}

func TestGitSourceCleanupNoOp(t *testing.T) {
	src := NewGitSource("https://example.com/repo.git")
	// Cleanup without Download should not panic
	src.Cleanup()
}
