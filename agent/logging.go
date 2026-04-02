package agent

import (
	"context"
	"fmt"
	"time"
)

// ToolCallEvent represents a single tool invocation event for logging/auditing
type ToolCallEvent struct {
	Tool      string
	Args      map[string]any
	Result    any
	Error     error
	Duration  time.Duration
	Timestamp time.Time
	ThreadID  string
	RequestID string
}

// ModelCallEvent represents a single model invocation.
type ModelCallEvent struct {
	Model        string
	MessageCount int
	ToolCount    int
	Duration     time.Duration
	Timestamp    time.Time
	ThreadID     string
	RequestID    string
	Error        error
}

// RunSummaryEvent represents run-level metrics for one Invoke call.
type RunSummaryEvent struct {
	ThreadID      string
	RequestID     string
	Duration      time.Duration
	Iterations    int
	ModelCalls    int
	ModelErrors   int
	ToolCalls     int
	ToolErrors    int
	ToolErrorRate float64
	Error         string
}

// Logger is the interface for tool call logging
type Logger interface {
	LogToolCall(event *ToolCallEvent)
	LogModelCall(event *ModelCallEvent)
	LogRunSummary(event *RunSummaryEvent)
}

// SimpleLogger is a basic logger that prints to stdout
type SimpleLogger struct{}

func (sl *SimpleLogger) LogToolCall(event *ToolCallEvent) {
	if event == nil {
		return
	}

	status := "success"
	if event.Error != nil {
		status = "error"
	}

	fmt.Printf("[%s] TOOL CALL %s: %s (took %v)\n",
		event.Timestamp.Format("2006-01-02T15:04:05"),
		event.Tool,
		status,
		event.Duration,
	)

	if event.Error != nil {
		fmt.Printf("  Error: %v\n", event.Error)
	}

	fmt.Printf("  Args: %v\n", event.Args)
	fmt.Printf("  Result: %v\n", event.Result)
	fmt.Printf("  ThreadID: %s\n", event.ThreadID)
	fmt.Printf("  RequestID: %s\n", event.RequestID)
}

func (sl *SimpleLogger) LogModelCall(event *ModelCallEvent) {
	if event == nil {
		return
	}
	status := "success"
	if event.Error != nil {
		status = "error"
	}
	fmt.Printf("[%s] MODEL CALL %s: %s (messages=%d tools=%d took %v)\n",
		event.Timestamp.Format("2006-01-02T15:04:05"),
		event.Model,
		status,
		event.MessageCount,
		event.ToolCount,
		event.Duration,
	)
	if event.Error != nil {
		fmt.Printf("  Error: %v\n", event.Error)
	}
	fmt.Printf("  ThreadID: %s\n", event.ThreadID)
	fmt.Printf("  RequestID: %s\n", event.RequestID)
}

func (sl *SimpleLogger) LogRunSummary(event *RunSummaryEvent) {
	if event == nil {
		return
	}
	fmt.Printf("[RUN SUMMARY] thread=%s request=%s duration=%v iterations=%d model_calls=%d model_errors=%d tool_calls=%d tool_errors=%d tool_error_rate=%.2f%%\n",
		event.ThreadID,
		event.RequestID,
		event.Duration,
		event.Iterations,
		event.ModelCalls,
		event.ModelErrors,
		event.ToolCalls,
		event.ToolErrors,
		event.ToolErrorRate*100,
	)
	if event.Error != "" {
		fmt.Printf("  Error: %s\n", event.Error)
	}
}

// NoOpLogger discards all log events
type NoOpLogger struct{}

func (nl *NoOpLogger) LogToolCall(event *ToolCallEvent) {
	// Discard logs
}

func (nl *NoOpLogger) LogModelCall(event *ModelCallEvent) {
	// Discard logs
}

func (nl *NoOpLogger) LogRunSummary(event *RunSummaryEvent) {
	// Discard logs
}

// ContextLogger is a logger that stores events in context for retrieval
type ContextLogger struct {
	toolEvents   []*ToolCallEvent
	modelEvents  []*ModelCallEvent
	runSummaries []*RunSummaryEvent
}

func (cl *ContextLogger) LogToolCall(event *ToolCallEvent) {
	if event != nil {
		cl.toolEvents = append(cl.toolEvents, event)
	}
}

func (cl *ContextLogger) LogModelCall(event *ModelCallEvent) {
	if event != nil {
		cl.modelEvents = append(cl.modelEvents, event)
	}
}

func (cl *ContextLogger) LogRunSummary(event *RunSummaryEvent) {
	if event != nil {
		cl.runSummaries = append(cl.runSummaries, event)
	}
}

func (cl *ContextLogger) ToolEvents() []*ToolCallEvent {
	return cl.toolEvents
}

func (cl *ContextLogger) ModelEvents() []*ModelCallEvent {
	return cl.modelEvents
}

func (cl *ContextLogger) RunSummaries() []*RunSummaryEvent {
	return cl.runSummaries
}

// NewContextLogger creates a new context logger
func NewContextLogger() *ContextLogger {
	return &ContextLogger{
		toolEvents:   make([]*ToolCallEvent, 0),
		modelEvents:  make([]*ModelCallEvent, 0),
		runSummaries: make([]*RunSummaryEvent, 0),
	}
}

// LoggerFromContext retrieves logger from context, or returns NoOpLogger if not found
func LoggerFromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value("logger").(Logger); ok {
		return logger
	}
	return &NoOpLogger{}
}

// ContextWithLogger adds a logger to the context
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, "logger", logger)
}
