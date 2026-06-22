package tools

import (
	"testing"
)

func TestValidateBoolAnswer(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      bool
		wantError bool
	}{
		{"lowercase yes", "yes", true, false},
		{"uppercase yes", "YES", true, false},
		{"mixed case yes", "Yes", true, false},
		{"lowercase y", "y", true, false},
		{"uppercase y", "Y", true, false},
		{"lowercase no", "no", false, false},
		{"uppercase no", "NO", false, false},
		{"mixed case no", "No", false, false},
		{"lowercase n", "n", false, false},
		{"uppercase n", "N", false, false},
		{"invalid response", "maybe", false, true},
		{"empty string", "", false, true},
		{"random text", "xyz", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateBoolAnswer(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ValidateBoolAnswer(%q) expected error but got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateBoolAnswer(%q) unexpected error: %v", tt.input, err)
				}
				if got != tt.want {
					t.Errorf("ValidateBoolAnswer(%q) = %v, want %v", tt.input, got, tt.want)
				}
			}
		})
	}
}
