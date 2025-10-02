package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// OpenUSPMetrics holds all metrics for OpenUSP services
type OpenUSPMetrics struct {
	// Common metrics
	ServiceInfo       *prometheus.GaugeVec
	RequestsTotal     *prometheus.CounterVec
	RequestDuration   *prometheus.HistogramVec
	ActiveConnections prometheus.Gauge
	ErrorsTotal       *prometheus.CounterVec

	// USP specific metrics
	USPMessagesTotal *prometheus.CounterVec
	USPMessageErrors *prometheus.CounterVec
	DevicesOnboarded prometheus.Counter

	// Data service metrics
	DatabaseOperations *prometheus.CounterVec
	DatabaseErrors     *prometheus.CounterVec

	// gRPC metrics
	GRPCRequestsTotal *prometheus.CounterVec
	GRPCDuration      *prometheus.HistogramVec
}

// NewOpenUSPMetrics creates a new metrics instance
func NewOpenUSPMetrics(serviceName string) *OpenUSPMetrics {
	metrics := &OpenUSPMetrics{
		ServiceInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "openusp_service_info",
				Help: "Information about OpenUSP service",
			},
			[]string{"service", "version", "build_time"},
		),

		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_requests_total",
				Help: "Total number of requests processed",
			},
			[]string{"service", "method", "endpoint", "status"},
		),

		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "openusp_request_duration_seconds",
				Help:    "Request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method", "endpoint"},
		),

		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "openusp_active_connections",
				Help: "Number of active connections",
			},
		),

		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_errors_total",
				Help: "Total number of errors",
			},
			[]string{"service", "type", "error"},
		),

		USPMessagesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_usp_messages_total",
				Help: "Total number of USP messages processed",
			},
			[]string{"service", "version", "message_type", "transport"},
		),

		USPMessageErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_usp_message_errors_total",
				Help: "Total number of USP message errors",
			},
			[]string{"service", "version", "message_type", "error_type"},
		),

		DevicesOnboarded: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "openusp_devices_onboarded_total",
				Help: "Total number of devices successfully onboarded",
			},
		),

		DatabaseOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_database_operations_total",
				Help: "Total number of database operations",
			},
			[]string{"service", "operation", "table", "status"},
		),

		DatabaseErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_database_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"service", "operation", "table", "error_type"},
		),

		GRPCRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "openusp_grpc_requests_total",
				Help: "Total number of gRPC requests",
			},
			[]string{"service", "method", "status"},
		),

		GRPCDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "openusp_grpc_duration_seconds",
				Help:    "gRPC request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		metrics.ServiceInfo,
		metrics.RequestsTotal,
		metrics.RequestDuration,
		metrics.ActiveConnections,
		metrics.ErrorsTotal,
		metrics.USPMessagesTotal,
		metrics.USPMessageErrors,
		metrics.DevicesOnboarded,
		metrics.DatabaseOperations,
		metrics.DatabaseErrors,
		metrics.GRPCRequestsTotal,
		metrics.GRPCDuration,
	)

	// Set service info
	metrics.ServiceInfo.WithLabelValues(serviceName, "1.1.0", time.Now().Format("2006-01-02T15:04:05Z")).Set(1)

	return metrics
}

// HTTPHandler returns the Prometheus HTTP handler
func HTTPHandler() http.Handler {
	return promhttp.Handler()
}

// RecordHTTPRequest records HTTP request metrics
func (m *OpenUSPMetrics) RecordHTTPRequest(service, method, endpoint, status string, duration time.Duration) {
	m.RequestsTotal.WithLabelValues(service, method, endpoint, status).Inc()
	m.RequestDuration.WithLabelValues(service, method, endpoint).Observe(duration.Seconds())
}

// RecordUSPMessage records USP message metrics
func (m *OpenUSPMetrics) RecordUSPMessage(service, version, messageType, transport string) {
	m.USPMessagesTotal.WithLabelValues(service, version, messageType, transport).Inc()
}

// RecordUSPError records USP error metrics
func (m *OpenUSPMetrics) RecordUSPError(service, version, messageType, errorType string) {
	m.USPMessageErrors.WithLabelValues(service, version, messageType, errorType).Inc()
}

// RecordDeviceOnboarded increments device onboarded counter
func (m *OpenUSPMetrics) RecordDeviceOnboarded() {
	m.DevicesOnboarded.Inc()
}

// RecordDatabaseOperation records database operation metrics
func (m *OpenUSPMetrics) RecordDatabaseOperation(service, operation, table, status string) {
	m.DatabaseOperations.WithLabelValues(service, operation, table, status).Inc()
}

// RecordDatabaseError records database error metrics
func (m *OpenUSPMetrics) RecordDatabaseError(service, operation, table, errorType string) {
	m.DatabaseErrors.WithLabelValues(service, operation, table, errorType).Inc()
}

// RecordGRPCRequest records gRPC request metrics
func (m *OpenUSPMetrics) RecordGRPCRequest(service, method, status string, duration time.Duration) {
	m.GRPCRequestsTotal.WithLabelValues(service, method, status).Inc()
	m.GRPCDuration.WithLabelValues(service, method).Observe(duration.Seconds())
}

// SetActiveConnections sets the active connections gauge
func (m *OpenUSPMetrics) SetActiveConnections(count float64) {
	m.ActiveConnections.Set(count)
}

// RecordError records general error metrics
func (m *OpenUSPMetrics) RecordError(service, errorType, error string) {
	m.ErrorsTotal.WithLabelValues(service, errorType, error).Inc()
}
