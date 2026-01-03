package scheduler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/vultisig/verifier/types"
)

// Configuration field names
const (
	cfgFrequency = "frequency"
	cfgEndDate   = "endDate"

	// Frequency values (must match app-recurring exactly)
	freqOnetime  = "one-time"
	freqMinutely = "minutely"
	freqHourly   = "hourly"
	freqDaily    = "daily"
	freqWeekly   = "weekly"
	freqBiWeekly = "bi-weekly"
	freqMonthly  = "monthly"
)

// DefaultInterval provides a standard implementation of the Interval interface.
// Plugins can use this directly or implement their own custom logic.
type DefaultInterval struct{}

// NewDefaultInterval creates a new DefaultInterval instance.
func NewDefaultInterval() *DefaultInterval {
	return &DefaultInterval{}
}

// FromNowWhenNext calculates when the next execution should occur based on the policy's recipe configuration.
// Returns zero time if there should be no more executions (policy expired or one-time completed).
func (i *DefaultInterval) FromNowWhenNext(policy types.PluginPolicy) (time.Time, error) {
	recipe, err := policy.GetRecipe()
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to unpack recipe: %w", err)
	}

	cfg := recipe.GetConfiguration()
	if cfg == nil {
		return time.Time{}, fmt.Errorf("recipe configuration is nil")
	}
	fields := cfg.GetFields()
	if fields == nil {
		return time.Time{}, fmt.Errorf("recipe configuration fields are nil")
	}

	// Check if endDate has passed
	if endDateField, exists := fields[cfgEndDate]; exists {
		endDateStr := endDateField.GetStringValue()
		if endDateStr != "" {
			endTime, err := parseDateTime(endDateStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("failed to parse endDate '%s': %w", endDateStr, err)
			}
			if time.Now().After(endTime) {
				return time.Time{}, nil // Expired
			}
		}
	}

	// Determine next execution based on frequency
	var next time.Time
	freqField, exists := fields[cfgFrequency]
	if !exists {
		return time.Time{}, fmt.Errorf("frequency field not found in configuration")
	}
	freq := freqField.GetStringValue()

	switch freq {
	case freqOnetime:
		return time.Time{}, nil // One-time = no next execution
	case freqMinutely:
		next = time.Now().Add(time.Minute)
	case freqHourly:
		next = time.Now().Add(time.Hour)
	case freqDaily:
		next = time.Now().AddDate(0, 0, 1)
	case freqWeekly:
		next = time.Now().AddDate(0, 0, 7)
	case freqBiWeekly:
		next = time.Now().AddDate(0, 0, 14)
	case freqMonthly:
		next = time.Now().AddDate(0, 1, 0)
	default:
		return time.Time{}, fmt.Errorf("unknown frequency: %s", freq)
	}

	// Check if next execution would be after endDate
	if endDateField, exists := fields[cfgEndDate]; exists {
		endDateStr := endDateField.GetStringValue()
		if endDateStr != "" {
			endTime, err := parseDateTime(endDateStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("failed to parse endDate '%s': %w", endDateStr, err)
			}
			if next.After(endTime) {
				return time.Time{}, nil // Next execution would be after end date
			}
		}
	}

	return next, nil
}

// parseDateTime parses a date string that can be either RFC3339 format or Unix milliseconds
func parseDateTime(dateStr string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
		return t, nil
	}

	// Try Unix milliseconds (e.g., "1765464900000")
	if ms, err := strconv.ParseInt(dateStr, 10, 64); err == nil {
		return time.UnixMilli(ms), nil
	}

	return time.Time{}, fmt.Errorf("invalid date format: %s (expected RFC3339 or Unix milliseconds)", dateStr)
}
