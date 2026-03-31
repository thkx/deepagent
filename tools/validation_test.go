package tools

import (
	"context"
	"testing"
)

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name           string
		args           map[string]any
		requiredKeys   []string
		shouldPass     bool
	}{
		{
			name:         "all required params present",
			args:         map[string]any{"path": "/tmp", "content": "test"},
			requiredKeys: []string{"path", "content"},
			shouldPass:   true,
		},
		{
			name:         "missing required param",
			args:         map[string]any{"path": "/tmp"},
			requiredKeys: []string{"path", "content"},
			shouldPass:   false,
		},
		{
			name:         "empty string param",
			args:         map[string]any{"path": ""},
			requiredKeys: []string{"path"},
			shouldPass:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ValidateRequired(tt.requiredKeys...)
			err := validator(tt.args)
			if (err == nil) != tt.shouldPass {
				t.Errorf("ValidateRequired() error = %v, shouldPass = %v", err, tt.shouldPass)
			}
		})
	}
}

func TestValidateType(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		key       string
		typ       string
		shouldPass bool
	}{
		{
			name:      "valid string type",
			args:      map[string]any{"path": "/tmp"},
			key:       "path",
			typ:       "string",
			shouldPass: true,
		},
		{
			name:      "invalid string type",
			args:      map[string]any{"count": 42},
			key:       "count",
			typ:       "string",
			shouldPass: false,
		},
		{
			name:      "valid number type",
			args:      map[string]any{"count": 42.0},
			key:       "count",
			typ:       "number",
			shouldPass: true,
		},
		{
			name:      "valid boolean type",
			args:      map[string]any{"enabled": true},
			key:       "enabled",
			typ:       "boolean",
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ValidateType(tt.key, tt.typ)
			err := validator(tt.args)
			if (err == nil) != tt.shouldPass {
				t.Errorf("ValidateType() error = %v, shouldPass = %v", err, tt.shouldPass)
			}
		})
	}
}

func TestChainValidators(t *testing.T) {
	// Test chaining multiple validators
	args := map[string]any{"path": "/tmp", "count": 42}

	validator := ChainValidators(
		ValidateRequired("path", "count"),
		ValidateType("path", "string"),
		ValidateType("count", "number"),
	)

	err := validator(args)
	if err != nil {
		t.Errorf("ChainValidators() expected no error, got %v", err)
	}

	// Test with invalid args
	invalidArgs := map[string]any{"path": "/tmp", "count": "not a number"}
	err = validator(invalidArgs)
	if err == nil {
		t.Errorf("ChainValidators() expected error for invalid type")
	}
}

func TestWrappedTool(t *testing.T) {
	// Create a simple tool
	innerTool := NewTool("test", "test tool", func(ctx context.Context, args map[string]any) (any, error) {
		return "success", nil
	})

	// Wrap with validation
	validator := ValidateRequired("name")
	wrapped := NewWrappedTool(innerTool, validator)

	// Test with valid args
	result, err := wrapped.Call(context.Background(), map[string]any{"name": "test"})
	if err != nil {
		t.Errorf("WrappedTool.Call() expected no error, got %v", err)
	}
	if result != "success" {
		t.Errorf("WrappedTool.Call() expected 'success', got %v", result)
	}

	// Test with invalid args (missing required param)
	_, err = wrapped.Call(context.Background(), map[string]any{})
	if err == nil {
		t.Errorf("WrappedTool.Call() expected validation error")
	}
}

func TestValidateLength(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		key        string
		minLen     int
		maxLen     int
		shouldPass bool
	}{
		{
			name:       "valid length",
			args:       map[string]any{"text": "hello"},
			key:        "text",
			minLen:     1,
			maxLen:     10,
			shouldPass: true,
		},
		{
			name:       "too short",
			args:       map[string]any{"text": "hi"},
			key:        "text",
			minLen:     3,
			maxLen:     10,
			shouldPass: false,
		},
		{
			name:       "too long",
			args:       map[string]any{"text": "hello world"},
			key:        "text",
			minLen:     1,
			maxLen:     5,
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ValidateLength(tt.key, tt.minLen, tt.maxLen)
			err := validator(tt.args)
			if (err == nil) != tt.shouldPass {
				t.Errorf("ValidateLength() error = %v, shouldPass = %v", err, tt.shouldPass)
			}
		})
	}
}

func TestValidateEnum(t *testing.T) {
	tests := []struct {
		name       string
		args       map[string]any
		key        string
		allowed    []any
		shouldPass bool
	}{
		{
			name:       "valid enum value",
			args:       map[string]any{"status": "active"},
			key:        "status",
			allowed:    []any{"active", "inactive", "pending"},
			shouldPass: true,
		},
		{
			name:       "invalid enum value",
			args:       map[string]any{"status": "unknown"},
			key:        "status",
			allowed:    []any{"active", "inactive", "pending"},
			shouldPass: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := ValidateEnum(tt.key, tt.allowed...)
			err := validator(tt.args)
			if (err == nil) != tt.shouldPass {
				t.Errorf("ValidateEnum() error = %v, shouldPass = %v", err, tt.shouldPass)
			}
		})
	}
}
