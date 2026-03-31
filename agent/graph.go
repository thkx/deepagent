package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type Graph struct {
	llm          llms.ChatModel
	tools        map[string]tools.Tool // 工具注册表
	prompt       string
	backend      fs.Backend
	checkpointer Checkpointer
	memory       memory.Store
	hitl         *HumanInTheLoop
}

func buildGraph(llm llms.ChatModel, toolList []tools.Tool, prompt string, backend fs.Backend, cp Checkpointer, mem memory.Store, hitl *HumanInTheLoop) *Graph {
	toolMap := make(map[string]tools.Tool)
	for _, t := range toolList {
		toolMap[t.Name()] = t
	}
	return &Graph{
		llm:          llm,
		tools:        toolMap,
		prompt:       prompt,
		backend:      backend,
		checkpointer: cp,
		memory:       mem,
		hitl:         hitl,
	}
}

func (g *Graph) Run(ctx context.Context, input Input) (Output, error) {
	// 1. 尝试从 checkpointer 恢复
	state, exists, err := g.checkpointer.Load(input.ThreadID)
	if err != nil {
		return nil, err
	}
	if !exists {
		state = State{
			Messages: []Message{{Role: "system", Content: g.prompt}},
			ThreadID: input.ThreadID,
		}
		for _, m := range input.Messages {
			state.Messages = append(state.Messages, m)
		}
	}

	// 2. ReAct 循环（生产级：最多 20 轮 + 中断支持）
	for state.Iteration < 20 {
		state.Iteration++

		// 调用 LLM（带 tool 定义）
		content, toolCalls, err := g.llm.Invoke(ctx, convertToLLMMessages(state.Messages), nil)
		if err != nil {
			return nil, err
		}

		// 无 tool call → 最终回答
		if len(toolCalls) == 0 && content != "" {
			state.Messages = append(state.Messages, Message{Role: "assistant", Content: content})
			state.Final = content
			_ = g.checkpointer.Save(input.ThreadID, state) // 持久化
			return Output{"messages": state.Messages, "final": state.Final}, nil
		}

		// 3. ✅ 完整 tool calling 解析与执行
		state.Messages = append(state.Messages, Message{Role: "assistant", Content: content})
		for _, tc := range toolCalls {
			tool, ok := g.tools[tc.Name]
			if !ok {
				continue
			}

			// Human-in-the-loop 检查
			if g.hitl.ShouldInterrupt(tc.Name) {
				_, _ = g.hitl.WaitForApproval(ctx, tc.Name, tc.Arguments)
			}

			// 解析参数
			var args map[string]any
			if tc.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Arguments), &args)
			}

			// 执行工具
			result, callErr := tool.Call(ctx, args)
			if callErr != nil {
				result = fmt.Sprintf("Tool error: %v", callErr)
			}

			// 追加 tool 结果（标准格式）
			state.Messages = append(state.Messages, Message{
				Role:    "tool",
				Content: fmt.Sprintf("Tool %s result: %+v", tc.Name, result),
			})

			// 长时记忆保存（示例）
			_ = g.memory.Put(ctx, input.ThreadID, "last_tool_result", result)
		}

		// 每轮后持久化
		if err := g.checkpointer.Save(input.ThreadID, state); err != nil {
			fmt.Printf("Checkpoint save warning: %v\n", err)
		}
	}

	return nil, fmt.Errorf("max iterations reached")
}

func convertToLLMMessages(msgs []Message) []llms.ChatMessage {
	var res []llms.ChatMessage
	for _, m := range msgs {
		res = append(res, llms.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return res
}
