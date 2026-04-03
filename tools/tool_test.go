package tools

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeout(t *testing.T) {
	tests := []struct {
		name          string
		toolDelay     time.Duration
		timeout       time.Duration
		shouldTimeout bool
	}{
		{
			name:          "tool completes before timeout",
			toolDelay:     100 * time.Millisecond,
			timeout:       500 * time.Millisecond,
			shouldTimeout: false,
		},
		{
			name:          "tool exceeds timeout",
			toolDelay:     500 * time.Millisecond,
			timeout:       100 * time.Millisecond,
			shouldTimeout: true,
		},
		{
			name:          "zero timeout disabled",
			toolDelay:     200 * time.Millisecond,
			timeout:       0,
			shouldTimeout: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a tool that takes toolDelay to complete
			innerTool := NewTool("test", "test tool", func(ctx context.Context, args map[string]any) (any, error) {
				timer := time.NewTimer(tt.toolDelay)
				defer timer.Stop()

				select {
				case <-timer.C:
					return "completed", nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			})

			wrapped := WithTimeout(innerTool, tt.timeout)
			_, err := wrapped.Call(context.Background(), map[string]any{})

			if tt.shouldTimeout && err == nil {
				t.Errorf("WithTimeout() expected timeout error, got nil")
			}
			if !tt.shouldTimeout && err != nil {
				t.Errorf("WithTimeout() expected no error, got %v", err)
			}
		})
	}
}

func TestTimeoutToolName(t *testing.T) {
	innerTool := NewTool("myTool", "description", func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	})

	wrapped := WithTimeout(innerTool, 100*time.Millisecond)
	if wrapped.Name() != "myTool" {
		t.Errorf("TimeoutTool.Name() expected 'myTool', got %v", wrapped.Name())
	}
}

func TestTimeoutToolDescription(t *testing.T) {
	innerTool := NewTool("myTool", "test description", func(ctx context.Context, args map[string]any) (any, error) {
		return nil, nil
	})

	wrapped := WithTimeout(innerTool, 100*time.Millisecond)
	if wrapped.Description() != "test description" {
		t.Errorf("TimeoutTool.Description() expected 'test description', got %v", wrapped.Description())
	}
}
