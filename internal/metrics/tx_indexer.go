package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	pluginmetrics "github.com/vultisig/verifier/plugin/metrics"
	"github.com/vultisig/vultisig-go/common"
)

// TxIndexerPrometheusMetrics is the Prometheus implementation of TxIndexerMetrics interface
type TxIndexerPrometheusMetrics struct {
	// Transaction status tracking by chain and status
	OnChainStatusTotal *prometheus.CounterVec
	
	// Processing rates and durations
	ProcessingRateTotal      *prometheus.CounterVec
	IterationDurationSeconds *prometheus.HistogramVec
	
	// System health metrics
	LastProcessingTimestamp prometheus.Gauge
	ActiveTransactions      *prometheus.GaugeVec
	
	// Error tracking
	ProcessingErrorsTotal *prometheus.CounterVec
	RPCErrorsTotal        *prometheus.CounterVec
	
	// Chain-specific metrics
	ChainHeightGauge *prometheus.GaugeVec
}

// NewTxIndexerMetrics creates and registers all TX indexer metrics
func NewTxIndexerMetrics() pluginmetrics.TxIndexerMetrics {
	return &TxIndexerPrometheusMetrics{
		OnChainStatusTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tx_indexer_onchain_status_total",
				Help: "Total number of transactions by chain and on-chain status",
			},
			[]string{"chain", "status"},
		),
		
		ProcessingRateTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tx_indexer_processing_rate_total", 
				Help: "Total number of transactions processed by chain",
			},
			[]string{"chain"},
		),
		
		IterationDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "tx_indexer_iteration_duration_seconds",
				Help:    "Time spent processing each iteration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"chain"},
		),
		
		LastProcessingTimestamp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "tx_indexer_last_processing_timestamp",
				Help: "Timestamp of the last successful processing run",
			},
		),
		
		ActiveTransactions: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tx_indexer_active_transactions",
				Help: "Number of active transactions by chain and status",
			},
			[]string{"chain", "status"},
		),
		
		ProcessingErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tx_indexer_processing_errors_total",
				Help: "Total number of processing errors by chain and type",
			},
			[]string{"chain", "error_type"},
		),
		
		RPCErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "tx_indexer_rpc_errors_total",
				Help: "Total number of RPC errors by chain",
			},
			[]string{"chain"},
		),
		
		ChainHeightGauge: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "tx_indexer_chain_height",
				Help: "Current block height for each chain",
			},
			[]string{"chain"},
		),
	}
}

// Register registers all metrics with the provided registry
func (m *TxIndexerPrometheusMetrics) Register(registry pluginmetrics.Registry) {
	registry.MustRegister(
		m.OnChainStatusTotal,
		m.ProcessingRateTotal,
		m.IterationDurationSeconds,
		m.LastProcessingTimestamp,
		m.ActiveTransactions,
		m.ProcessingErrorsTotal,
		m.RPCErrorsTotal,
		m.ChainHeightGauge,
	)
}

// RecordTransactionStatus records a transaction status change
func (m *TxIndexerPrometheusMetrics) RecordTransactionStatus(chain common.Chain, status string) {
	m.OnChainStatusTotal.WithLabelValues(chain.String(), status).Inc()
}

// RecordProcessing records a processed transaction
func (m *TxIndexerPrometheusMetrics) RecordProcessing(chain common.Chain) {
	m.ProcessingRateTotal.WithLabelValues(chain.String()).Inc()
}

// RecordIterationDuration records the time spent in an iteration
func (m *TxIndexerPrometheusMetrics) RecordIterationDuration(chain common.Chain, duration float64) {
	m.IterationDurationSeconds.WithLabelValues(chain.String()).Observe(duration)
}

// SetLastProcessingTimestamp sets the timestamp of the last processing run
func (m *TxIndexerPrometheusMetrics) SetLastProcessingTimestamp(timestamp float64) {
	m.LastProcessingTimestamp.Set(timestamp)
}

// SetActiveTransactions sets the number of active transactions
func (m *TxIndexerPrometheusMetrics) SetActiveTransactions(chain common.Chain, status string, count float64) {
	m.ActiveTransactions.WithLabelValues(chain.String(), status).Set(count)
}

// RecordProcessingError records a processing error
func (m *TxIndexerPrometheusMetrics) RecordProcessingError(chain common.Chain, errorType string) {
	m.ProcessingErrorsTotal.WithLabelValues(chain.String(), errorType).Inc()
}

// RecordRPCError records an RPC error
func (m *TxIndexerPrometheusMetrics) RecordRPCError(chain common.Chain) {
	m.RPCErrorsTotal.WithLabelValues(chain.String()).Inc()
}

// SetChainHeight sets the current block height for a chain
func (m *TxIndexerPrometheusMetrics) SetChainHeight(chain common.Chain, height float64) {
	m.ChainHeightGauge.WithLabelValues(chain.String()).Set(height)
}