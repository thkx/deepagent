package llms

import (
	"reflect"
	"testing"
)

func TestGeneratePropertySchema(t *testing.T) {
	tests := []struct {
		name string
		typ  reflect.Type
		want map[string]any
	}{
		{
			name: "string type",
			typ:  reflect.TypeOf(""),
			want: map[string]any{"type": "string"},
		},
		{
			name: "int type",
			typ:  reflect.TypeOf(int64(0)),
			want: map[string]any{"type": "number"},
		},
		{
			name: "float type",
			typ:  reflect.TypeOf(float64(0)),
			want: map[string]any{"type": "number"},
		},
		{
			name: "bool type",
			typ:  reflect.TypeOf(true),
			want: map[string]any{"type": "boolean"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GeneratePropertySchema(tt.typ)
			if got["type"] != tt.want["type"] {
				t.Errorf("GeneratePropertySchema() got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateParametersSchema(t *testing.T) {
	tests := []struct {
		name string
		tool Tool
		want map[string]any
	}{
		{
			name: "tool with nil parameters",
			tool: Tool{
				Name:        "test",
				Description: "test tool",
				Parameters:  nil,
			},
			want: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		{
			name: "tool with map parameters",
			tool: Tool{
				Name:        "test",
				Description: "test tool",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{"type": "string"},
					},
				},
			},
			want: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateParametersSchema(tt.tool)
			if got["type"] != tt.want["type"] {
				t.Errorf("GenerateParametersSchema() got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConvertToOpenAITools(t *testing.T) {
	tools := []Tool{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters:  nil,
		},
	}

	got := ConvertToOpenAITools(tools)
	if len(got) != 1 {
		t.Errorf("ConvertToOpenAITools() expected 1 tool, got %d", len(got))
	}

	if len(got) > 0 {
		toolDef := got[0].(map[string]any)
		if toolDef["type"] != "function" {
			t.Errorf("ConvertToOpenAITools() expected type 'function', got %v", toolDef["type"])
		}

		funcDef := toolDef["function"].(map[string]any)
		if funcDef["name"] != "test_tool" {
			t.Errorf("ConvertToOpenAITools() expected name 'test_tool', got %v", funcDef["name"])
		}
	}
}
