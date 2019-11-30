package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

// ServeMetrics exposes metrics in Prometheus format
func ServeMetrics(metricsPort string) {
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+metricsPort, nil)
}
