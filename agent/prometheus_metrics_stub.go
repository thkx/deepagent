//go:build !prometheus
// +build !prometheus

package agent

// NewPrometheusCollector stub when prometheus support is not compiled in.
func NewPrometheusCollector(_ interface{}) MetricsCollector {
    return nil
}
