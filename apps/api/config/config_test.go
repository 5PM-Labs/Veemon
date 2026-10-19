package config

import (
	"strings"
	"testing"
)

func TestConfig_Validate_PasetoSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty is rejected", "", true},
		{"placeholder is rejected", placeholderPasetoSecret, true},
		{"too short is rejected", "short-secret", true},
		{"31 bytes is rejected", strings.Repeat("a", 31), true},
		{"32 raw bytes is accepted", strings.Repeat("a", 32), false},
		{"64-char hex is accepted", strings.Repeat("ab", 32), false},
		{"64 chars non-hex falls back to length rule", strings.Repeat("g", 64), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{PasetoSecret: tt.secret}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
