package consul

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusMetrics provides Consul-related metrics for Prometheus
type PrometheusMetrics struct {
	serviceRegistrations   prometheus.Counter
	serviceDeregistrations prometheus.Counter
	serviceDiscoveries     prometheus.Counter
	consulHealthChecks     prometheus.Gauge
	serviceHealthStatus    *prometheus.GaugeVec
}

// NewPrometheusMetrics creates Consul metrics for Prometheus
func NewPrometheusMetrics() *PrometheusMetrics {
	return &PrometheusMetrics{
		serviceRegistrations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "consul_service_registrations_total",
			Help: "Total number of service registrations with Consul",
		}),
		serviceDeregistrations: promauto.NewCounter(prometheus.CounterOpts{
			Name: "consul_service_deregistrations_total",
			Help: "Total number of service deregistrations from Consul",
		}),
		serviceDiscoveries: promauto.NewCounter(prometheus.CounterOpts{
			Name: "consul_service_discoveries_total",
			Help: "Total number of service discovery requests",
		}),
		consulHealthChecks: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "consul_health_status",
			Help: "Consul cluster health status (1=healthy, 0=unhealthy)",
		}),
		serviceHealthStatus: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "consul_service_health_status",
			Help: "Health status of services in Consul (1=passing, 0=failing)",
		}, []string{"service_name", "service_id", "protocol"}),
	}
}

// Enhanced Service Registry with Prometheus integration
type ServiceRegistryWithMetrics struct {
	*ServiceRegistry
	metrics *PrometheusMetrics
}

// NewServiceRegistryWithMetrics creates a service registry with Prometheus metrics
func NewServiceRegistryWithMetrics(config *Config) (*ServiceRegistryWithMetrics, error) {
	registry, err := NewServiceRegistry(config)
	if err != nil {
		return nil, err
	}

	metrics := NewPrometheusMetrics()

	enhanced := &ServiceRegistryWithMetrics{
		ServiceRegistry: registry,
		metrics:         metrics,
	}

	// Start metrics collection
	go enhanced.startMetricsCollection()

	return enhanced, nil
}

// RegisterService with metrics tracking
func (sr *ServiceRegistryWithMetrics) RegisterService(ctx context.Context, serviceName, serviceType string) (*ServiceInfo, error) {
	service, err := sr.ServiceRegistry.RegisterService(ctx, serviceName, serviceType)
	if err != nil {
		return nil, err
	}

	// Update metrics
	sr.metrics.serviceRegistrations.Inc()
	sr.metrics.serviceHealthStatus.WithLabelValues(
		service.Name,
		service.ID,
		service.Protocol,
	).Set(1) // Initially set as healthy

	return service, nil
}

// DiscoverService with metrics tracking
func (sr *ServiceRegistryWithMetrics) DiscoverService(serviceName string) (*ServiceInfo, error) {
	service, err := sr.ServiceRegistry.DiscoverService(serviceName)
	if err != nil {
		return nil, err
	}

	// Update metrics
	sr.metrics.serviceDiscoveries.Inc()

	return service, nil
}

// DeregisterService with metrics tracking
func (sr *ServiceRegistryWithMetrics) DeregisterService(serviceName string) error {
	// Get service info before deregistration for metrics
	service, _ := sr.ServiceRegistry.DiscoverService(serviceName)

	err := sr.ServiceRegistry.DeregisterService(serviceName)
	if err != nil {
		return err
	}

	// Update metrics
	sr.metrics.serviceDeregistrations.Inc()
	if service != nil {
		sr.metrics.serviceHealthStatus.WithLabelValues(
			service.Name,
			service.ID,
			service.Protocol,
		).Set(0) // Mark as unhealthy/removed
	}

	return nil
}

// startMetricsCollection starts collecting Consul and service metrics
func (sr *ServiceRegistryWithMetrics) startMetricsCollection() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sr.updateConsulMetrics()
		}
	}
}

// updateConsulMetrics updates Consul health and service metrics
func (sr *ServiceRegistryWithMetrics) updateConsulMetrics() {
	// Check Consul health
	if err := sr.ServiceRegistry.Health(); err != nil {
		sr.metrics.consulHealthChecks.Set(0)
		return
	}
	sr.metrics.consulHealthChecks.Set(1)

	// Update service health metrics
	services, err := sr.ServiceRegistry.GetAllServices()
	if err != nil {
		return
	}

	for _, service := range services {
		healthValue := 0.0
		if service.Health == "passing" {
			healthValue = 1.0
		}

		sr.metrics.serviceHealthStatus.WithLabelValues(
			service.Name,
			service.ID,
			service.Protocol,
		).Set(healthValue)
	}
}
