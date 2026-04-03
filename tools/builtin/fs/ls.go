package fs

import (
	"context"

	"github.com/thkx/deepagent/tools"
)

func NewLSTool(backend Backend) tools.Tool {
	return tools.NewToolWithParameters(
		"ls",
		"List files and directories in the virtual filesystem. Args: path (optional, default: '.')",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"additionalProperties": false,
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			path := "."
			if p, ok := args["path"].(string); ok && p != "" {
				path = p
			}
			files, err := backend.List(ctx, path)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"path":  path,
				"files": files,
			}, nil
		},
	)
}
