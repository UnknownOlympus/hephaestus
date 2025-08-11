package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds the various metrics used for monitoring the application.
// It includes counters for runs, login attempts, and items parsed,
// a gauge for the last successful run, and a histogram for run duration.
type Metrics struct {
	Runs              *prometheus.CounterVec
	ItemsParsed       *prometheus.CounterVec
	LastSuccessfulRun *prometheus.GaugeVec
	RunDuration       *prometheus.HistogramVec
	EmailsFixed       prometheus.Counter
	DBQueryDuration   *prometheus.HistogramVec
}

// NewMetrics creates a new Metrics instance with the provided Registerer.
// It initializes various Prometheus metrics including counters for runs,
// login attempts, and items parsed, as well as gauges and histograms for
// tracking the last successful run and the duration of runs.
//
// Parameters:
//   - reg: A prometheus.Registerer used to register the metrics.
//
// Returns:
//   - A pointer to the newly created Metrics instance.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	metrics := &Metrics{
		Runs: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "usprovider_runs_total",
			Help: "Total times the parser has successfully or unsuccessfully completed its full cycle.",
		}, []string{"status"}),
		ItemsParsed: promauto.With(reg).NewCounterVec(prometheus.CounterOpts{
			Name: "usprovider_items_parsed_total",
			Help: "Total number of parsed items",
		}, []string{"type"}),
		LastSuccessfulRun: promauto.With(reg).NewGaugeVec(prometheus.GaugeOpts{
			Name: "usprovider_last_successful_run_timestamp",
			Help: "Last time when run was successfully",
		}, []string{"type"}),
		RunDuration: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name: "usprovider_run_duration_seconds",
			Help: "Measures how long it takes for a full parser cycle to complete",
		}, []string{"type"}),
		EmailsFixed: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "usprovider_emails_fixed_total",
			Help: "Total number of employee emails that were fixed or generated.",
		}),
		DBQueryDuration: promauto.With(reg).NewHistogramVec(prometheus.HistogramOpts{
			Name:    "usprovider_db_query_duration_seconds",
			Help:    "Duration of database queries.",
			Buckets: prometheus.DefBuckets,
		}, []string{"query_type"}), // query_type: 'get_employee', 'upsert_task'
	}

	metrics.Runs.WithLabelValues("success")
	metrics.Runs.WithLabelValues("failure")

	return metrics
}
