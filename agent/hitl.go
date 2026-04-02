package agent

import (
	"context"
	"fmt"
)

type InterruptConfig map[string]bool

type HumanInTheLoop struct {
	config   InterruptConfig
	approver func(ctx context.Context, toolName string, args any) (string, error)
}

func NewHumanInTheLoop(config InterruptConfig) *HumanInTheLoop {
	if config == nil {
		config = make(InterruptConfig)
	}
	return &HumanInTheLoop{config: config, approver: consoleApprover}
}

func NewHumanInTheLoopWithApprover(config InterruptConfig, approver func(ctx context.Context, toolName string, args any) (string, error)) *HumanInTheLoop {
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

func consoleApprover(ctx context.Context, toolName string, args any) (string, error) {
	_ = ctx
	fmt.Printf("\n[Human-in-the-loop] Tool '%s' wants to run with args: %+v\n", toolName, args)
	fmt.Print("Approve? (y/n/modify): ")
	// 这里可读取 stdin 或通过外部接口等待
	var response string
	fmt.Scanln(&response)
	if response == "y" || response == "yes" {
		return "approved", nil
	}
	return "rejected", fmt.Errorf("user rejected tool call")
}
