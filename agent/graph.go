package agent

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/thkx/deepagent/llms"
	"github.com/thkx/deepagent/memory"
	"github.com/thkx/deepagent/tools"
	"github.com/thkx/deepagent/tools/builtin/fs"
)

type Graph struct {
	llm                  llms.ChatModel
	tools                map[string]tools.Tool // 工具注册表
	prompt               string
	backend              fs.Backend
	checkpointer         Checkpointer
	memory               memory.Store
	hitl                 *HumanInTheLoop
	logger               Logger
	eventSink            func(Event)
	hitlAudit            HITLAuditLogger
	hitlAuditIncludeArgs bool
}

func buildGraph(llm llms.ChatModel, toolList []tools.Tool, prompt string, backend fs.Backend, cp Checkpointer, mem memory.Store, hitl *HumanInTheLoop, logger Logger) *Graph {
	toolMap := make(map[string]tools.Tool)
	for _, t := range toolList {
		toolMap[t.Name()] = t
	}
	if logger == nil {
		// Default to JSON logger when DEEPAGENT_JSON_LOG=true, otherwise simple stdout logger
		if os.Getenv("DEEPAGENT_JSON_LOG") == "true" {
			// Optionally enable Prometheus metrics collector via DEEPAGENT_PROMETHEUS=true
			var metricsCollector MetricsCollector
			if os.Getenv("DEEPAGENT_PROMETHEUS") == "true" {
				metricsCollector = NewPrometheusCollector(nil)
			}
			logger = NewJSONLogger(metricsCollector)
		} else {
			logger = &SimpleLogger{}
		}
	}
	return &Graph{
		llm:          llm,
		tools:        toolMap,
		prompt:       prompt,
		backend:      backend,
		checkpointer: cp,
		memory:       mem,
		hitl:         hitl,
		logger:       logger,
	}
}

func (g *Graph) Run(ctx context.Context, input Input) (out Output, runErr error) {
	startedAt := time.Now()
	requestID := newRequestID(input.ThreadID)
	modelCalls := 0
	modelErrors := 0
	toolCallCount := 0
	toolErrors := 0
	iteration := 0
	defer func() {
		toolErrorRate := 0.0
		if toolCallCount > 0 {
			toolErrorRate = float64(toolErrors) / float64(toolCallCount)
		}
		errMsg := ""
		if runErr != nil {
			errMsg = runErr.Error()
		}
		g.logger.LogRunSummary(&RunSummaryEvent{
			ThreadID:      input.ThreadID,
			RequestID:     requestID,
			Duration:      time.Since(startedAt),
			Iterations:    iteration,
			ModelCalls:    modelCalls,
			ModelErrors:   modelErrors,
			ToolCalls:     toolCallCount,
			ToolErrors:    toolErrors,
			ToolErrorRate: toolErrorRate,
			Error:         errMsg,
		})
		if runErr != nil {
			if ae, ok := runErr.(*AgentError); ok {
				g.emit(Event{Type: "error", Content: ae.Payload()})
			} else {
				g.emit(Event{
					Type: "error",
					Content: map[string]any{
						"message": runErr.Error(),
					},
				})
			}
		}
	}()

	// 1. 尝试从 checkpointer 恢复
	state, exists, err := g.checkpointer.Load(input.ThreadID)
	if err != nil {
		runErr = wrapRunError(ErrCodeCheckpointLoad, "failed to load checkpoint", input.ThreadID, requestID, err)
		return nil, runErr
	}
	if !exists {
		state = State{
			Messages: []Message{{Role: "system", Content: g.prompt}},
			ThreadID: input.ThreadID,
		}
	}
	for _, m := range input.Messages {
		state.Messages = append(state.Messages, m)
	}

	// 2. ReAct 循环（生产级：最多 20 轮 + 中断支持）
	for state.Iteration < 20 {
		state.Iteration++
		iteration = state.Iteration

		// 调用 LLM（带 tool 定义）
		toolList := convertToolsToLLMFormat(g.tools)
		modelStart := time.Now()
		modelCalls++
		content, toolCalls, err := g.llm.Invoke(ctx, convertToLLMMessages(state.Messages), toolList)
		modelDuration := time.Since(modelStart)
		if err != nil {
			modelErrors++
		}
		g.logger.LogModelCall(&ModelCallEvent{
			Model:        fmt.Sprintf("%T", g.llm),
			MessageCount: len(state.Messages),
			ToolCount:    len(toolList),
			Duration:     modelDuration,
			Timestamp:    modelStart,
			ThreadID:     input.ThreadID,
			RequestID:    requestID,
			Error:        err,
		})
		g.emit(Event{
			Type: "model_call",
			Content: map[string]any{
				"model":      fmt.Sprintf("%T", g.llm),
				"duration_s": modelDuration.Seconds(),
				"iteration":  state.Iteration,
				"error":      err != nil,
			},
		})
		if err != nil {
			runErr = wrapRunError(ErrCodeLLMInvoke, "failed to invoke llm", input.ThreadID, requestID, err)
			return nil, runErr
		}

		// 无 tool call → 最终回答
		if len(toolCalls) == 0 && content != "" {
			state.Messages = append(state.Messages, Message{Role: "assistant", Content: content})
			g.emit(Event{
				Type: "message",
				Content: map[string]any{
					"role":      "assistant",
					"content":   content,
					"iteration": state.Iteration,
				},
			})
			state.Final = content
			_ = g.checkpointer.Save(input.ThreadID, state) // 持久化
			out = Output{
				"messages":   state.Messages,
				"final":      state.Final,
				"thread_id":  input.ThreadID,
				"request_id": requestID,
			}
			g.emit(Event{Type: "final", Content: out})
			return out, nil
		}

		// 3. ✅ 完整 tool calling 解析与执行
		state.Messages = append(state.Messages, Message{Role: "assistant", Content: content})
		if content != "" {
			g.emit(Event{
				Type: "message",
				Content: map[string]any{
					"role":      "assistant",
					"content":   content,
					"iteration": state.Iteration,
				},
			})
		}
		for _, tc := range toolCalls {
			tool, ok := g.tools[tc.Name]
			if !ok {
				continue
			}

			g.emit(Event{
				Type: "tool_call",
				Content: map[string]any{
					"tool":      tc.Name,
					"arguments": tc.Arguments,
					"iteration": state.Iteration,
				},
			})

			// Human-in-the-loop 检查
			if g.hitl.ShouldInterrupt(tc.Name) {
				g.emit(Event{
					Type: "hitl_request",
					Content: map[string]any{
						"tool":         tc.Name,
						"tool_call_id": tc.ID,
						"iteration":    state.Iteration,
						"thread_id":    input.ThreadID,
						"request_id":   requestID,
					},
				})
				g.logHITLAudit(HITLAuditEntry{
					Event:      "hitl_request",
					Tool:       tc.Name,
					ToolCallID: tc.ID,
					ThreadID:   input.ThreadID,
					RequestID:  requestID,
					Iteration:  state.Iteration,
					Arguments:  g.auditArgsPayload(tc.Arguments),
					ArgsHash:   hashString(tc.Arguments),
				})
				approvalCtx := contextWithApprovalMetadata(ctx, approvalMetadata{
					ThreadID:   input.ThreadID,
					RequestID:  requestID,
					ToolCallID: tc.ID,
					Iteration:  state.Iteration,
				})
				decision, approvalErr := g.hitl.WaitForApproval(approvalCtx, tc.Name, tc.Arguments)
				if approvalErr != nil {
					toolErrors++
					g.emit(Event{
						Type: "hitl_decision",
						Content: map[string]any{
							"tool":         tc.Name,
							"tool_call_id": tc.ID,
							"decision":     decision,
							"approved":     false,
							"error":        approvalErr.Error(),
							"thread_id":    input.ThreadID,
							"request_id":   requestID,
							"iteration":    state.Iteration,
						},
					})
					g.logHITLAudit(HITLAuditEntry{
						Event:      "hitl_decision",
						Tool:       tc.Name,
						ToolCallID: tc.ID,
						ThreadID:   input.ThreadID,
						RequestID:  requestID,
						Iteration:  state.Iteration,
						Decision:   decision,
						Approved:   false,
						Error:      approvalErr.Error(),
					})
					rejectedResult := ToolResult{
						Tool:       tc.Name,
						ToolCallID: tc.ID,
						OK:         false,
						Error:      approvalErr.Error(),
						Code:       ToolCodeHITLRejected,
					}
					state.Messages = append(state.Messages, Message{Role: "tool", Content: formatToolMessage(rejectedResult)})
					continue
				}
				g.emit(Event{
					Type: "hitl_decision",
					Content: map[string]any{
						"tool":         tc.Name,
						"tool_call_id": tc.ID,
						"decision":     decision,
						"approved":     true,
						"thread_id":    input.ThreadID,
						"request_id":   requestID,
						"iteration":    state.Iteration,
					},
				})
				g.logHITLAudit(HITLAuditEntry{
					Event:      "hitl_decision",
					Tool:       tc.Name,
					ToolCallID: tc.ID,
					ThreadID:   input.ThreadID,
					RequestID:  requestID,
					Iteration:  state.Iteration,
					Decision:   decision,
					Approved:   true,
				})
			}

			// 解析参数
			var args map[string]any
			if tc.Arguments != "" {
				_ = json.Unmarshal([]byte(tc.Arguments), &args)
			}

			// 执行工具（先验证参数，再记录时间）
			startTime := time.Now()

			// Generate parameters schema from tool definition and validate
			schema := llms.GenerateParametersSchema(llms.Tool{Name: tool.Name(), Description: tool.Description(), Parameters: tool.Parameters()})
			var result any
			var callErr error
			// Prefer full JSON Schema validation when available
			if err := tools.ValidateAgainstJSONSchema(schema, args); err != nil {
				callErr = err
			} else {
				result, callErr = tool.Call(ctx, args)
			}

			duration := time.Since(startTime)
			toolCallCount++
			if callErr != nil {
				toolErrors++
			}

			// 记录工具调用
			g.logger.LogToolCall(&ToolCallEvent{
				Tool:      tc.Name,
				Args:      args,
				Result:    result,
				Error:     callErr,
				Duration:  duration,
				Timestamp: startTime,
				ThreadID:  input.ThreadID,
				RequestID: requestID,
			})

			toolResult := ToolResult{
				Tool:       tc.Name,
				ToolCallID: tc.ID,
				OK:         callErr == nil,
			}
			g.emit(Event{Type: "tool_result", Content: toolResult})
			if callErr != nil {
				toolResult.Error = callErr.Error()
				toolResult.Code = ToolCodeExecutionError
			} else {
				toolResult.Data = result
			}

			// 追加 tool 结果（标准格式）
			state.Messages = append(state.Messages, Message{
				Role:    "tool",
				Content: formatToolMessage(toolResult),
			})

			// 长时记忆保存（示例）
			_ = g.memory.Put(ctx, input.ThreadID, "last_tool_result", toolResult)
		}

		// 每轮后持久化
		if err := g.checkpointer.Save(input.ThreadID, state); err != nil {
			fmt.Printf("Checkpoint save warning: %v\n", err)
		}
	}

	runErr = wrapRunError(ErrCodeMaxIterations, "max iterations reached", input.ThreadID, requestID, fmt.Errorf("max iterations reached"))
	return nil, runErr
}

func (g *Graph) emit(event Event) {
	if g.eventSink == nil {
		return
	}
	g.eventSink(event)
}

func (g *Graph) runWithEventSink(ctx context.Context, input Input, sink func(Event)) (Output, error) {
	cloned := *g
	cloned.eventSink = sink
	return cloned.Run(ctx, input)
}

func (g *Graph) logHITLAudit(entry HITLAuditEntry) {
	if g.hitlAudit == nil {
		return
	}
	_ = g.hitlAudit.LogHITLEvent(entry)
}

func (g *Graph) auditArgsPayload(raw string) any {
	if !g.hitlAuditIncludeArgs {
		return nil
	}
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
		return parsed
	}
	return raw
}

func hashString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func convertToLLMMessages(msgs []Message) []llms.ChatMessage {
	var res []llms.ChatMessage
	for _, m := range msgs {
		res = append(res, llms.ChatMessage{Role: m.Role, Content: m.Content})
	}
	return res
}

// convertToolsToLLMFormat converts internal tools map to LLM format with parameters
func convertToolsToLLMFormat(toolMap map[string]tools.Tool) []llms.Tool {
	var res []llms.Tool
	for _, tool := range toolMap {
		res = append(res, llms.Tool{
			Name:        tool.Name(),
			Description: tool.Description(),
			Parameters:  tool.Parameters(),
		})
	}
	return res
}

func formatToolMessage(result ToolResult) string {
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf(`{"tool":"%s","ok":false,"error":"%s","code":"%s"}`, result.Tool, strings.ReplaceAll(err.Error(), `"`, `'`), ToolCodeSerialization)
	}
	return string(payload)
}

func newRequestID(threadID string) string {
	prefix := "run"
	if threadID != "" {
		prefix = threadID
	}
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func wrapRunError(code, message, threadID, requestID string, cause error) error {
	return &AgentError{
		Code:      code,
		Message:   message,
		ThreadID:  threadID,
		RequestID: requestID,
		Cause:     cause,
	}
}
