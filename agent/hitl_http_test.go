package agent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHTTPApproverApproved(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token-x" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"decision":"approved"}`))
	}))
	defer srv.Close()

	approver, err := NewHTTPApprover(HTTPApproverConfig{
		URL:    srv.URL,
		APIKey: "token-x",
	})
	if err != nil {
		t.Fatalf("unexpected NewHTTPApprover error: %v", err)
	}

	decision, err := approver(context.Background(), "execute", map[string]any{"command": "pwd"})
	if err != nil {
		t.Fatalf("expected approved decision, got err: %v", err)
	}
	if decision != "approved" {
		t.Fatalf("unexpected decision: %s", decision)
	}
}

func TestNewHTTPApproverRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"decision":"rejected","reason":"denied by policy"}`))
	}))
	defer srv.Close()

	approver, err := NewHTTPApprover(HTTPApproverConfig{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected NewHTTPApprover error: %v", err)
	}

	decision, err := approver(context.Background(), "execute", map[string]any{"command": "pwd"})
	if err == nil {
		t.Fatalf("expected rejection error")
	}
	if decision != "rejected" {
		t.Fatalf("unexpected decision: %s", decision)
	}
	if !strings.Contains(err.Error(), "denied by policy") {
		t.Fatalf("unexpected rejection error: %v", err)
	}
}

func TestNewHTTPApproverIncludesApprovalMetadata(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("failed to decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"decision":"approved"}`))
	}))
	defer srv.Close()

	approver, err := NewHTTPApprover(HTTPApproverConfig{URL: srv.URL})
	if err != nil {
		t.Fatalf("unexpected NewHTTPApprover error: %v", err)
	}

	ctx := contextWithApprovalMetadata(context.Background(), approvalMetadata{
		ThreadID:   "thread-1",
		RequestID:  "req-1",
		ToolCallID: "call-1",
		Iteration:  3,
	})
	_, err = approver(ctx, "execute", map[string]any{"command": "pwd"})
	if err != nil {
		t.Fatalf("expected approval, got err: %v", err)
	}

	if captured["thread_id"] != "thread-1" {
		t.Fatalf("missing thread_id in request body: %+v", captured)
	}
	if captured["request_id"] != "req-1" {
		t.Fatalf("missing request_id in request body: %+v", captured)
	}
	if captured["tool_call_id"] != "call-1" {
		t.Fatalf("missing tool_call_id in request body: %+v", captured)
	}
}
