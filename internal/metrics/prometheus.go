package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP request metrics
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code", "tenant_id"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "tenant_id"},
	)

	// Payment metrics
	paymentsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payments_total",
			Help: "Total number of payments created",
		},
		[]string{"tenant_id", "currency", "type", "status"},
	)

	paymentAmountTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_amount_total",
			Help: "Total amount of payments processed",
		},
		[]string{"tenant_id", "currency", "type", "status"},
	)

	paymentProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_processing_duration_seconds",
			Help:    "Time taken to process payments",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		},
		[]string{"tenant_id", "status"},
	)

	// Database metrics
	dbConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_active",
			Help: "Number of active database connections",
		},
	)

	dbQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type", "table"},
	)

	dbQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"query_type", "table", "status"},
	)

	// Redis metrics
	redisConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "redis_connections_active",
			Help: "Number of active Redis connections",
		},
	)

	redisOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "redis_operations_total",
			Help: "Total number of Redis operations",
		},
		[]string{"operation", "status"},
	)

	// Worker metrics
	workerJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_jobs_total",
			Help: "Total number of jobs processed by workers",
		},
		[]string{"job_type", "status"},
	)

	workerQueueLength = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "worker_queue_length",
			Help: "Current length of worker queues",
		},
		[]string{"queue_name"},
	)

	// OPA metrics
	opaEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "opa_evaluations_total",
			Help: "Total number of OPA policy evaluations",
		},
		[]string{"policy", "result"},
	)

	opaEvaluationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "opa_evaluation_duration_seconds",
			Help:    "OPA policy evaluation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"policy"},
	)

	// Idempotency metrics
	idempotencyCacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idempotency_cache_hits_total",
			Help: "Total number of idempotency cache hits",
		},
		[]string{"tenant_id"},
	)

	idempotencyCacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "idempotency_cache_misses_total",
			Help: "Total number of idempotency cache misses",
		},
		[]string{"tenant_id"},
	)

	// Load balancer metrics
	loadBalancerRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "load_balancer_requests_total",
			Help: "Total number of requests through load balancer",
		},
		[]string{"backend", "status"},
	)

	loadBalancerBackendHealth = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_backend_health",
			Help: "Health status of backend servers (1=healthy, 0=unhealthy)",
		},
		[]string{"backend"},
	)

	// Event metrics
	eventsPublishedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_published_total",
			Help: "Total number of events published",
		},
		[]string{"event_type", "tenant_id"},
	)

	eventsProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "events_processed_total",
			Help: "Total number of events processed",
		},
		[]string{"event_type", "tenant_id", "handler"},
	)
)

// MetricsCollector provides methods to record metrics
type MetricsCollector struct{}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// HTTP metrics
func (m *MetricsCollector) RecordHTTPRequest(method, endpoint, statusCode, tenantID string, duration float64) {
	httpRequestsTotal.WithLabelValues(method, endpoint, statusCode, tenantID).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint, tenantID).Observe(duration)
}

// Payment metrics
func (m *MetricsCollector) RecordPaymentCreated(tenantID, currency, paymentType, status string, amount float64) {
	paymentsTotal.WithLabelValues(tenantID, currency, paymentType, status).Inc()
	paymentAmountTotal.WithLabelValues(tenantID, currency, paymentType, status).Add(amount)
}

func (m *MetricsCollector) RecordPaymentProcessing(tenantID, status string, duration float64) {
	paymentProcessingDuration.WithLabelValues(tenantID, status).Observe(duration)
}

// Database metrics
func (m *MetricsCollector) SetDBConnectionsActive(count float64) {
	dbConnectionsActive.Set(count)
}

func (m *MetricsCollector) RecordDBQuery(queryType, table, status string, duration float64) {
	dbQueriesTotal.WithLabelValues(queryType, table, status).Inc()
	dbQueryDuration.WithLabelValues(queryType, table).Observe(duration)
}

// Redis metrics
func (m *MetricsCollector) SetRedisConnectionsActive(count float64) {
	redisConnectionsActive.Set(count)
}

func (m *MetricsCollector) RecordRedisOperation(operation, status string) {
	redisOperationsTotal.WithLabelValues(operation, status).Inc()
}

// Worker metrics
func (m *MetricsCollector) RecordWorkerJob(jobType, status string) {
	workerJobsTotal.WithLabelValues(jobType, status).Inc()
}

func (m *MetricsCollector) SetWorkerQueueLength(queueName string, length float64) {
	workerQueueLength.WithLabelValues(queueName).Set(length)
}

// OPA metrics
func (m *MetricsCollector) RecordOPAEvaluation(policy, result string, duration float64) {
	opaEvaluationsTotal.WithLabelValues(policy, result).Inc()
	opaEvaluationDuration.WithLabelValues(policy).Observe(duration)
}

// Idempotency metrics
func (m *MetricsCollector) RecordIdempotencyCacheHit(tenantID string) {
	idempotencyCacheHits.WithLabelValues(tenantID).Inc()
}

func (m *MetricsCollector) RecordIdempotencyCacheMiss(tenantID string) {
	idempotencyCacheMisses.WithLabelValues(tenantID).Inc()
}

// Load balancer metrics
func (m *MetricsCollector) RecordLoadBalancerRequest(backend, status string) {
	loadBalancerRequestsTotal.WithLabelValues(backend, status).Inc()
}

func (m *MetricsCollector) SetLoadBalancerBackendHealth(backend string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	loadBalancerBackendHealth.WithLabelValues(backend).Set(value)
}

// Event metrics
func (m *MetricsCollector) RecordEventPublished(eventType, tenantID string) {
	eventsPublishedTotal.WithLabelValues(eventType, tenantID).Inc()
}

func (m *MetricsCollector) RecordEventProcessed(eventType, tenantID, handler string) {
	eventsProcessedTotal.WithLabelValues(eventType, tenantID, handler).Inc()
}
