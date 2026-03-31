package fs

import (
	"context"

	"github.com/thkx/deepagent/tools"
)

func NewWriteFileTool(backend Backend) tools.Tool {
	return tools.NewTool(
		"write_file",
		"Write content to a file (create or overwrite). Args: path (required), content (required)",
		func(ctx context.Context, args map[string]any) (any, error) {
			path, ok1 := args["path"].(string)
			content, ok2 := args["content"].(string)
			if !ok1 || path == "" || !ok2 {
				return nil, ErrMissingPathOrContent
			}
			err := backend.Write(ctx, path, content)
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"path":   path,
				"status": "written",
				"length": len(content),
			}, nil
		},
	)
}
