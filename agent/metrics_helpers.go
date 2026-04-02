package agent

// GetPrometheusRegistry returns the underlying metrics registry/handle when available.
// When metrics collector does not expose one (or is nil), it returns nil.
//
// In prometheus-enabled builds, the returned value is *prometheus.Registry.
func GetPrometheusRegistry(metrics MetricsCollector) any {
	if metrics == nil {
		return nil
	}
	if provider, ok := metrics.(MetricsProvider); ok {
		return provider.Metrics()
	}
	return nil
}
