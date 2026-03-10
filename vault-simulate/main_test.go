package main

import (
	"testing"
)

func TestParseRequestTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single type",
			input:    "kv",
			expected: []string{"kv"},
		},
		{
			name:     "multiple types",
			input:    "kv,ldap,database",
			expected: []string{"kv", "ldap", "database"},
		},
		{
			name:     "with spaces",
			input:    " kv , ldap , database ",
			expected: []string{"kv", "ldap", "database"},
		},
		{
			name:     "uppercase input normalized to lowercase",
			input:    "KV,LDAP,Database",
			expected: []string{"kv", "ldap", "database"},
		},
		{
			name:     "empty trailing comma ignored",
			input:    "kv,",
			expected: []string{"kv"},
		},
		{
			name:     "empty string returns empty slice",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRequestTypes(tt.input)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d types, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("expected type[%d]=%q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestValidateRequestTypes(t *testing.T) {
	tests := []struct {
		name    string
		types   []string
		wantErr bool
	}{
		{
			name:    "valid kv",
			types:   []string{"kv"},
			wantErr: false,
		},
		{
			name:    "valid all types",
			types:   []string{"kv", "ldap", "database"},
			wantErr: false,
		},
		{
			name:    "invalid type",
			types:   []string{"kv", "redis"},
			wantErr: true,
		},
		{
			name:    "empty slice",
			types:   []string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRequestTypes(tt.types)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequestTypes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
