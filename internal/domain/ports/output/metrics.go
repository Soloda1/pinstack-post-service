package ports

import "time"

type MetricsProvider interface {
	IncrementGRPCRequests(method, status string)
	RecordGRPCRequestDuration(method, status string, duration time.Duration)

	IncrementDatabaseQueries(queryType string, success bool)
	RecordDatabaseQueryDuration(queryType string, duration time.Duration)

	IncrementCacheHits()
	IncrementCacheMisses()
	RecordCacheOperationDuration(operation string, duration time.Duration)
	RecordCacheHitDuration(operation string, duration time.Duration)
	RecordCacheMissDuration(operation string, duration time.Duration)

	IncrementPostOperations(operation string, success bool)
	IncrementTagOperations(operation string, success bool)
	IncrementMediaOperations(operation string, success bool)
	SetActiveConnections(count int)

	SetServiceHealth(healthy bool)
}
