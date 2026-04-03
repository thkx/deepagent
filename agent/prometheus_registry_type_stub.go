//go:build !prometheus
// +build !prometheus

package agent

// PrometheusRegistry is a placeholder type when prometheus support is disabled.
type PrometheusRegistry struct{}
