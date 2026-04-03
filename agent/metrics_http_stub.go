//go:build !prometheus
// +build !prometheus

package agent

import "net/http"

// GetPrometheusHTTPHandler stub for builds without prometheus support.
func GetPrometheusHTTPHandler(metrics MetricsCollector) http.Handler {
	_ = metrics
	return nil
}
