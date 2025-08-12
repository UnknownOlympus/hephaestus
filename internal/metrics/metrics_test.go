package metrics_test

import (
	"testing"

	"github.com/UnknownOlympus/hephaestus/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(_ *testing.T) {
	reg := prometheus.NewRegistry()

	_ = metrics.NewMetrics(reg)
}
