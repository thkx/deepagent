package agent

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type captureHITLAuditLogger struct {
	entries []HITLAuditEntry
}

func (c *captureHITLAuditLogger) LogHITLEvent(entry HITLAuditEntry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestFileHITLAuditLoggerWritesJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hitl_audit.jsonl")
	logger, err := NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("NewFileHITLAuditLogger error: %v", err)
	}

	if err := logger.LogHITLEvent(HITLAuditEntry{Event: "hitl_request", Tool: "execute"}); err != nil {
		t.Fatalf("LogHITLEvent error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	var entry HITLAuditEntry
	if err := json.Unmarshal(data[:len(data)-1], &entry); err != nil {
		t.Fatalf("invalid jsonl entry: %v", err)
	}
	if entry.Event != "hitl_request" || entry.Tool != "execute" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	if entry.Hash == "" {
		t.Fatalf("expected hash to be populated")
	}
	if err := VerifyHITLAuditFileChain(path); err != nil {
		t.Fatalf("expected valid audit chain, got: %v", err)
	}
}

func TestVerifyHITLAuditFileChainDetectsTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hitl_audit.jsonl")
	logger, err := NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("NewFileHITLAuditLogger error: %v", err)
	}
	if err := logger.LogHITLEvent(HITLAuditEntry{Event: "hitl_request", Tool: "execute"}); err != nil {
		t.Fatalf("LogHITLEvent error: %v", err)
	}
	if err := logger.LogHITLEvent(HITLAuditEntry{Event: "hitl_decision", Tool: "execute", Decision: "approved", Approved: true}); err != nil {
		t.Fatalf("LogHITLEvent error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	tampered := strings.Replace(string(data), "execute", "exfiltrate", 1)
	if err := os.WriteFile(path, []byte(tampered), 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	if err := VerifyHITLAuditFileChain(path); err == nil {
		t.Fatalf("expected tampering verification error")
	}
}

func TestVerifyHITLAuditFileChainStrictRejectsMissingRequiredFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "minimal.jsonl")
	logger, err := NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("NewFileHITLAuditLogger error: %v", err)
	}
	if err := logger.LogHITLEvent(HITLAuditEntry{Event: "hitl_request", Tool: "execute"}); err != nil {
		t.Fatalf("LogHITLEvent error: %v", err)
	}

	if err := VerifyHITLAuditFileChainStrict(path); err == nil {
		t.Fatalf("expected strict verification failure for missing required fields")
	}
}

func TestVerifyHITLAuditFileChainStrictPassesForCompleteEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "strict_ok.jsonl")
	logger, err := NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("NewFileHITLAuditLogger error: %v", err)
	}
	req := HITLAuditEntry{
		Event:      "hitl_request",
		Tool:       "execute",
		ToolCallID: "tc-1",
		ThreadID:   "thread-1",
		RequestID:  "req-1",
		Iteration:  1,
	}
	if err := logger.LogHITLEvent(req); err != nil {
		t.Fatalf("LogHITLEvent request error: %v", err)
	}
	dec := HITLAuditEntry{
		Event:      "hitl_decision",
		Tool:       "execute",
		ToolCallID: "tc-1",
		ThreadID:   "thread-1",
		RequestID:  "req-1",
		Iteration:  1,
		Decision:   "approved",
		Approved:   true,
	}
	if err := logger.LogHITLEvent(dec); err != nil {
		t.Fatalf("LogHITLEvent decision error: %v", err)
	}

	if err := VerifyHITLAuditFileChainStrict(path); err != nil {
		t.Fatalf("expected strict verification success, got: %v", err)
	}
}

func TestGraphLogsHITLAuditEntries(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"command": "pwd"})
	llm := &streamScriptedLLM{
		steps: []struct {
			content string
			calls   []llms.ToolCall
		}{
			{
				calls: []llms.ToolCall{
					{ID: "call-123", Name: "execute", Arguments: string(args)},
				},
			},
			{content: "done"},
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
			return "ok", nil
		},
	)

	capture := &captureHITLAuditLogger{}
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
				return "approved", nil
			},
		),
		&NoOpLogger{},
	)
	g.hitlAudit = capture

	if _, err := g.Run(context.Background(), Input{
		ThreadID: "thread-audit",
		Messages: []Message{{Role: "user", Content: "run execute"}},
	}); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if len(capture.entries) < 2 {
		t.Fatalf("expected at least 2 hitl audit entries, got %d", len(capture.entries))
	}
	if capture.entries[0].Event != "hitl_request" {
		t.Fatalf("expected first audit event to be request, got: %+v", capture.entries[0])
	}
	if capture.entries[0].ArgsHash == "" {
		t.Fatalf("expected args hash in hitl_request entry")
	}
	if capture.entries[0].Arguments != nil {
		t.Fatalf("expected arguments to be redacted by default, got: %+v", capture.entries[0].Arguments)
	}
	if capture.entries[1].Event != "hitl_decision" || !capture.entries[1].Approved {
		t.Fatalf("expected second audit event to be approved decision, got: %+v", capture.entries[1])
	}
}

func TestGraphLogsHITLAuditArgumentsWhenEnabled(t *testing.T) {
	args, _ := json.Marshal(map[string]any{"command": "pwd"})
	llm := &streamScriptedLLM{
		steps: []struct {
			content string
			calls   []llms.ToolCall
		}{
			{
				calls: []llms.ToolCall{
					{ID: "call-456", Name: "execute", Arguments: string(args)},
				},
			},
			{content: "done"},
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
			return "ok", nil
		},
	)

	capture := &captureHITLAuditLogger{}
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
				return "approved", nil
			},
		),
		&NoOpLogger{},
	)
	g.hitlAudit = capture
	g.hitlAuditIncludeArgs = true

	if _, err := g.Run(context.Background(), Input{
		ThreadID: "thread-audit-args",
		Messages: []Message{{Role: "user", Content: "run execute"}},
	}); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(capture.entries) == 0 {
		t.Fatalf("expected hitl audit entries")
	}
	req := capture.entries[0]
	if req.Event != "hitl_request" {
		t.Fatalf("expected first event hitl_request, got: %+v", req)
	}
	argsMap, ok := req.Arguments.(map[string]any)
	if !ok {
		t.Fatalf("expected decoded arguments map, got: %T (%+v)", req.Arguments, req.Arguments)
	}
	if argsMap["command"] != "pwd" {
		t.Fatalf("unexpected arguments payload: %+v", argsMap)
	}
}
