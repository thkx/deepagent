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
}

// Logger is the interface for tool call logging
type Logger interface {
	LogToolCall(event *ToolCallEvent)
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
}

// NoOpLogger discards all log events
type NoOpLogger struct{}

func (nl *NoOpLogger) LogToolCall(event *ToolCallEvent) {
	// Discard logs
}

// ContextLogger is a logger that stores events in context for retrieval
type ContextLogger struct {
	events []*ToolCallEvent
}

func (cl *ContextLogger) LogToolCall(event *ToolCallEvent) {
	if event != nil {
		cl.events = append(cl.events, event)
	}
}

func (cl *ContextLogger) Events() []*ToolCallEvent {
	return cl.events
}

// NewContextLogger creates a new context logger
func NewContextLogger() *ContextLogger {
	return &ContextLogger{
		events: make([]*ToolCallEvent, 0),
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
