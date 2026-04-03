package fs

import (
	"context"

	"github.com/thkx/deepagent/tools"
)

func NewEditFileTool(backend Backend) tools.Tool {
	return tools.NewToolWithParameters(
		"edit_file",
		"Edit a file using natural language instructions (LLM-assisted). Args: path (required), instructions (required)",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":         map[string]any{"type": "string"},
				"instructions": map[string]any{"type": "string"},
			},
			"required":             []string{"path", "instructions"},
			"additionalProperties": false,
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			path, ok1 := args["path"].(string)
			instructions, ok2 := args["instructions"].(string)
			if !ok1 || path == "" || !ok2 || instructions == "" {
				return nil, ErrMissingPathOrInstructions
			}
			err := backend.Edit(ctx, path, instructions)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"path":         path,
				"status":       "edited",
				"instructions": instructions,
			}, nil
		},
	)
}
