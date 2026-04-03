//go:build !prometheus
// +build !prometheus

package agent

import "testing"

func TestGetPrometheusHTTPHandlerStub(t *testing.T) {
	if h := GetPrometheusHTTPHandler(nil); h != nil {
		t.Fatalf("expected nil handler in non-prometheus build")
	}
}
