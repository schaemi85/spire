package application

import "testing"

func TestIsCommitHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid commit hashes
		{
			name:     "Full SHA-1 hash (40 chars)",
			input:    "a07fd0ab91777d11bb20e582587e03860c779983",
			expected: true,
		},
		{
			name:     "Short hash (7 chars)",
			input:    "a07fd0a",
			expected: true,
		},
		{
			name:     "Short hash (8 chars)",
			input:    "a07fd0ab",
			expected: true,
		},
		{
			name:     "Uppercase hex",
			input:    "A07FD0AB91777D11BB20E582587E03860C779983",
			expected: true,
		},
		{
			name:     "Mixed case hex",
			input:    "A07fd0aB91777d11bb20E582587e03860c779983",
			expected: true,
		},
		{
			name:     "Numeric only hash",
			input:    "1234567890",
			expected: true,
		},
		// Invalid inputs - should return false
		{
			name:     "Semantic version tag",
			input:    "v1.7.0",
			expected: false,
		},
		{
			name:     "Tag without v prefix",
			input:    "1.7.0",
			expected: false,
		},
		{
			name:     "Semantic version with prerelease",
			input:    "v1.7.0-alpha",
			expected: false,
		},
		{
			name:     "Semantic version with prerelease and build",
			input:    "v1.7.0-beta.1+build.123",
			expected: false,
		},
		{
			name:     "Semantic version without v and prerelease",
			input:    "2.0.0-rc.1",
			expected: false,
		},
		{
			name:     "Too short (6 chars)",
			input:    "a07fd0",
			expected: false,
		},
		{
			name:     "Too long (41 chars)",
			input:    "a07fd0ab91777d11bb20e582587e03860c7799831",
			expected: false,
		},
		{
			name:     "Contains non-hex character (g)",
			input:    "g07fd0ab91777d11bb20e582587e03860c779983",
			expected: false,
		},
		{
			name:     "Contains dash",
			input:    "a07fd0a-test",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Branch name",
			input:    "main",
			expected: false,
		},
		{
			name:     "Feature branch name",
			input:    "feature/new-feature",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isCommitHash(tt.input)
			if result != tt.expected {
				t.Errorf("isCommitHash(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSemverPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid semantic versions
		{
			name:     "Basic semver with v prefix",
			input:    "v1.0.0",
			expected: true,
		},
		{
			name:     "Basic semver without v prefix",
			input:    "1.0.0",
			expected: true,
		},
		{
			name:     "Semver with v prefix and prerelease",
			input:    "v1.7.0-alpha",
			expected: true,
		},
		{
			name:     "Semver with prerelease and build metadata",
			input:    "v1.7.0-beta.1+build.123",
			expected: true,
		},
		{
			name:     "Semver without v and with prerelease",
			input:    "2.0.0-rc.1",
			expected: true,
		},
		{
			name:     "Large version numbers",
			input:    "v10.20.30",
			expected: true,
		},
		{
			name:     "Version with complex prerelease",
			input:    "1.0.0-alpha.1.2.3",
			expected: true,
		},
		{
			name:     "Version with build metadata only",
			input:    "1.0.0+20130313144700",
			expected: true,
		},
		{
			name:     "Version with prerelease and build",
			input:    "1.0.0-beta+exp.sha.5114f85",
			expected: true,
		},
		// Invalid semantic versions - should return false
		{
			name:     "Commit hash (40 chars)",
			input:    "a07fd0ab91777d11bb20e582587e03860c779983",
			expected: false,
		},
		{
			name:     "Short commit hash",
			input:    "a07fd0a",
			expected: false,
		},
		{
			name:     "Missing patch version",
			input:    "v1.0",
			expected: false,
		},
		{
			name:     "Missing minor and patch",
			input:    "v1",
			expected: false,
		},
		{
			name:     "Leading zeros in major",
			input:    "v01.0.0",
			expected: false,
		},
		{
			name:     "Leading zeros in minor",
			input:    "v1.01.0",
			expected: false,
		},
		{
			name:     "Leading zeros in patch",
			input:    "v1.0.01",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Branch name",
			input:    "main",
			expected: false,
		},
		{
			name:     "Random text",
			input:    "not-a-version",
			expected: false,
		},
		{
			name:     "Version with spaces",
			input:    "v1.0.0 alpha",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSemverPattern(tt.input)
			if result != tt.expected {
				t.Errorf("isSemverPattern(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}
