package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	appstoreInstallationsTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "installations_total",
			Help:      "Current number of installations per plugin",
		},
		[]string{"plugin_id"},
	)

	appstorePoliciesTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "policies_total",
			Help:      "Current number of active policies per plugin",
		},
		[]string{"plugin_id"},
	)

	appstoreFeesEarnedTotal = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "fees_earned_total",
			Help:      "Total fees earned per plugin in USDC",
		},
		[]string{"plugin_id"},
	)

	appstoreInstallationsGrandTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "installations_grand_total",
			Help:      "Total installations across all plugins",
		},
	)

	appstorePoliciesGrandTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "policies_grand_total",
			Help:      "Total active policies across all plugins",
		},
	)

	appstoreFeesGrandTotal = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "fees_grand_total",
			Help:      "Total fees earned across all plugins in USDC",
		},
	)

	appstoreCollectorLastUpdateTimestamp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "verifier",
			Subsystem: "appstore",
			Name:      "collector_last_update_timestamp",
			Help:      "Unix timestamp of last successful metrics update",
		},
	)
)

type AppStoreMetrics struct{}

func NewAppStoreMetrics() *AppStoreMetrics {
	return &AppStoreMetrics{}
}

func (a *AppStoreMetrics) UpdateInstallations(data map[string]int64) {
	appstoreInstallationsTotal.Reset()

	var grandTotal int64
	for pluginID, count := range data {
		appstoreInstallationsTotal.WithLabelValues(pluginID).Set(float64(count))
		grandTotal += count
	}

	appstoreInstallationsGrandTotal.Set(float64(grandTotal))
}

func (a *AppStoreMetrics) UpdatePolicies(data map[string]int64) {
	appstorePoliciesTotal.Reset()

	var grandTotal int64
	for pluginID, count := range data {
		appstorePoliciesTotal.WithLabelValues(pluginID).Set(float64(count))
		grandTotal += count
	}

	appstorePoliciesGrandTotal.Set(float64(grandTotal))
}

const usdcDecimals = 1e6

func (a *AppStoreMetrics) UpdateFees(data map[string]int64) {
	appstoreFeesEarnedTotal.Reset()

	var grandTotal int64
	for pluginID, total := range data {
		appstoreFeesEarnedTotal.WithLabelValues(pluginID).Set(float64(total) / usdcDecimals)
		grandTotal += total
	}

	appstoreFeesGrandTotal.Set(float64(grandTotal) / usdcDecimals)
}

func (a *AppStoreMetrics) UpdateTimestamp() {
	appstoreCollectorLastUpdateTimestamp.Set(float64(time.Now().Unix()))
}
