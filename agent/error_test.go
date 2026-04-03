package agent

import (
	"errors"
	"testing"
)

func TestAgentErrorPayload(t *testing.T) {
	err := &AgentError{
		Code:      ErrCodeLLMInvoke,
		Message:   "failed to invoke llm",
		ThreadID:  "thread-1",
		RequestID: "req-1",
	}
	payload := err.Payload()
	if payload["code"] != ErrCodeLLMInvoke {
		t.Fatalf("unexpected code payload: %v", payload["code"])
	}
	if payload["thread_id"] != "thread-1" {
		t.Fatalf("unexpected thread payload: %v", payload["thread_id"])
	}
	if payload["request_id"] != "req-1" {
		t.Fatalf("unexpected request payload: %v", payload["request_id"])
	}
}

func TestWrapRunError(t *testing.T) {
	cause := errors.New("boom")
	err := wrapRunError(ErrCodeCheckpointLoad, "failed to load checkpoint", "thread-x", "req-x", cause)
	ae, ok := err.(*AgentError)
	if !ok {
		t.Fatalf("expected *AgentError")
	}
	if ae.Code != ErrCodeCheckpointLoad || ae.ThreadID != "thread-x" || ae.RequestID != "req-x" {
		t.Fatalf("unexpected wrapped error payload: %+v", ae)
	}
	if !errors.Is(ae, cause) {
		t.Fatalf("expected wrapped cause")
	}
}
