//go:build prometheus
// +build prometheus

package agent

import (
    "github.com/prometheus/client_golang/prometheus"
)

type prometheusCollector struct {
    toolCalls      *prometheus.CounterVec
    toolDurations  *prometheus.HistogramVec
    modelCalls     *prometheus.CounterVec
    modelDurations *prometheus.HistogramVec
    runDuration    prometheus.Histogram
}

// NewPrometheusCollector creates a collector and registers metrics on the provided registry.
func NewPrometheusCollector(registry *prometheus.Registry) MetricsCollector {
    pc := &prometheusCollector{
        toolCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
            Name: "deepagent_tool_calls_total",
            Help: "Total number of tool calls",
        }, []string{"tool", "result"}),
        toolDurations: prometheus.NewHistogramVec(prometheus.HistogramOpts{
            Name:    "deepagent_tool_call_duration_seconds",
            Help:    "Tool call duration seconds",
            Buckets: prometheus.DefBuckets,
        }, []string{"tool"}),
        modelCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
            Name: "deepagent_model_calls_total",
            Help: "Total number of model calls",
        }, []string{"model", "result"}),
        modelDurations: prometheus.NewHistogramVec(prometheus.HistogramOpts{
            Name:    "deepagent_model_call_duration_seconds",
            Help:    "Model call duration seconds",
            Buckets: prometheus.DefBuckets,
        }, []string{"model"}),
        runDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "deepagent_run_duration_seconds",
            Help:    "Run duration seconds",
            Buckets: prometheus.DefBuckets,
        }),
    }
    registry.MustRegister(pc.toolCalls, pc.toolDurations, pc.modelCalls, pc.modelDurations, pc.runDuration)
    return pc
}

func (p *prometheusCollector) RecordToolCall(event *ToolCallEvent) {
    if event == nil {
        return
    }
    result := "ok"
    if event.Error != nil {
        result = "error"
    }
    p.toolCalls.WithLabelValues(event.Tool, result).Inc()
    p.toolDurations.WithLabelValues(event.Tool).Observe(event.Duration.Seconds())
}

func (p *prometheusCollector) RecordModelCall(event *ModelCallEvent) {
    if event == nil {
        return
    }
    result := "ok"
    if event.Error != nil {
        result = "error"
    }
    p.modelCalls.WithLabelValues(event.Model, result).Inc()
    p.modelDurations.WithLabelValues(event.Model).Observe(event.Duration.Seconds())
}

func (p *prometheusCollector) RecordRunSummary(event *RunSummaryEvent) {
    if event == nil {
        return
    }
    p.runDuration.Observe(event.Duration.Seconds())
}
