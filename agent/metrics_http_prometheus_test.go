//go:build prometheus
// +build prometheus

package agent

import "testing"

func TestGetPrometheusHTTPHandlerWithCollector(t *testing.T) {
	collector := NewPrometheusCollector(nil)
	h := GetPrometheusHTTPHandler(collector)
	if h == nil {
		t.Fatalf("expected non-nil prometheus handler")
	}
}

func TestGetPrometheusHTTPHandlerWithNilCollector(t *testing.T) {
	h := GetPrometheusHTTPHandler(nil)
	if h != nil {
		t.Fatalf("expected nil handler for nil collector")
	}
}
