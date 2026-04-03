package agent

import (
	"context"
	"fmt"
)

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
	Tool       string `json:"tool"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	OK         bool   `json:"ok"`
	Data       any    `json:"data,omitempty"`
	Error      string `json:"error,omitempty"`
	Code       string `json:"code,omitempty"`
}

type AgentError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	ThreadID  string `json:"thread_id,omitempty"`
	RequestID string `json:"request_id,omitempty"`
	Cause     error  `json:"-"`
}

func (e *AgentError) Error() string {
	if e == nil {
		return ""
	}
	if e.RequestID != "" {
		return fmt.Sprintf("%s (code=%s thread=%s request=%s)", e.Message, e.Code, e.ThreadID, e.RequestID)
	}
	return fmt.Sprintf("%s (code=%s)", e.Message, e.Code)
}

func (e *AgentError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *AgentError) Payload() map[string]any {
	if e == nil {
		return map[string]any{}
	}
	return map[string]any{
		"code":       e.Code,
		"message":    e.Message,
		"thread_id":  e.ThreadID,
		"request_id": e.RequestID,
	}
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
