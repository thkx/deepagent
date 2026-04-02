package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type HTTPApproverConfig struct {
	URL        string
	APIKey     string
	Timeout    time.Duration
	Headers    map[string]string
	HTTPClient *http.Client
}

type hitlApprovalRequest struct {
	Tool       string `json:"tool"`
	Args       any    `json:"args"`
	ThreadID   string `json:"thread_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Iteration  int    `json:"iteration,omitempty"`
}

type hitlApprovalResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

// NewHTTPApprover creates an approver that calls an external HTTP service.
// The service should return JSON: {"decision":"approved|rejected|modify","reason":"..."}.
func NewHTTPApprover(cfg HTTPApproverConfig) (ApproverFunc, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, fmt.Errorf("hitl approver url is required")
	}

	client := cfg.HTTPClient
	if client == nil {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	return func(ctx context.Context, toolName string, args any) (string, error) {
		meta, _ := approvalMetadataFromContext(ctx)
		body, err := json.Marshal(hitlApprovalRequest{
			Tool:       toolName,
			Args:       args,
			ThreadID:   meta.ThreadID,
			RequestID:  meta.RequestID,
			ToolCallID: meta.ToolCallID,
			Iteration:  meta.Iteration,
		})
		if err != nil {
			return "rejected", fmt.Errorf("failed to marshal hitl approval request: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return "rejected", fmt.Errorf("failed to create hitl approval request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if strings.TrimSpace(cfg.APIKey) != "" {
			req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(cfg.APIKey))
		}
		for k, v := range cfg.Headers {
			if strings.TrimSpace(k) != "" {
				req.Header.Set(k, v)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			return "rejected", fmt.Errorf("hitl approver request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return "rejected", fmt.Errorf("hitl approver returned non-success status: %s", resp.Status)
		}

		var out hitlApprovalResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return "rejected", fmt.Errorf("invalid hitl approver response: %w", err)
		}

		decision := strings.ToLower(strings.TrimSpace(out.Decision))
		switch decision {
		case "approved", "approve", "ok", "yes", "y":
			return "approved", nil
		case "modify":
			msg := strings.TrimSpace(out.Reason)
			if msg == "" {
				msg = "modify is not implemented yet"
			}
			return "modify", errors.New(msg)
		case "rejected", "reject", "no", "n":
			msg := strings.TrimSpace(out.Reason)
			if msg == "" {
				msg = "tool call rejected by hitl approver"
			}
			return "rejected", errors.New(msg)
		default:
			return "rejected", fmt.Errorf("unknown hitl decision: %q", out.Decision)
		}
	}, nil
}
