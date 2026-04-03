package agent

import (
	"context"
	"fmt"
	"strings"
)

type InterruptConfig map[string]bool

type ApproverFunc func(ctx context.Context, toolName string, args any) (string, error)

type approvalMetadata struct {
	ThreadID   string `json:"thread_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Iteration  int    `json:"iteration,omitempty"`
}

type approvalMetadataContextKey struct{}

type HumanInTheLoop struct {
	config   InterruptConfig
	approver ApproverFunc
}

func NewHumanInTheLoop(config InterruptConfig) *HumanInTheLoop {
	if config == nil {
		config = make(InterruptConfig)
	}
	return &HumanInTheLoop{config: config, approver: consoleApprover}
}

func NewHumanInTheLoopWithApprover(config InterruptConfig, approver ApproverFunc) *HumanInTheLoop {
	if config == nil {
		config = make(InterruptConfig)
	}
	if approver == nil {
		approver = consoleApprover
	}
	return &HumanInTheLoop{config: config, approver: approver}
}

func (h *HumanInTheLoop) ShouldInterrupt(toolName string) bool {
	return h.config[toolName]
}

// 真实等待人工输入（简化演示，生产中替换为 WebSocket/HTTP API）
func (h *HumanInTheLoop) WaitForApproval(ctx context.Context, toolName string, args any) (string, error) {
	return h.approver(ctx, toolName, args)
}

func contextWithApprovalMetadata(ctx context.Context, meta approvalMetadata) context.Context {
	return context.WithValue(ctx, approvalMetadataContextKey{}, meta)
}

func approvalMetadataFromContext(ctx context.Context) (approvalMetadata, bool) {
	meta, ok := ctx.Value(approvalMetadataContextKey{}).(approvalMetadata)
	return meta, ok
}

func consoleApprover(ctx context.Context, toolName string, args any) (string, error) {
	fmt.Printf("\n[Human-in-the-loop] Tool '%s' wants to run with args: %+v\n", toolName, args)
	fmt.Print("Approve? (y/n/modify): ")
	// keep compatibility with console input while allowing cancellation.
	responseCh := make(chan string, 1)
	go func() {
		var response string
		_, _ = fmt.Scanln(&response)
		responseCh <- response
	}()
	select {
	case <-ctx.Done():
		return "cancelled", ctx.Err()
	case response := <-responseCh:
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return "approved", nil
		}
		if response == "modify" {
			return "modify", fmt.Errorf("modify is not implemented yet")
		}
		return "rejected", fmt.Errorf("user rejected tool call")
	}
}
