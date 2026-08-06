package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saas_http_requests_total",
			Help: "Total number of HTTP requests processed by API server",
		},
		[]string{"path", "method", "status"},
	)

	HTTPRequestDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "saas_http_request_duration_seconds",
			Help:    "Histogram of HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	ActiveProjectsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "saas_active_projects_total",
			Help: "Current total count of active user projects",
		},
	)

	AgentJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "saas_agent_jobs_total",
			Help: "Total count of agent jobs processed by type and status",
		},
		[]string{"type", "status"},
	)

	DaytonaSandboxesActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "saas_daytona_sandboxes_active",
			Help: "Total count of active Daytona Cloud Sandbox workspaces",
		},
	)
)
