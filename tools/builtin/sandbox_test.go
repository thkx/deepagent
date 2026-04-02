package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/thkx/deepagent/tools/builtin/fs"
)

func TestSandboxExecuteRejectsUnsafeSyntax(t *testing.T) {
	sb := NewSandboxBackend(fs.NewInMemoryBackend())
	out, err := sb.(*SandboxBackend).Execute(context.Background(), "ls; pwd")
	if err == nil {
		t.Fatalf("expected error for unsafe syntax, got output: %q", out)
	}
	if !strings.Contains(err.Error(), "unsafe command syntax") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSandboxExecuteRejectsTraversalArg(t *testing.T) {
	sb := NewSandboxBackend(fs.NewInMemoryBackend())
	_, err := sb.(*SandboxBackend).Execute(context.Background(), "cat ../secret")
	if err == nil {
		t.Fatalf("expected error for traversal argument")
	}
	if !strings.Contains(err.Error(), "unsafe argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}
