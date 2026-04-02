//go:build prometheus
// +build prometheus

package agent

import "github.com/prometheus/client_golang/prometheus"

// PrometheusRegistry aliases prometheus.Registry when prometheus support is enabled.
type PrometheusRegistry = prometheus.Registry
