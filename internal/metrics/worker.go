package metrics

import (
	"context"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Task processing metrics
	workerTasksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "tasks_total",
			Help:      "Total number of tasks processed by type and status",
		},
		[]string{"task_type", "status"}, // status: completed, failed
	)

	workerTaskDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "task_duration_seconds",
			Help:      "Duration of task processing by type",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"task_type"},
	)

	workerTasksActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "tasks_active",
			Help:      "Number of currently active tasks by type",
		},
		[]string{"task_type"},
	)

	// Vault operation metrics
	workerVaultOperationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "vault_operations_total",
			Help:      "Total number of vault operations by operation and status",
		},
		[]string{"operation", "status"}, // operation: keygen, keysign, reshare
	)

	workerVaultOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "vault_operation_duration_seconds",
			Help:      "Duration of vault operations by type",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	workerSignaturesGenerated = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "signatures_generated_total",
			Help:      "Total number of successful signatures generated",
		},
	)

	// Error tracking
	workerErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "errors_total",
			Help:      "Total number of errors by task type and error type",
		},
		[]string{"task_type", "error_type"},
	)

	// Health tracking
	workerLastTaskTimestamp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "worker",
			Name:      "last_task_timestamp",
			Help:      "Timestamp of when worker last processed a task",
		},
	)
)

// WorkerMetrics provides methods to update worker-related metrics
type WorkerMetrics struct{}

// NewWorkerMetrics creates a new instance of WorkerMetrics
func NewWorkerMetrics() *WorkerMetrics {
	return &WorkerMetrics{}
}

// Register registers all worker metrics with the provided registry
func (wm *WorkerMetrics) Register(registry *prometheus.Registry) {
	registry.MustRegister(
		workerTasksTotal,
		workerTaskDuration,
		workerTasksActive,
		workerVaultOperationsTotal,
		workerVaultOperationDuration,
		workerSignaturesGenerated,
		workerErrorsTotal,
		workerLastTaskTimestamp,
	)
}

// RecordTaskCompleted records a successfully completed task
func (wm *WorkerMetrics) RecordTaskCompleted(taskType string, duration float64) {
	workerTasksTotal.WithLabelValues(taskType, "completed").Inc()
	workerTaskDuration.WithLabelValues(taskType).Observe(duration)
	workerLastTaskTimestamp.SetToCurrentTime()
}

// RecordTaskFailed records a failed task
func (wm *WorkerMetrics) RecordTaskFailed(taskType string, duration float64) {
	workerTasksTotal.WithLabelValues(taskType, "failed").Inc()
	workerTaskDuration.WithLabelValues(taskType).Observe(duration)
	workerLastTaskTimestamp.SetToCurrentTime()
}

// RecordTaskStarted records when a task starts (increments active count)
func (wm *WorkerMetrics) RecordTaskStarted(taskType string) {
	workerTasksActive.WithLabelValues(taskType).Inc()
}

// RecordTaskFinished records when a task finishes (decrements active count)
func (wm *WorkerMetrics) RecordTaskFinished(taskType string) {
	workerTasksActive.WithLabelValues(taskType).Dec()
}

// RecordVaultOperation records a vault operation (keygen, keysign, reshare)
func (wm *WorkerMetrics) RecordVaultOperation(operation, status string, duration float64) {
	workerVaultOperationsTotal.WithLabelValues(operation, status).Inc()
	workerVaultOperationDuration.WithLabelValues(operation).Observe(duration)
	
	// Special case: count successful signatures
	if operation == "keysign" && status == "completed" {
		workerSignaturesGenerated.Inc()
	}
}

// RecordError records an error during task processing
func (wm *WorkerMetrics) RecordError(taskType, errorType string) {
	workerErrorsTotal.WithLabelValues(taskType, errorType).Inc()
}

// WithWorkerMetrics wraps a task handler with worker metrics collection
func WithWorkerMetrics(handler asynq.HandlerFunc, taskType string, metrics *WorkerMetrics) asynq.HandlerFunc {
	return asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		// If metrics are disabled, just run the handler
		if metrics == nil {
			return handler.ProcessTask(ctx, task)
		}

		start := time.Now()
		metrics.RecordTaskStarted(taskType)
		defer metrics.RecordTaskFinished(taskType)

		// Execute the task
		err := handler.ProcessTask(ctx, task)
		duration := time.Since(start).Seconds()

		// Record results
		if err != nil {
			metrics.RecordTaskFailed(taskType, duration)
			metrics.RecordError(taskType, classifyError(err))
		} else {
			metrics.RecordTaskCompleted(taskType, duration)
			
			// Record task-specific success metrics
			switch taskType {
			case "keysign":
				metrics.RecordVaultOperation("keysign", "completed", duration)
				// Assume one signature per successful keysign task
				// (In reality, this might be multiple signatures, but we can't tell from wrapper level)
			case "keygen":
				metrics.RecordVaultOperation("keygen", "completed", duration)
			case "reshare":
				metrics.RecordVaultOperation("reshare", "completed", duration)
			}
		}

		return err
	})
}

// classifyError provides basic error classification for metrics
func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "context"):
		return "context_cancelled"
	case strings.Contains(errStr, "network") || strings.Contains(errStr, "connection"):
		return "network"
	case strings.Contains(errStr, "validation") || strings.Contains(errStr, "invalid"):
		return "validation"
	default:
		return "unknown"
	}
}