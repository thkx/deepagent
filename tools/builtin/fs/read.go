package fs

import (
	"context"

	"github.com/thkx/deepagent/tools"
)

func NewReadFileTool(backend Backend) tools.Tool {
	return tools.NewTool(
		"read_file",
		"Read the content of a file. Args: path (required)",
		func(ctx context.Context, args map[string]any) (any, error) {
			path, ok := args["path"].(string)
			if !ok || path == "" {
				return nil, ErrMissingPath
			}
			content, err := backend.Read(ctx, path)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"path":    path,
				"content": content,
			}, nil
		},
	)
}
