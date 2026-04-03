//go:build prometheus
// +build prometheus

package agent

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// GetPrometheusHTTPHandler returns an http.Handler for exposing Prometheus metrics.
// It returns nil when metrics collector is nil or does not expose a *prometheus.Registry.
func GetPrometheusHTTPHandler(metrics MetricsCollector) http.Handler {
	regAny := GetPrometheusRegistry(metrics)
	reg, ok := regAny.(*prometheus.Registry)
	if !ok || reg == nil {
		return nil
	}
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
}
