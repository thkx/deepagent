package tools

import (
	"context"
	"fmt"
	"time"
)

type Tool interface {
	Name() string
	Description() string
	Parameters() any
	Call(ctx context.Context, args map[string]any) (any, error)
}

type FuncTool struct {
	name        string
	description string
	parameters  any
	fn          func(ctx context.Context, args map[string]any) (any, error)
}

func NewTool(name, desc string, fn func(ctx context.Context, args map[string]any) (any, error)) Tool {
	return &FuncTool{name: name, description: desc, parameters: nil, fn: fn}
}

func NewToolWithParameters(name, desc string, parameters any, fn func(ctx context.Context, args map[string]any) (any, error)) Tool {
	return &FuncTool{name: name, description: desc, parameters: parameters, fn: fn}
}

func (t *FuncTool) Name() string        { return t.name }
func (t *FuncTool) Description() string { return t.description }
func (t *FuncTool) Parameters() any     { return t.parameters }
func (t *FuncTool) Call(ctx context.Context, args map[string]any) (any, error) {
	return t.fn(ctx, args)
}

// TimeoutTool wraps a tool with timeout protection
type TimeoutTool struct {
	tool    Tool
	timeout time.Duration
}

// WithTimeout wraps a tool with a timeout
func WithTimeout(tool Tool, timeout time.Duration) Tool {
	return &TimeoutTool{
		tool:    tool,
		timeout: timeout,
	}
}

func (t *TimeoutTool) Name() string        { return t.tool.Name() }
func (t *TimeoutTool) Description() string { return t.tool.Description() }
func (t *TimeoutTool) Parameters() any     { return t.tool.Parameters() }

func (t *TimeoutTool) Call(ctx context.Context, args map[string]any) (any, error) {
	if t.timeout <= 0 {
		return t.tool.Call(ctx, args)
	}

	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Use a channel to handle timeout
	resultChan := make(chan struct {
		result any
		err    error
	}, 1)

	go func() {
		result, err := t.tool.Call(ctx, args)
		resultChan <- struct {
			result any
			err    error
		}{result, err}
	}()

	select {
	case res := <-resultChan:
		return res.result, res.err
	case <-ctx.Done():
		return nil, fmt.Errorf("tool %s timeout after %v", t.tool.Name(), t.timeout)
	}
}
