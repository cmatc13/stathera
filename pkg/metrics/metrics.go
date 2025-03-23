// Package metrics provides metrics collection capabilities for the application.
package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all the metrics collectors for the application.
type Metrics struct {
	// Registry is the Prometheus registry for all metrics.
	Registry *prometheus.Registry

	// Common metrics
	RequestCount        *prometheus.CounterVec
	RequestDuration     *prometheus.HistogramVec
	RequestInFlight     *prometheus.GaugeVec
	ErrorCount          *prometheus.CounterVec
	ServiceUptime       prometheus.Gauge
	ServiceLastStarted  prometheus.Gauge
	DependencyUp        *prometheus.GaugeVec
	DependencyLatency   *prometheus.HistogramVec
	DependencyErrorRate *prometheus.CounterVec

	// Transaction metrics
	TransactionCount      *prometheus.CounterVec
	TransactionAmount     *prometheus.HistogramVec
	TransactionDuration   *prometheus.HistogramVec
	TransactionErrorCount *prometheus.CounterVec

	// Order book metrics
	OrderCount      *prometheus.CounterVec
	OrderAmount     *prometheus.HistogramVec
	OrderDuration   *prometheus.HistogramVec
	OrderErrorCount *prometheus.CounterVec
	OrderBookDepth  *prometheus.GaugeVec

	// Supply metrics
	TotalSupply    prometheus.Gauge
	InflationRate  prometheus.Gauge
	SupplyChanges  *prometheus.CounterVec
	ReserveBalance prometheus.Gauge
}

// Config holds the configuration for metrics.
type Config struct {
	// Namespace is the Prometheus namespace for all metrics.
	Namespace string
	// Subsystem is the Prometheus subsystem for all metrics.
	Subsystem string
	// ServiceName is the name of the service that is collecting metrics.
	ServiceName string
}

// DefaultConfig returns a default metrics configuration.
func DefaultConfig() Config {
	return Config{
		Namespace:   "stathera",
		Subsystem:   "",
		ServiceName: "stathera",
	}
}

// New creates a new metrics collector with the given configuration.
func New(cfg Config) *Metrics {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	m := &Metrics{
		Registry: registry,

		// Common metrics
		RequestCount: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_total",
				Help:      "Total number of requests received",
			},
			[]string{"service", "method", "path", "status"},
		),

		RequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"service", "method", "path"},
		),

		RequestInFlight: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "requests_in_flight",
				Help:      "Current number of requests being processed",
			},
			[]string{"service"},
		),

		ErrorCount: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "errors_total",
				Help:      "Total number of errors",
			},
			[]string{"service", "type", "code"},
		),

		ServiceUptime: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "service_uptime_seconds",
				Help:      "Service uptime in seconds",
				ConstLabels: prometheus.Labels{
					"service": cfg.ServiceName,
				},
			},
		),

		ServiceLastStarted: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "service_last_started_timestamp",
				Help:      "Timestamp when the service was last started",
				ConstLabels: prometheus.Labels{
					"service": cfg.ServiceName,
				},
			},
		),

		DependencyUp: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "dependency_up",
				Help:      "Whether the dependency is up (1) or down (0)",
			},
			[]string{"service", "dependency"},
		),

		DependencyLatency: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "dependency_latency_seconds",
				Help:      "Dependency request latency in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"service", "dependency", "operation"},
		),

		DependencyErrorRate: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: cfg.Subsystem,
				Name:      "dependency_errors_total",
				Help:      "Total number of dependency errors",
			},
			[]string{"service", "dependency", "operation"},
		),

		// Transaction metrics
		TransactionCount: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: "transaction",
				Name:      "total",
				Help:      "Total number of transactions processed",
			},
			[]string{"type", "status"},
		),

		TransactionAmount: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: "transaction",
				Name:      "amount",
				Help:      "Transaction amount distribution",
				Buckets:   []float64{1, 10, 100, 1000, 10000, 100000},
			},
			[]string{"type"},
		),

		TransactionDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: "transaction",
				Name:      "duration_seconds",
				Help:      "Transaction processing duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"type"},
		),

		TransactionErrorCount: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: "transaction",
				Name:      "errors_total",
				Help:      "Total number of transaction errors",
			},
			[]string{"type", "code"},
		),

		// Order book metrics
		OrderCount: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: "order",
				Name:      "total",
				Help:      "Total number of orders processed",
			},
			[]string{"type", "status"},
		),

		OrderAmount: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: "order",
				Name:      "amount",
				Help:      "Order amount distribution",
				Buckets:   []float64{1, 10, 100, 1000, 10000, 100000},
			},
			[]string{"type"},
		),

		OrderDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: cfg.Namespace,
				Subsystem: "order",
				Name:      "duration_seconds",
				Help:      "Order processing duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"type"},
		),

		OrderErrorCount: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: "order",
				Name:      "errors_total",
				Help:      "Total number of order errors",
			},
			[]string{"type", "code"},
		),

		OrderBookDepth: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: "orderbook",
				Name:      "depth",
				Help:      "Current depth of the order book",
			},
			[]string{"side"},
		),

		// Supply metrics
		TotalSupply: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: "supply",
				Name:      "total",
				Help:      "Total supply of the currency",
			},
		),

		InflationRate: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: "supply",
				Name:      "inflation_rate",
				Help:      "Current inflation rate",
			},
		),

		SupplyChanges: factory.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: cfg.Namespace,
				Subsystem: "supply",
				Name:      "changes_total",
				Help:      "Total number of supply changes",
			},
			[]string{"type"},
		),

		ReserveBalance: factory.NewGauge(
			prometheus.GaugeOpts{
				Namespace: cfg.Namespace,
				Subsystem: "supply",
				Name:      "reserve_balance",
				Help:      "Current balance of the reserve account",
			},
		),
	}

	// Set initial values
	m.ServiceLastStarted.Set(float64(time.Now().Unix()))

	return m
}

// Handler returns an HTTP handler for exposing metrics.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

// RecordUptime starts a goroutine that updates the service uptime metric.
func (m *Metrics) RecordUptime(done <-chan struct{}) {
	startTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				m.ServiceUptime.Set(time.Since(startTime).Seconds())
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
}

// RecordRequest records metrics for an HTTP request.
func (m *Metrics) RecordRequest(service, method, path string, status int, duration time.Duration) {
	m.RequestCount.WithLabelValues(service, method, path, http.StatusText(status)).Inc()
	m.RequestDuration.WithLabelValues(service, method, path).Observe(duration.Seconds())
}

// RecordError records an error metric.
func (m *Metrics) RecordError(service, errorType, errorCode string) {
	m.ErrorCount.WithLabelValues(service, errorType, errorCode).Inc()
}

// RecordDependencyStatus records the status of a dependency.
func (m *Metrics) RecordDependencyStatus(service, dependency string, up bool) {
	var value float64
	if up {
		value = 1
	}
	m.DependencyUp.WithLabelValues(service, dependency).Set(value)
}

// RecordDependencyLatency records the latency of a dependency operation.
func (m *Metrics) RecordDependencyLatency(service, dependency, operation string, duration time.Duration) {
	m.DependencyLatency.WithLabelValues(service, dependency, operation).Observe(duration.Seconds())
}

// RecordDependencyError records an error with a dependency.
func (m *Metrics) RecordDependencyError(service, dependency, operation string) {
	m.DependencyErrorRate.WithLabelValues(service, dependency, operation).Inc()
}

// RecordTransaction records metrics for a transaction.
func (m *Metrics) RecordTransaction(txType, status string, amount float64, duration time.Duration) {
	m.TransactionCount.WithLabelValues(txType, status).Inc()
	m.TransactionAmount.WithLabelValues(txType).Observe(amount)
	m.TransactionDuration.WithLabelValues(txType).Observe(duration.Seconds())
}

// RecordTransactionError records a transaction error.
func (m *Metrics) RecordTransactionError(txType, errorCode string) {
	m.TransactionErrorCount.WithLabelValues(txType, errorCode).Inc()
}

// RecordOrder records metrics for an order.
func (m *Metrics) RecordOrder(orderType, status string, amount float64, duration time.Duration) {
	m.OrderCount.WithLabelValues(orderType, status).Inc()
	m.OrderAmount.WithLabelValues(orderType).Observe(amount)
	m.OrderDuration.WithLabelValues(orderType).Observe(duration.Seconds())
}

// RecordOrderError records an order error.
func (m *Metrics) RecordOrderError(orderType, errorCode string) {
	m.OrderErrorCount.WithLabelValues(orderType, errorCode).Inc()
}

// RecordOrderBookDepth records the current depth of the order book.
func (m *Metrics) RecordOrderBookDepth(side string, depth float64) {
	m.OrderBookDepth.WithLabelValues(side).Set(depth)
}

// RecordTotalSupply records the total supply of the currency.
func (m *Metrics) RecordTotalSupply(supply float64) {
	m.TotalSupply.Set(supply)
}

// RecordInflationRate records the current inflation rate.
func (m *Metrics) RecordInflationRate(rate float64) {
	m.InflationRate.Set(rate)
}

// RecordSupplyChange records a change in the supply.
func (m *Metrics) RecordSupplyChange(changeType string) {
	m.SupplyChanges.WithLabelValues(changeType).Inc()
}

// RecordReserveBalance records the current balance of the reserve account.
func (m *Metrics) RecordReserveBalance(balance float64) {
	m.ReserveBalance.Set(balance)
}
