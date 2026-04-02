package agent

import (
	"context"
	"testing"
	"time"
)

func TestConsoleApproverRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := consoleApprover(ctx, "execute", map[string]any{"command": "pwd"})
	if err == nil {
		t.Fatalf("expected context cancellation error")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("consoleApprover should return quickly on cancelled context")
	}
}
