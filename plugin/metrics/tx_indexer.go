package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vultisig/vultisig-go/common"
)

// Registry is an interface for metrics registries to avoid circular dependencies
type Registry interface {
	MustRegister(collectors ...prometheus.Collector)
}

// TxIndexerMetrics interface for collecting transaction indexer metrics
type TxIndexerMetrics interface {
	// RecordTransactionStatus records a transaction status change
	RecordTransactionStatus(chain common.Chain, status string)
	
	// RecordProcessing records a processed transaction
	RecordProcessing(chain common.Chain)
	
	// RecordIterationDuration records the time spent in an iteration
	RecordIterationDuration(chain common.Chain, duration float64)
	
	// SetLastProcessingTimestamp sets the timestamp of the last processing run
	SetLastProcessingTimestamp(timestamp float64)
	
	// SetActiveTransactions sets the number of active transactions
	SetActiveTransactions(chain common.Chain, status string, count float64)
	
	// RecordProcessingError records a processing error
	RecordProcessingError(chain common.Chain, errorType string)
	
	// RecordRPCError records an RPC error
	RecordRPCError(chain common.Chain)
	
	// SetChainHeight sets the current block height for a chain
	SetChainHeight(chain common.Chain, height float64)
}

// NilTxIndexerMetrics is a no-op implementation for when metrics are disabled
type NilTxIndexerMetrics struct{}

// NewNilTxIndexerMetrics creates a no-op metrics implementation
func NewNilTxIndexerMetrics() TxIndexerMetrics {
	return &NilTxIndexerMetrics{}
}

// All methods are no-ops - safe to call, do nothing
func (n *NilTxIndexerMetrics) RecordTransactionStatus(chain common.Chain, status string) {}
func (n *NilTxIndexerMetrics) RecordProcessing(chain common.Chain) {}
func (n *NilTxIndexerMetrics) RecordIterationDuration(chain common.Chain, duration float64) {}
func (n *NilTxIndexerMetrics) SetLastProcessingTimestamp(timestamp float64) {}
func (n *NilTxIndexerMetrics) SetActiveTransactions(chain common.Chain, status string, count float64) {}
func (n *NilTxIndexerMetrics) RecordProcessingError(chain common.Chain, errorType string) {}
func (n *NilTxIndexerMetrics) RecordRPCError(chain common.Chain) {}
func (n *NilTxIndexerMetrics) SetChainHeight(chain common.Chain, height float64) {}