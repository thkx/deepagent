package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/thkx/deepagent/llms"
)

type testLLM struct{}

func (m *testLLM) Invoke(ctx context.Context, messages []llms.ChatMessage, tools []llms.Tool) (string, []llms.ToolCall, error) {
	return "ok", nil, nil
}

func TestCreateDeepAgentRequiresAPIKeyWhenLLMIsNil(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	_, err := CreateDeepAgent(Options{})
	if err == nil {
		t.Fatalf("expected error when OPENAI_API_KEY is missing and opts.LLM is nil")
	}
}

func TestCreateDeepAgentAllowsInjectedLLMWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")

	_, err := CreateDeepAgent(Options{
		LLM: &testLLM{},
	})
	if err != nil {
		t.Fatalf("expected no error with injected LLM, got: %v", err)
	}
}

func TestBuildDefaultLLMAnthropicRequiresAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, err := buildDefaultLLM(Options{
		Provider: "anthropic",
		Model:    "claude-3-5-sonnet-latest",
	})
	if err == nil || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("expected ANTHROPIC_API_KEY error, got: %v", err)
	}
}

func TestBuildDefaultLLMGroqRequiresAPIKey(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "")
	_, err := buildDefaultLLM(Options{
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
	})
	if err == nil || !strings.Contains(err.Error(), "GROQ_API_KEY") {
		t.Fatalf("expected GROQ_API_KEY error, got: %v", err)
	}
}

func TestBuildDefaultLLMOllamaNoAPIKeyNeeded(t *testing.T) {
	llm, err := buildDefaultLLM(Options{
		Provider: "ollama",
		Model:    "qwen2.5:7b",
	})
	if err != nil {
		t.Fatalf("expected ollama llm without api key, got: %v", err)
	}
	if llm == nil {
		t.Fatalf("expected non-nil ollama llm")
	}
}

func TestInferProviderFromModel(t *testing.T) {
	tests := []struct {
		model    string
		expected string
	}{
		{model: "claude-3-7-sonnet-20250219", expected: "anthropic"},
		{model: "groq/llama-3.3-70b-versatile", expected: "groq"},
		{model: "ollama/qwen2.5:7b", expected: "ollama"},
		{model: "gpt-4o", expected: "openai"},
	}
	for _, tt := range tests {
		if got := inferProviderFromModel(tt.model); got != tt.expected {
			t.Fatalf("inferProviderFromModel(%q)=%q, expected %q", tt.model, got, tt.expected)
		}
	}
}
