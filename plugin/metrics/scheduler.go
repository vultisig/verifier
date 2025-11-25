package metrics

// SchedulerMetrics interface for collecting scheduler metrics
type SchedulerMetrics interface {
	// SetActivePolicies sets the total number of active policies
	SetActivePolicies(count float64)
	
	// SetStuckPolicies sets the number of policies with next_execution < now (stuck policies)
	SetStuckPolicies(count float64)
}