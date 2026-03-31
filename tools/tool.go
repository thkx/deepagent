package tools

import "context"

type Tool interface {
	Name() string
	Description() string
	Call(ctx context.Context, args map[string]any) (any, error)
}

type FuncTool struct {
	name        string
	description string
	fn          func(ctx context.Context, args map[string]any) (any, error)
}

func NewTool(name, desc string, fn func(ctx context.Context, args map[string]any) (any, error)) Tool {
	return &FuncTool{name: name, description: desc, fn: fn}
}

func (t *FuncTool) Name() string        { return t.name }
func (t *FuncTool) Description() string { return t.description }
func (t *FuncTool) Call(ctx context.Context, args map[string]any) (any, error) {
	return t.fn(ctx, args)
}
