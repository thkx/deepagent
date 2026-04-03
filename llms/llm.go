package llms

import "context"

type ChatMessage struct {
	Role    string
	Content string
}

type ToolCall struct {
	ID        string
	Name      string
	Arguments string // JSON string
}

type Tool struct {
	Name        string
	Description string
	Parameters  any // JSON schema 或 struct
}

type ChatModel interface {
	Invoke(ctx context.Context, messages []ChatMessage, tools []Tool) (string, []ToolCall, error)
}
