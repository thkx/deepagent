package builtin

import (
	"context"
	"fmt"

	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

func NewExecuteTool(backend fs.Backend) tools.Tool {
	// 如果传入的是 SandboxBackend，则使用其 Execute 方法
	sandbox, ok := backend.(*SandboxBackend)
	if !ok {
		// fallback
		sandbox = NewSandboxBackend(backend).(*SandboxBackend)
	}

	return tools.NewTool(
		"execute",
		"Execute a command in secure sandbox. Args: command (required)",
		func(ctx context.Context, args map[string]any) (any, error) {
			cmd, ok := args["command"].(string)
			if !ok || cmd == "" {
				return nil, fmt.Errorf("command is required")
			}

			output, err := sandbox.Execute(ctx, cmd)
			return map[string]any{
				"command": cmd,
				"output":  output,
				"error":   err != nil,
			}, nil
		},
	)
}
