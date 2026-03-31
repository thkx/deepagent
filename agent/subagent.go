package agent

import (
	"context"
	"fmt"

	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

func NewTaskTool(parentOpts Options) tools.Tool {
	return tools.NewTool("task",
		"Spawn a subagent for a subtask. Results will be automatically merged into parent filesystem.",
		func(ctx context.Context, args map[string]any) (any, error) {
			description, ok := args["description"].(string)
			if !ok || description == "" {
				return nil, fmt.Errorf("description is required")
			}

			subOpts := parentOpts
			subOpts.SystemPrompt = "You are a focused sub-agent. Task: " + description
			subOpts.Backend = fs.NewInMemoryBackend()

			subAgent, err := CreateDeepAgent(subOpts)
			if err != nil {
				return nil, err
			}

			subOut, err := subAgent.Invoke(ctx, Input{
				Messages: []Message{{Role: "user", Content: description}},
			})
			if err != nil {
				return nil, err
			}

			// 自动 merge 到父文件系统
			resultFile := fmt.Sprintf("subagent_%s.md", description[:min(30, len(description))])
			mergeContent := fmt.Sprintf("# Subagent Result: %s\n\n%s\n", description, subOut["final"])
			_ = parentOpts.Backend.Write(ctx, resultFile, mergeContent)

			return map[string]any{
				"task":        description,
				"result_file": resultFile,
				"status":      "merged",
				"output":      subOut,
			}, nil
		},
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
