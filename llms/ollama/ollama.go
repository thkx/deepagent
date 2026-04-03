package ollama

import (
	"context"
	"strings"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/llms/openai"
)

const defaultBaseURL = "http://localhost:11434/v1"

type Ollama struct {
	inner llms.ChatModel
}

func New(model, baseURL string) *Ollama {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	// Ollama OpenAI-compatible endpoint typically ignores auth token.
	return &Ollama{
		inner: openai.NewWithBaseURL("ollama", model, baseURL),
	}
}

func (o *Ollama) Invoke(ctx context.Context, messages []llms.ChatMessage, tools []llms.Tool) (string, []llms.ToolCall, error) {
	return o.inner.Invoke(ctx, messages, tools)
}
