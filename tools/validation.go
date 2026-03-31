package tools

import (
	"context"
	"fmt"
	"reflect"
)

// ValidatorFunc is a validation function that checks tool arguments
type ValidatorFunc func(args map[string]any) error

// ValidateRequired creates a validator that checks for required fields
func ValidateRequired(keys ...string) ValidatorFunc {
	return func(args map[string]any) error {
		for _, key := range keys {
			if _, ok := args[key]; !ok || args[key] == "" {
				return fmt.Errorf("missing required parameter: %s", key)
			}
		}
		return nil
	}
}

// ValidateType creates a validator that checks parameter types
// expectedType should be one of: "string", "number", "boolean", "array", "object"
func ValidateType(key, expectedType string) ValidatorFunc {
	return func(args map[string]any) error {
		val, ok := args[key]
		if !ok {
			return nil // Type check only applies if value exists
		}

		switch expectedType {
		case "string":
			if _, ok := val.(string); !ok {
				return fmt.Errorf("parameter %s must be a string, got %T", key, val)
			}
		case "number":
			switch val.(type) {
			case float64, int, int32, int64, float32:
				// Valid numeric type
			default:
				return fmt.Errorf("parameter %s must be a number, got %T", key, val)
			}
		case "boolean":
			if _, ok := val.(bool); !ok {
				return fmt.Errorf("parameter %s must be a boolean, got %T", key, val)
			}
		case "array":
			if _, ok := val.([]any); !ok {
				// Also accept json.Number from JSON unmarshaling
				reflect.TypeOf(val).Kind()
				return fmt.Errorf("parameter %s must be an array, got %T", key, val)
			}
		case "object":
			if _, ok := val.(map[string]any); !ok {
				return fmt.Errorf("parameter %s must be an object, got %T", key, val)
			}
		default:
			return fmt.Errorf("unknown type: %s", expectedType)
		}
		return nil
	}
}

// ValidateLength creates a validator that checks string length
func ValidateLength(key string, minLen, maxLen int) ValidatorFunc {
	return func(args map[string]any) error {
		val, ok := args[key]
		if !ok {
			return nil
		}

		strVal, ok := val.(string)
		if !ok {
			return fmt.Errorf("parameter %s must be a string", key)
		}

		if len(strVal) < minLen {
			return fmt.Errorf("parameter %s is too short (min %d, got %d)", key, minLen, len(strVal))
		}
		if maxLen > 0 && len(strVal) > maxLen {
			return fmt.Errorf("parameter %s is too long (max %d, got %d)", key, maxLen, len(strVal))
		}
		return nil
	}
}

// ValidateEnum creates a validator that checks if value is one of allowed values
func ValidateEnum(key string, allowedValues ...any) ValidatorFunc {
	return func(args map[string]any) error {
		val, ok := args[key]
		if !ok {
			return nil
		}

		for _, allowed := range allowedValues {
			if fmt.Sprintf("%v", val) == fmt.Sprintf("%v", allowed) {
				return nil
			}
		}

		return fmt.Errorf("parameter %s has invalid value %v (allowed: %v)", key, val, allowedValues)
	}
}

// ChainValidators combines multiple validators into a single validator
func ChainValidators(validators ...ValidatorFunc) ValidatorFunc {
	return func(args map[string]any) error {
		for _, validator := range validators {
			if err := validator(args); err != nil {
				return err
			}
		}
		return nil
	}
}

// WrappedTool wraps a Tool with parameter validation
type WrappedTool struct {
	tool       Tool
	validators ValidatorFunc
}

// NewWrappedTool creates a new wrapped tool with validators
func NewWrappedTool(tool Tool, validators ValidatorFunc) Tool {
	return &WrappedTool{
		tool:       tool,
		validators: validators,
	}
}

func (wt *WrappedTool) Name() string {
	return wt.tool.Name()
}

func (wt *WrappedTool) Description() string {
	return wt.tool.Description()
}

func (wt *WrappedTool) Call(ctx context.Context, args map[string]any) (any, error) {
	// Run validators first
	if wt.validators != nil {
		if err := wt.validators(args); err != nil {
			return nil, err
		}
	}
	// If validation passes, call the wrapped tool
	return wt.tool.Call(ctx, args)
}
