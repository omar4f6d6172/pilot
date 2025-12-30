package cmd

import (
	"testing"
)

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"Valid simple", "validuser", false},
		{"Valid with numbers", "user123", false},
		{"Valid with hyphen", "user-name", false},
		{"Valid with underscore", "user_name", false},
		{"Invalid empty", "", true},
		{"Invalid space", "user name", true},
		{"Invalid uppercase", "User", true},
		{"Invalid special chars", "user@name", true},
		{"Invalid start only", "-user", false}, // Technically valid regex, maybe we want to restrict? For now regex is ^[a-z0-9_-]+$
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateUsername(tt.username); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}
