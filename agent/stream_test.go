package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type streamScriptedLLM struct {
	steps []struct {
		content string
		calls   []llms.ToolCall
	}
	idx int
}

func TestStreamEmitsHITLEvents(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"command": "pwd"})
	llm := &streamScriptedLLM{
		steps: []struct {
			content string
			calls   []llms.ToolCall
		}{
			{
				calls: []llms.ToolCall{
					{ID: "hitl-1", Name: "execute", Arguments: string(args)},
				},
			},
			{
				content: "after-hitl",
			},
		},
	}

	executeTool := tools.NewToolWithParameters(
		"execute",
		"execute",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
			},
			"required": []string{"command"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			return nil, fmt.Errorf("should not be called when rejected")
		},
	)

	g := buildGraph(
		llm,
		[]tools.Tool{executeTool},
		"system",
		fs.NewInMemoryBackend(),
		NewFileCheckpointer(t.TempDir()),
		memory.NewFileMemoryStore(t.TempDir()),
		NewHumanInTheLoopWithApprover(
			InterruptConfig{"execute": true},
			func(ctx context.Context, toolName string, args any) (string, error) {
				return "rejected", fmt.Errorf("blocked by policy")
			},
		),
		&NoOpLogger{},
	)
	agt := &deepAgentImpl{graph: g}

	ch, err := agt.Stream(context.Background(), Input{
		ThreadID: "stream-hitl",
		Messages: []Message{{Role: "user", Content: "try execute"}},
	})
	if err != nil {
		t.Fatalf("stream error: %v", err)
	}

	var hasRequest, hasDecision bool
	for ev := range ch {
		if ev.Type == "hitl_request" {
			hasRequest = true
		}
		if ev.Type == "hitl_decision" {
			hasDecision = true
		}
	}
	if !hasRequest || !hasDecision {
		t.Fatalf("missing hitl stream events: request=%v decision=%v", hasRequest, hasDecision)
	}
}

func (s *streamScriptedLLM) Invoke(ctx context.Context, messages []llms.ChatMessage, toolDefs []llms.Tool) (string, []llms.ToolCall, error) {
	if s.idx >= len(s.steps) {
		return "done", nil, nil
	}
	step := s.steps[s.idx]
	s.idx++
	return step.content, step.calls, nil
}

func TestStreamEmitsProcessEvents(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"path": "note.md", "content": "hello"})
	llm := &streamScriptedLLM{
		steps: []struct {
			content string
			calls   []llms.ToolCall
		}{
			{
				content: "",
				calls: []llms.ToolCall{
					{ID: "tc1", Name: "write_file", Arguments: string(args)},
				},
			},
			{
				content: "done",
				calls:   nil,
			},
		},
	}

	writeTool := tools.NewToolWithParameters(
		"write_file",
		"write file",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			},
			"required": []string{"path", "content"},
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			return "ok", nil
		},
	)

	g := buildGraph(
		llm,
		[]tools.Tool{writeTool},
		"system",
		fs.NewInMemoryBackend(),
		NewFileCheckpointer(t.TempDir()),
		memory.NewFileMemoryStore(t.TempDir()),
		NewHumanInTheLoop(nil),
		&NoOpLogger{},
	)
	agt := &deepAgentImpl{graph: g}

	ch, err := agt.Stream(context.Background(), Input{
		ThreadID: "stream-thread",
		Messages: []Message{{Role: "user", Content: "run"}},
	})
	if err != nil {
		t.Fatalf("stream error: %v", err)
	}

	var hasModelCall, hasToolCall, hasToolResult, hasFinal bool
	for ev := range ch {
		switch ev.Type {
		case "model_call":
			hasModelCall = true
		case "tool_call":
			hasToolCall = true
		case "tool_result":
			hasToolResult = true
		case "final":
			hasFinal = true
		}
	}

	if !hasModelCall || !hasToolCall || !hasToolResult || !hasFinal {
		t.Fatalf("missing stream events: model=%v tool_call=%v tool_result=%v final=%v", hasModelCall, hasToolCall, hasToolResult, hasFinal)
	}
}
