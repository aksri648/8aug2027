package metrics_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saas-agent-platform/backend/internal/metrics"
)

func TestPrometheusMiddlewareAndMetrics(t *testing.T) {
	handler := metrics.PrometheusMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))

	req := httptest.NewRequest("GET", "/metrics-test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	metrics.ActiveProjectsTotal.Inc()
	metrics.DaytonaSandboxesActive.Inc()
	metrics.AgentJobsTotal.WithLabelValues("codegen", "succeeded").Inc()
}
