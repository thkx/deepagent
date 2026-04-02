package agent

import (
    "encoding/json"
    "fmt"
    "os"
    "sync"
    "time"
)

// MetricsCollector allows integration with metrics systems (optional)
type MetricsCollector interface {
    RecordToolCall(event *ToolCallEvent)
    RecordModelCall(event *ModelCallEvent)
    RecordRunSummary(event *RunSummaryEvent)
}

// JSONLogger writes structured JSON logs to stdout and optionally emits metrics.
type JSONLogger struct {
    out     *os.File
    mu      sync.Mutex
    metrics MetricsCollector
}

// NewJSONLogger creates a JSONLogger. Pass nil metrics to disable metric hooks.
func NewJSONLogger(metrics MetricsCollector) *JSONLogger {
    return &JSONLogger{out: os.Stdout, metrics: metrics}
}

func (jl *JSONLogger) emit(v any) {
    jl.mu.Lock()
    defer jl.mu.Unlock()
    b, err := json.Marshal(v)
    if err != nil {
        // fallback to fmt on marshal error
        fmt.Fprintf(jl.out, "{\"level\":\"error\",\"error\":\"json marshal failed: %v\"}\n", err)
        return
    }
    jl.out.Write(append(b, '\n'))
}

func (jl *JSONLogger) LogToolCall(event *ToolCallEvent) {
    if event == nil {
        return
    }
    out := map[string]any{
        "ts":       event.Timestamp.Format(time.RFC3339Nano),
        "level":    "info",
        "type":     "tool_call",
        "tool":     event.Tool,
        "duration": event.Duration.Seconds(),
        "args":     event.Args,
        "result":   event.Result,
        "thread":   event.ThreadID,
        "request":  event.RequestID,
    }
    if event.Error != nil {
        out["level"] = "error"
        out["error"] = event.Error.Error()
    }
    jl.emit(out)
    if jl.metrics != nil {
        jl.metrics.RecordToolCall(event)
    }
}

func (jl *JSONLogger) LogModelCall(event *ModelCallEvent) {
    if event == nil {
        return
    }
    out := map[string]any{
        "ts":          event.Timestamp.Format(time.RFC3339Nano),
        "level":       "info",
        "type":        "model_call",
        "model":       event.Model,
        "message_count": event.MessageCount,
        "tool_count":  event.ToolCount,
        "duration":    event.Duration.Seconds(),
        "thread":      event.ThreadID,
        "request":     event.RequestID,
    }
    if event.Error != nil {
        out["level"] = "error"
        out["error"] = event.Error.Error()
    }
    jl.emit(out)
    if jl.metrics != nil {
        jl.metrics.RecordModelCall(event)
    }
}

func (jl *JSONLogger) LogRunSummary(event *RunSummaryEvent) {
    if event == nil {
        return
    }
    out := map[string]any{
        "ts":            time.Now().Format(time.RFC3339Nano),
        "level":         "info",
        "type":          "run_summary",
        "thread":        event.ThreadID,
        "request":       event.RequestID,
        "duration":      event.Duration.Seconds(),
        "iterations":    event.Iterations,
        "model_calls":   event.ModelCalls,
        "model_errors":  event.ModelErrors,
        "tool_calls":    event.ToolCalls,
        "tool_errors":   event.ToolErrors,
        "tool_error_rate": event.ToolErrorRate,
    }
    if event.Error != "" {
        out["level"] = "error"
        out["error"] = event.Error
    }
    jl.emit(out)
    if jl.metrics != nil {
        jl.metrics.RecordRunSummary(event)
    }
}
