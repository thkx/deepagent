package builtin

import (
	"context"
	"fmt"

	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

func NewExecuteTool(backend fs.Backend) tools.Tool {
	return NewExecuteToolWithConfig(backend, nil)
}

func NewExecuteToolWithConfig(backend fs.Backend, cfg *ExecuteConfig) tools.Tool {
	// 如果传入的是 SandboxBackend，则使用其 Execute 方法
	sandbox, ok := backend.(*SandboxBackend)
	if !ok {
		if cfg != nil {
			sandbox = NewSandboxBackendWithConfig(backend, *cfg).(*SandboxBackend)
		} else {
			// fallback
			sandbox = NewSandboxBackend(backend).(*SandboxBackend)
		}
	} else if cfg != nil {
		sandbox.config = *cfg
	}

	return tools.NewToolWithParameters(
		"execute",
		"Execute a command in secure sandbox. Args: command (required)",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
			},
			"required":             []string{"command"},
			"additionalProperties": false,
		},
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
