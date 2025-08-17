package prometheus

import (
	"strconv"
	"time"

	ports "pinstack-post-service/internal/domain/ports/output"
)

type PrometheusMetricsProvider struct{}

func NewPrometheusMetricsProvider() ports.MetricsProvider {
	return &PrometheusMetricsProvider{}
}

func (p *PrometheusMetricsProvider) IncrementGRPCRequests(method, status string) {
	GRPCRequestsTotal.WithLabelValues(method, status).Inc()
}

func (p *PrometheusMetricsProvider) RecordGRPCRequestDuration(method, status string, duration time.Duration) {
	GRPCRequestDuration.WithLabelValues(method, status).Observe(duration.Seconds())
}

func (p *PrometheusMetricsProvider) IncrementDatabaseQueries(queryType string, success bool) {
	DatabaseQueriesTotal.WithLabelValues(queryType, strconv.FormatBool(success)).Inc()
}

func (p *PrometheusMetricsProvider) RecordDatabaseQueryDuration(queryType string, duration time.Duration) {
	DatabaseQueryDuration.WithLabelValues(queryType).Observe(duration.Seconds())
}

func (p *PrometheusMetricsProvider) IncrementCacheHits() {
	CacheHitsTotal.Inc()
}

func (p *PrometheusMetricsProvider) IncrementCacheMisses() {
	CacheMissesTotal.Inc()
}

func (p *PrometheusMetricsProvider) RecordCacheOperationDuration(operation string, duration time.Duration) {
	CacheOperationDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

func (p *PrometheusMetricsProvider) IncrementPostOperations(operation string, success bool) {
	PostOperationsTotal.WithLabelValues(operation, strconv.FormatBool(success)).Inc()
}

func (p *PrometheusMetricsProvider) IncrementTagOperations(operation string, success bool) {
	TagOperationsTotal.WithLabelValues(operation, strconv.FormatBool(success)).Inc()
}

func (p *PrometheusMetricsProvider) IncrementMediaOperations(operation string, success bool) {
	MediaOperationsTotal.WithLabelValues(operation, strconv.FormatBool(success)).Inc()
}

func (p *PrometheusMetricsProvider) SetActiveConnections(count int) {
	ActiveConnections.Set(float64(count))
}

func (p *PrometheusMetricsProvider) SetServiceHealth(healthy bool) {
	if healthy {
		ServiceHealth.Set(1)
	} else {
		ServiceHealth.Set(0)
	}
}
