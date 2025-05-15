package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payflow_http_requests_total",
			Help: "Toplam HTTP istek sayısı",
		},
		[]string{"method", "endpoint", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payflow_http_request_duration_seconds",
			Help:    "HTTP istek süresi (saniye)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payflow_database_operations_total",
			Help: "Toplam veritabanı operasyonu sayısı",
		},
		[]string{"operation", "entity"},
	)

	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payflow_database_operation_duration_seconds",
			Help:    "Veritabanı operasyon süresi (saniye)",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "entity"},
	)

	TransactionProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payflow_transactions_processed_total",
			Help: "İşlenen toplam işlem sayısı",
		},
		[]string{"type", "status"},
	)

	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "payflow_active_users",
			Help: "Aktif kullanıcı sayısı",
		},
	)

	WorkerPoolQueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "payflow_worker_pool_queue_size",
			Help: "Worker pool kuyruğundaki iş sayısı",
		},
	)

	WorkerPoolActiveWorkers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "payflow_worker_pool_active_workers",
			Help: "Aktif worker sayısı",
		},
	)

	CacheHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "payflow_cache_hits_total",
			Help: "Önbellek isabet sayısı",
		},
	)

	CacheMisses = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "payflow_cache_misses_total",
			Help: "Önbellek isabet etmeme sayısı",
		},
	)
)

func RecordHttpRequest(method, endpoint, status string, duration time.Duration) {
	HttpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HttpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func RecordDatabaseOperation(operation, entity string, duration time.Duration) {
	DatabaseOperationsTotal.WithLabelValues(operation, entity).Inc()
	DatabaseOperationDuration.WithLabelValues(operation, entity).Observe(duration.Seconds())
}

func RecordTransaction(txType string, status string) {
	TransactionProcessed.WithLabelValues(txType, status).Inc()
}

func UpdateWorkerPoolStats(queueSize, activeWorkers int) {
	WorkerPoolQueueSize.Set(float64(queueSize))
	WorkerPoolActiveWorkers.Set(float64(activeWorkers))
}

func RecordCacheHit() {
	CacheHits.Inc()
}

func RecordCacheMiss() {
	CacheMisses.Inc()
}
