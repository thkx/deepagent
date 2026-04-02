package llms

import (
	"encoding/json"
	"reflect"
)

// GeneratePropertySchema generates JSON schema for a struct field type.
// Supports basic types (string, number, boolean), slices, and structs.
func GeneratePropertySchema(t reflect.Type) map[string]any {
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": GeneratePropertySchema(t.Elem()),
		}
	case reflect.Struct:
		return generateStructSchema(t)
	case reflect.Ptr:
		return GeneratePropertySchema(t.Elem())
	default:
		return map[string]any{"type": "object"}
	}
}

// generateStructSchema generates schema for a struct type
func generateStructSchema(t reflect.Type) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")

		if jsonTag == "-" || jsonTag == "" {
			continue
		}

		fieldName := jsonTag
		if jsonTag == "" {
			fieldName = field.Name
		}

		properties[fieldName] = GeneratePropertySchema(field.Type)

		// Check if field is required (no omitempty tag)
		if field.Tag.Get("json") != "" && field.Tag.Get("required") == "true" {
			required = append(required, fieldName)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// GenerateParametersSchema generates JSON schema for tool parameters.
// Returns OpenAI-compatible tool schema.
func GenerateParametersSchema(tool Tool) map[string]any {
	if tool.Parameters == nil {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		}
	}

	// If Parameters is already a map, assume it's JSON schema
	if paramsMap, ok := tool.Parameters.(map[string]any); ok {
		return paramsMap
	}

	// If Parameters is a struct or pointer to struct, generate schema
	paramsVal := reflect.ValueOf(tool.Parameters)
	if paramsVal.Kind() == reflect.Ptr {
		paramsVal = paramsVal.Elem()
	}

	if paramsVal.Kind() == reflect.Struct {
		return generateStructSchema(paramsVal.Type())
	}

	// Fallback: empty schema
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// ConvertToOpenAITools converts internal Tool definitions to OpenAI ChatCompletionTools format
func ConvertToOpenAITools(tools []Tool) []any {
	result := []any{}
	for _, tool := range tools {
		parametersSchema := GenerateParametersSchema(tool)

		openaiTool := map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  parametersSchema,
			},
		}
		result = append(result, openaiTool)
	}
	return result
}

// MarshalParametersSchema marshals parameters schema to JSON string for API calls
func MarshalParametersSchema(tool Tool) string {
	schema := GenerateParametersSchema(tool)
	bytes, _ := json.Marshal(schema)
	return string(bytes)
}
