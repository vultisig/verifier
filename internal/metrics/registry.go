package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/sirupsen/logrus"
)

// Service names for metrics registration
const (
	ServiceAuth      = "auth"
	ServicePlugin    = "plugin"
	ServicePolicy    = "policy"
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
		case ServiceAuth:
			registerAuthMetrics(registry, logger)
		case ServicePlugin:
			registerPluginMetrics(registry, logger)
		case ServicePolicy:
			registerPolicyMetrics(registry, logger)
		case ServiceFee:
			registerFeeMetrics(registry, logger)
		case ServiceTxIndexer:
			registerTxIndexerMetrics(registry, logger)
		case ServiceVault:
			registerVaultMetrics(registry, logger)
		case ServiceWorker:
			// TODO: Implement worker metrics
			logger.Debug("Worker metrics registration - TODO")
		case ServiceHTTP:
			// TODO: Implement HTTP metrics  
			logger.Debug("HTTP metrics registration - TODO")
		default:
			logger.Warnf("Unknown service type for metrics registration: %s", service)
		}
	}
}

// registerIfNotExists registers a collector if it's not already registered
func registerIfNotExists(collector prometheus.Collector, name string, registry *prometheus.Registry, logger *logrus.Logger) {
	if err := registry.Register(collector); err != nil {
		var alreadyRegErr prometheus.AlreadyRegisteredError
		if errors.As(err, &alreadyRegErr) {
			// This is expected on restart/reload - just debug log
			logger.Debugf("%s already registered", name)
		} else {
			// This is a real problem (descriptor mismatch, etc.) - log error but don't fail
			logger.Errorf("Failed to register %s: %v", name, err)
		}
	} else {
		logger.Debugf("Successfully registered %s", name)
	}
}

// registerAuthMetrics registers authentication-related metrics
func registerAuthMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// TODO: Implement auth metrics
	logger.Debug("Auth metrics registration - TODO")
}

// registerPluginMetrics registers plugin-related metrics
func registerPluginMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// TODO: Implement plugin metrics
	logger.Debug("Plugin metrics registration - TODO")
}

// registerPolicyMetrics registers policy-related metrics
func registerPolicyMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// TODO: Implement policy metrics
	logger.Debug("Policy metrics registration - TODO")
}

// registerFeeMetrics registers fee-related metrics
func registerFeeMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// TODO: Implement fee metrics
	logger.Debug("Fee metrics registration - TODO")
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
	logger.Debug("TX indexer metrics registered")
}

// registerVaultMetrics registers vault-related metrics
func registerVaultMetrics(registry *prometheus.Registry, logger *logrus.Logger) {
	// TODO: Implement vault metrics
	logger.Debug("Vault metrics registration - TODO")
}

