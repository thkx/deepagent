package groq

import (
	"context"
	"strings"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/llms/openai"
)

const defaultBaseURL = "https://api.groq.com/openai/v1"

type Groq struct {
	inner llms.ChatModel
}

func New(apiKey, model, baseURL string) *Groq {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Groq{
		inner: openai.NewWithBaseURL(apiKey, model, baseURL),
	}
}

func (g *Groq) Invoke(ctx context.Context, messages []llms.ChatMessage, tools []llms.Tool) (string, []llms.ToolCall, error) {
	return g.inner.Invoke(ctx, messages, tools)
}
