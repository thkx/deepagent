package builtin

import (
	"context"

	"github.com/thkx/deepagent/tools"
)

type Todo struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Status      string `json:"status"` // pending | done | blocked
}

func NewWriteTodosTool() tools.Tool {
	return tools.NewToolWithParameters("write_todos",
		"Break down complex task into a trackable todo list. Return structured JSON.",
		map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": true,
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			// 实际项目中结合 LLM structured output 返回 Todo 列表
			return []Todo{
				{ID: "1", Description: "Analyze task", Status: "done"},
				{ID: "2", Description: "Execute subtasks", Status: "pending"},
			}, nil
		},
	)
}
