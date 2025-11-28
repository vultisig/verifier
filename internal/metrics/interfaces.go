package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Registry is an interface for metrics registries to avoid circular dependencies
type Registry interface {
	MustRegister(collectors ...prometheus.Collector)
}