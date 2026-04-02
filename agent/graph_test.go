package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type scriptedLLM struct {
	steps []struct {
		content string
		calls   []llms.ToolCall
	}
	idx int
}

func (s *scriptedLLM) Invoke(ctx context.Context, messages []llms.ChatMessage, toolDefs []llms.Tool) (string, []llms.ToolCall, error) {
	if s.idx >= len(s.steps) {
		return "done", nil, nil
	}
	step := s.steps[s.idx]
	s.idx++
	return step.content, step.calls, nil
}

func TestConvertToolsToLLMFormatIncludesParameters(t *testing.T) {
	tool := tools.NewToolWithParameters(
		"write_file",
		"write",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []string{"path"},
		},
		func(ctx context.Context, args map[string]any) (any, error) { return "ok", nil },
	)

	out := convertToolsToLLMFormat(map[string]tools.Tool{"write_file": tool})
	if len(out) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(out))
	}
	if out[0].Parameters == nil {
		t.Fatalf("expected tool parameters to be propagated")
	}
}

func TestRunSkipsToolExecutionWhenHitlRejects(t *testing.T) {
	var called bool
	tool := tools.NewToolWithParameters(
		"execute",
		"execute command",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
			},
			"required": []string{"command"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			called = true
			return "ok", nil
		},
	)

	llm := &scriptedLLM{
		steps: []struct {
			content string
			calls   []llms.ToolCall
		}{
			{
				calls: []llms.ToolCall{
					{Name: "execute", Arguments: `{"command":"pwd"}`},
				},
			},
			{content: "final answer"},
		},
	}

	hitl := NewHumanInTheLoopWithApprover(
		InterruptConfig{"execute": true},
		func(ctx context.Context, toolName string, args any) (string, error) {
			return "rejected", fmt.Errorf("rejected in test")
		},
	)

	g := buildGraph(
		llm,
		[]tools.Tool{tool},
		"system prompt",
		fs.NewInMemoryBackend(),
		NewFileCheckpointer(t.TempDir()),
		memory.NewFileMemoryStore(t.TempDir()),
		hitl,
		&NoOpLogger{},
	)

	out, err := g.Run(context.Background(), Input{
		ThreadID: "thread-hitl-reject",
		Messages: []Message{{Role: "user", Content: "run execute"}},
	})
	if err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if called {
		t.Fatalf("tool should not be called when human approval rejects")
	}
	if out["final"] != "final answer" {
		t.Fatalf("unexpected final output: %v", out["final"])
	}
}
