package agent

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Input struct {
	Messages []Message `json:"messages"`
	ThreadID string    `json:"thread_id,omitempty"`
}

type Output map[string]any // 兼容 Python {"messages": [...]}

type ToolResult struct {
	Tool  string `json:"tool"`
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
	Code  string `json:"code,omitempty"`
}

type Event struct {
	Type    string `json:"type"` // "message", "tool_call", "todo_update", "final", "error"
	Content any    `json:"content"`
}

type State struct {
	Messages  []Message `json:"messages"`
	ToolCalls []any     `json:"tool_calls,omitempty"`
	ThreadID  string    `json:"thread_id"`
	Iteration int       `json:"iteration"`
	Final     string    `json:"final,omitempty"`
	// Checkpoint any       `json:"checkpoint,omitempty"` // 自定义状态
}

type DeepAgent interface {
	Invoke(ctx context.Context, input Input) (Output, error)
	Stream(ctx context.Context, input Input) (<-chan Event, error)
}
