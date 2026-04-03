package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

func TestExecuteToolReturnsErrorWhenSandboxExecutionFails(t *testing.T) {
	cfg := DefaultExecuteConfig()
	cfg.AllowedCommands = map[string]bool{
		"echo": true,
	}

	tool := NewExecuteToolWithConfig(fs.NewInMemoryBackend(), &cfg)
	_, err := tool.Call(context.Background(), map[string]any{"command": "ls"})
	if err == nil {
		t.Fatalf("expected execute tool to return error when sandbox rejects command")
	}
	if !strings.Contains(err.Error(), "execute failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
