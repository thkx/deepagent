package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type resumeAwareLLM struct {
	callIndex int
}

func (m *resumeAwareLLM) Invoke(ctx context.Context, messages []llms.ChatMessage, tools []llms.Tool) (string, []llms.ToolCall, error) {
	m.callIndex++
	switch m.callIndex {
	case 1:
		args, _ := json.Marshal(map[string]any{
			"path":    "plan.md",
			"content": "first plan",
		})
		return "", []llms.ToolCall{{Name: "write_file", Arguments: string(args)}}, nil
	case 2:
		return "first-final", nil, nil
	case 3:
		// Resume run should load prior messages from checkpoint
		if len(messages) < 4 {
			return "", nil, os.ErrInvalid
		}
		return "second-final", nil, nil
	default:
		return "done", nil, nil
	}
}

func TestInvokeToolCallCheckpointResume(t *testing.T) {
	cpDir := t.TempDir()
	memDir := t.TempDir()
	backend := fs.NewInMemoryBackend()
	model := &resumeAwareLLM{}

	agt, err := CreateDeepAgent(Options{
		LLM:          model,
		Backend:      backend,
		Checkpointer: NewFileCheckpointer(cpDir),
		Memory:       memory.NewFileMemoryStore(memDir),
		SystemPrompt: "You are test agent",
	})
	if err != nil {
		t.Fatalf("CreateDeepAgent error: %v", err)
	}

	out1, err := agt.Invoke(context.Background(), Input{
		ThreadID: "thread-e2e",
		Messages: []Message{{Role: "user", Content: "create a file"}},
	})
	if err != nil {
		t.Fatalf("first invoke error: %v", err)
	}
	if out1["final"] != "first-final" {
		t.Fatalf("unexpected first final: %v", out1["final"])
	}

	fileContent, err := backend.Read(context.Background(), "plan.md")
	if err != nil {
		t.Fatalf("expected written file, got err: %v", err)
	}
	if fileContent != "first plan" {
		t.Fatalf("unexpected file content: %q", fileContent)
	}

	if _, err := os.Stat(filepath.Join(cpDir, "thread-e2e.json")); err != nil {
		t.Fatalf("checkpoint file missing: %v", err)
	}

	out2, err := agt.Invoke(context.Background(), Input{
		ThreadID: "thread-e2e",
		Messages: []Message{{Role: "user", Content: "second turn"}},
	})
	if err != nil {
		t.Fatalf("second invoke error: %v", err)
	}
	if out2["final"] != "second-final" {
		t.Fatalf("unexpected second final: %v", out2["final"])
	}
}
