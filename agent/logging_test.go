package agent

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestContextLogger(t *testing.T) {
	logger := NewContextLogger()

	event1 := &ToolCallEvent{
		Tool:      "tool1",
		Args:      map[string]any{"arg": "value"},
		Result:    "result1",
		Error:     nil,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
		ThreadID:  "thread1",
	}

	event2 := &ToolCallEvent{
		Tool:      "tool2",
		Args:      map[string]any{},
		Result:    nil,
		Error:     nil,
		Duration:  50 * time.Millisecond,
		Timestamp: time.Now(),
		ThreadID:  "thread1",
	}

	logger.LogToolCall(event1)
	logger.LogToolCall(event2)

	events := logger.Events()
	if len(events) != 2 {
		t.Errorf("ContextLogger.Events() expected 2 events, got %d", len(events))
	}

	if events[0].Tool != "tool1" {
		t.Errorf("ContextLogger.Events() expected first tool to be 'tool1', got %v", events[0].Tool)
	}

	if events[1].Tool != "tool2" {
		t.Errorf("ContextLogger.Events() expected second tool to be 'tool2', got %v", events[1].Tool)
	}
}

func TestNoOpLogger(t *testing.T) {
	logger := &NoOpLogger{}

	event := &ToolCallEvent{
		Tool:      "test",
		Args:      map[string]any{},
		Result:    "result",
		Error:     nil,
		Duration:  10 * time.Millisecond,
		Timestamp: time.Now(),
		ThreadID:  "thread1",
	}

	// Should not panic or error
	logger.LogToolCall(event)
	logger.LogToolCall(nil) // Should handle nil gracefully
}

func TestSimpleLogger(t *testing.T) {
	logger := &SimpleLogger{}

	event := &ToolCallEvent{
		Tool:      "test_tool",
		Args:      map[string]any{"key": "value"},
		Result:    "success",
		Error:     nil,
		Duration:  100 * time.Millisecond,
		Timestamp: time.Now(),
		ThreadID:  "thread123",
	}

	// Should not panic
	logger.LogToolCall(event)
	logger.LogToolCall(nil) // Should handle nil gracefully
}

func TestContextWithLogger(t *testing.T) {
	logger := NewContextLogger()
	ctx := context.Background()
	ctxWithLogger := ContextWithLogger(ctx, logger)

	// Retrieve the logger from context
	retrievedLogger := LoggerFromContext(ctxWithLogger)
	if retrievedLogger != logger {
		t.Errorf("ContextWithLogger() failed to retrieve logger from context")
	}

	// Retrieve from context without logger should return NoOpLogger
	defaultLogger := LoggerFromContext(context.Background())
	if _, ok := defaultLogger.(*NoOpLogger); !ok {
		t.Errorf("LoggerFromContext() expected NoOpLogger, got %T", defaultLogger)
	}
}

func TestLoggerInterface(t *testing.T) {
	// Test that all logger implementations implement Logger interface
	var loggers []Logger
	loggers = append(loggers,
		&SimpleLogger{},
		&NoOpLogger{},
		NewContextLogger(),
	)

	event := &ToolCallEvent{
		Tool:      "test",
		Args:      map[string]any{},
		Result:    "result",
		Error:     nil,
		Duration:  10 * time.Millisecond,
		Timestamp: time.Now(),
		ThreadID:  "test",
	}

	for _, logger := range loggers {
		// Should not panic
		logger.LogToolCall(event)
	}
}

func TestToolCallEventWithError(t *testing.T) {
	logger := NewContextLogger()

	// Test logging an event with an error
	testErr := errors.New("tool execution failed")
	event := &ToolCallEvent{
		Tool:      "failing_tool",
		Args:      map[string]any{"arg": "value"},
		Result:    nil,
		Error:     testErr,
		Duration:  50 * time.Millisecond,
		Timestamp: time.Now(),
		ThreadID:  "thread1",
	}

	logger.LogToolCall(event)

	events := logger.Events()
	if len(events) != 1 {
		t.Errorf("ContextLogger expected 1 event, got %d", len(events))
	}

	if events[0].Error == nil {
		t.Errorf("ContextLogger expected event to have error")
	}

	if events[0].Error != testErr {
		t.Errorf("ContextLogger expected error, got %v", events[0].Error)
	}
}
