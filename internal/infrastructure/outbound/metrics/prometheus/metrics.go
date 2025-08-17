package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_server_requests_total",
			Help: "Total number of gRPC requests processed",
		},
		[]string{"method", "status"},
	)

	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_server_request_duration_seconds",
			Help:    "Duration of gRPC requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "status"},
	)

	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_queries_total",
			Help: "Total number of database queries executed",
		},
		[]string{"query_type", "success"},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)

	CacheHitsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
	)

	CacheMissesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
	)

	CacheOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Duration of cache operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	CacheHitDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_hit_duration_seconds",
			Help:    "Duration of cache hit operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	CacheMissDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_miss_duration_seconds",
			Help:    "Duration of cache miss operations in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	PostOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "post_operations_total",
			Help: "Total number of post operations processed",
		},
		[]string{"operation", "success"},
	)

	TagOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tag_operations_total",
			Help: "Total number of tag operations processed",
		},
		[]string{"operation", "success"},
	)

	MediaOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "media_operations_total",
			Help: "Total number of media operations processed",
		},
		[]string{"operation", "success"},
	)

	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Number of active connections",
		},
	)

	ServiceHealth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "service_health",
			Help: "Service health status (1 = healthy, 0 = unhealthy)",
		},
	)
)
