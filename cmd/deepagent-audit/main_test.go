package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thkx/deepagent/agent"
)

func TestRunRequiresFilePath(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run(nil, &out, &errOut, func(key string) string { return "" })
	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "audit file path is required") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestRunReturnsNonZeroOnInvalidChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	// invalid hash chain on purpose
	content := `{"timestamp":"2026-01-01T00:00:00Z","event":"hitl_request","hash":"bad","prev_hash":""}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"-file", path}, &out, &errOut, func(key string) string { return "" })
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "audit chain verification failed") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestRunReturnsZeroOnValidChain(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ok.jsonl")
	logger, err := agent.NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	if err := logger.LogHITLEvent(agent.HITLAuditEntry{Event: "hitl_request", Tool: "execute"}); err != nil {
		t.Fatalf("log event: %v", err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"-file", path}, &out, &errOut, func(key string) string { return "" })
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d, stderr=%s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "verification passed") {
		t.Fatalf("unexpected stdout: %s", out.String())
	}
}

func TestRunStrictReturnsNonZeroOnMissingRequiredFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "minimal.jsonl")
	logger, err := agent.NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	if err := logger.LogHITLEvent(agent.HITLAuditEntry{Event: "hitl_request", Tool: "execute"}); err != nil {
		t.Fatalf("log event: %v", err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"-file", path, "-strict"}, &out, &errOut, func(key string) string { return "" })
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "strict validation failed") {
		t.Fatalf("unexpected stderr: %s", errOut.String())
	}
}

func TestRunStrictReturnsZeroOnCompleteEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "strict_ok.jsonl")
	logger, err := agent.NewFileHITLAuditLogger(path)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	if err := logger.LogHITLEvent(agent.HITLAuditEntry{
		Event:      "hitl_request",
		Tool:       "execute",
		ToolCallID: "tc-1",
		ThreadID:   "thread-1",
		RequestID:  "req-1",
		Iteration:  1,
	}); err != nil {
		t.Fatalf("log request event: %v", err)
	}
	if err := logger.LogHITLEvent(agent.HITLAuditEntry{
		Event:      "hitl_decision",
		Tool:       "execute",
		ToolCallID: "tc-1",
		ThreadID:   "thread-1",
		RequestID:  "req-1",
		Iteration:  1,
		Decision:   "approved",
		Approved:   true,
	}); err != nil {
		t.Fatalf("log decision event: %v", err)
	}

	var out, errOut bytes.Buffer
	code := run([]string{"-file", path, "-strict"}, &out, &errOut, func(key string) string { return "" })
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, errOut.String())
	}
}
