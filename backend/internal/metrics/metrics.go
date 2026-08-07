package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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

type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriterDelegator) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriterDelegator{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start).Seconds()

		routePattern := r.URL.Path
		rctx := chi.RouteContext(r.Context())
		if rctx != nil && rctx.RoutePattern() != "" {
			routePattern = rctx.RoutePattern()
		}

		statusStr := strconv.Itoa(rw.statusCode)
		HTTPRequestsTotal.WithLabelValues(routePattern, r.Method, statusStr).Inc()
		HTTPRequestDurationSeconds.WithLabelValues(routePattern, r.Method).Observe(duration)
	})
}
