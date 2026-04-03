package agent

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

func NewTaskTool(parentOpts Options) tools.Tool {
	return tools.NewToolWithParameters("task",
		"Spawn a subagent for a subtask. Results will be automatically merged into parent filesystem.",
		map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description": map[string]any{"type": "string"},
			},
			"required":             []string{"description"},
			"additionalProperties": false,
		},
		func(ctx context.Context, args map[string]any) (any, error) {
			description, ok := args["description"].(string)
			if !ok || description == "" {
				return nil, fmt.Errorf("description is required")
			}

			subOpts := parentOpts
			subOpts.SystemPrompt = "You are a focused sub-agent. Task: " + description
			subOpts.Backend = fs.NewInMemoryBackend()

			subAgent, err := CreateDeepAgent(subOpts)
			if err != nil {
				return nil, err
			}

			subOut, err := subAgent.Invoke(ctx, Input{
				Messages: []Message{{Role: "user", Content: description}},
			})
			if err != nil {
				return nil, err
			}

			// 自动 merge 到父文件系统
			resultFile, err := nextSubagentResultFile(ctx, parentOpts.Backend, description)
			if err != nil {
				return nil, err
			}
			mergeContent := fmt.Sprintf("# Subagent Result: %s\n\n%s\n", description, subOut["final"])
			_ = parentOpts.Backend.Write(ctx, resultFile, mergeContent)

			return map[string]any{
				"task":        description,
				"result_file": resultFile,
				"status":      "merged",
				"output":      subOut,
			}, nil
		},
	)
}

func nextSubagentResultFile(ctx context.Context, backend fs.Backend, description string) (string, error) {
	base := slugify(description)
	if base == "" {
		base = "task"
	}
	if len(base) > 40 {
		base = base[:40]
		base = strings.Trim(base, "-")
	}

	existing, err := backend.List(ctx, ".")
	if err != nil {
		return "", err
	}
	exists := make(map[string]struct{}, len(existing))
	for _, name := range existing {
		exists[name] = struct{}{}
	}

	candidate := fmt.Sprintf("subagent_%s.md", base)
	if _, ok := exists[candidate]; !ok {
		return candidate, nil
	}

	for i := 2; i <= 9999; i++ {
		candidate = fmt.Sprintf("subagent_%s_%d.md", base, i)
		if _, ok := exists[candidate]; !ok {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to allocate subagent result filename for description: %q", description)
}

func slugify(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			if r > unicode.MaxASCII {
				continue
			}
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	return out
}
