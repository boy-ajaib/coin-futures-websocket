package server

import (
	"net/http"
	"time"

	"github.com/centrifugal/centrifuge"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds Prometheus metrics for the Centrifuge server
type Metrics struct {
	// Connection metrics
	connectionsTotal  *prometheus.CounterVec
	connectionsActive prometheus.Gauge
	connectionsFailed *prometheus.CounterVec

	// Channel metrics
	channelsTotal       prometheus.Gauge
	subscriptionsTotal  *prometheus.CounterVec
	subscriptionsActive prometheus.Gauge

	// Message metrics
	messagesPublished *prometheus.CounterVec
	messagesReceived  *prometheus.CounterVec

	// Server metrics
	nodeInfo *prometheus.GaugeVec
}

// NewMetrics creates a new Metrics instance with Prometheus collectors
func NewMetrics(node *centrifuge.Node) *Metrics {
	m := &Metrics{
		// Connection metrics
		connectionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "centrifuge_connections_total",
				Help: "Total number of connections",
			},
			[]string{"node"},
		),
		connectionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "centrifuge_connections_active",
				Help: "Number of active connections",
			},
		),
		connectionsFailed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "centrifuge_connections_failed_total",
				Help: "Total number of failed connections",
			},
			[]string{"node", "reason"},
		),

		// Channel metrics
		channelsTotal: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "centrifuge_channels_total",
				Help: "Number of active channels",
			},
		),
		subscriptionsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "centrifuge_subscriptions_total",
				Help: "Total number of subscriptions",
			},
			[]string{"node", "channel"},
		),
		subscriptionsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "centrifuge_subscriptions_active",
				Help: "Number of active subscriptions",
			},
		),

		// Message metrics
		messagesPublished: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "centrifuge_messages_published_total",
				Help: "Total number of messages published",
			},
			[]string{"node", "channel"},
		),
		messagesReceived: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "centrifuge_messages_received_total",
				Help: "Total number of messages received by clients",
			},
			[]string{"node"},
		),

		// Server metrics
		nodeInfo: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "centrifuge_node_info",
				Help: "Information about the Centrifuge node",
			},
			[]string{"node_name", "namespace", "version"},
		),
	}

	// Initialize node info with default values
	m.nodeInfo.WithLabelValues("unknown", "default", "unknown").Set(1)

	return m
}

// Register registers all metrics with the default Prometheus registry
func (m *Metrics) Register() error {
	registry := prometheus.DefaultRegisterer

	registry.MustRegister(
		m.connectionsTotal,
		m.connectionsActive,
		m.connectionsFailed,
		m.channelsTotal,
		m.subscriptionsTotal,
		m.subscriptionsActive,
		m.messagesPublished,
		m.messagesReceived,
		m.nodeInfo,
	)

	return nil
}

// RecordConnection records a new connection
func (m *Metrics) RecordConnection(nodeName string) {
	m.connectionsTotal.WithLabelValues(nodeName).Inc()
	m.connectionsActive.Inc()
}

// RecordDisconnection records a disconnection
func (m *Metrics) RecordDisconnection(nodeName string) {
	m.connectionsActive.Dec()
}

// RecordFailedConnection records a failed connection attempt
func (m *Metrics) RecordFailedConnection(nodeName, reason string) {
	m.connectionsFailed.WithLabelValues(nodeName, reason).Inc()
}

// RecordSubscription records a new subscription
func (m *Metrics) RecordSubscription(nodeName, channel string) {
	m.subscriptionsTotal.WithLabelValues(nodeName, channel).Inc()
	m.subscriptionsActive.Inc()
}

// RecordUnsubscription records an unsubscription
func (m *Metrics) RecordUnsubscription() {
	m.subscriptionsActive.Dec()
}

// RecordPublication records a message publication
func (m *Metrics) RecordPublication(nodeName, channel string) {
	m.messagesPublished.WithLabelValues(nodeName, channel).Inc()
}

// UpdateMetrics updates metrics from the current node state
func (m *Metrics) UpdateMetrics(node *centrifuge.Node, nodeName string) {
	if node == nil {
		return
	}

	// Update active connections
	m.connectionsActive.Set(float64(node.Hub().NumClients()))

	// Update active subscriptions (estimate from client connections)
	m.subscriptionsActive.Set(float64(node.Hub().NumClients()))
}

// MetricsHandler returns an HTTP handler for the metrics endpoint
func (s *CentrifugeServer) MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// StartMetricsCollector starts a background goroutine to collect metrics periodically
func (s *CentrifugeServer) StartMetricsCollector(metrics *Metrics, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			metrics.UpdateMetrics(s.node, s.config.NodeName)
		}
	}()
}

// MetricsMiddleware wraps the HTTP handler to track connection metrics
func (s *CentrifugeServer) MetricsMiddleware(metrics *Metrics, nodeName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Track connection metrics
			metrics.RecordConnection(nodeName)

			// Call the next handler
			next.ServeHTTP(w, r)

			// Note: Disconnection is tracked via the Disconnect handler
		})
	}
}

// SetupMetricsHandler registers the metrics endpoint with the given ServeMux
func (s *CentrifugeServer) SetupMetricsHandler(mux *http.ServeMux, path string) {
	mux.Handle(path, s.MetricsHandler())
}
