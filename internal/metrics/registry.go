package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/sirupsen/logrus"
)

// Service names for metrics registration
const (
	ServiceFee       = "fee"
	ServiceTxIndexer = "tx_indexer"
	ServiceVault     = "vault"
	ServiceWorker    = "worker"
	ServiceHTTP      = "http"
)

// RegisterMetrics registers metrics for the specified services with a custom registry
func RegisterMetrics(services []string, registry *prometheus.Registry, logger *logrus.Logger) {
	// Always register Go and process metrics
	registerIfNotExists(collectors.NewGoCollector(), "go_collector", registry, logger)
	registerIfNotExists(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}), "process_collector", registry, logger)

	// Register service-specific metrics
	for _, service := range services {
		switch service {
		case ServiceTxIndexer:
			registerTxIndexerMetrics(registry, logger)
		case ServiceWorker:
			registerWorkerMetrics(registry, logger)
		case ServiceHTTP:
			registerHTTPMetrics(registry, logger)
		default:
			logger.Warnf("Unknown service type for metrics registration: %s", service)
		}
	}
}

// registerIfNotExists registers a collector if it's not already registered
func registerIfNotExists(collector prometheus.Collector, name string, registry *prometheus.Registry, logger *logrus.Logger) {
	if err := registry.Register(collector); err != nil {
		var alreadyRegErr prometheus.AlreadyRegisteredError
		if !errors.As(err, &alreadyRegErr) {
			// This is a real problem (descriptor mismatch, etc.) - log error but don't fail
			logger.Errorf("Failed to register %s: %v", name, err)
		}
	}
}

// registerTxIndexerMetrics registers tx_indexer-related metrics
func registerTxIndexerMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// Register each TX indexer metric individually with defensive pattern
	registerIfNotExists(txIndexerTransactionStatus, "tx_indexer_transaction_status", registry, logger)
	registerIfNotExists(txIndexerProcessingTotal, "tx_indexer_processing_total", registry, logger)
	registerIfNotExists(txIndexerIterationDuration, "tx_indexer_iteration_duration", registry, logger)
	registerIfNotExists(txIndexerLastProcessingTimestamp, "tx_indexer_last_processing_timestamp", registry, logger)
	registerIfNotExists(txIndexerActiveTransactions, "tx_indexer_active_transactions", registry, logger)
	registerIfNotExists(txIndexerProcessingErrors, "tx_indexer_processing_errors", registry, logger)
	registerIfNotExists(txIndexerRPCErrors, "tx_indexer_rpc_errors", registry, logger)
	registerIfNotExists(txIndexerChainHeight, "tx_indexer_chain_height", registry, logger)
}

// registerWorkerMetrics registers worker-related metrics
func registerWorkerMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// Register each worker metric individually with defensive pattern
	registerIfNotExists(workerTasksTotal, "worker_tasks_total", registry, logger)
	registerIfNotExists(workerTaskDuration, "worker_task_duration", registry, logger)
	registerIfNotExists(workerTasksActive, "worker_tasks_active", registry, logger)
	registerIfNotExists(workerVaultOperationsTotal, "worker_vault_operations_total", registry, logger)
	registerIfNotExists(workerVaultOperationDuration, "worker_vault_operation_duration", registry, logger)
	registerIfNotExists(workerSignaturesGenerated, "worker_signatures_generated", registry, logger)
	registerIfNotExists(workerErrorsTotal, "worker_errors_total", registry, logger)
	registerIfNotExists(workerLastTaskTimestamp, "worker_last_task_timestamp", registry, logger)
}

// registerHTTPMetrics registers HTTP-related metrics
func registerHTTPMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// Register each HTTP metric individually with defensive pattern
	registerIfNotExists(httpRequestsTotal, "http_requests_total", registry, logger)
	registerIfNotExists(httpRequestDuration, "http_request_duration", registry, logger)
	registerIfNotExists(httpActiveRequests, "http_active_requests", registry, logger)
}
