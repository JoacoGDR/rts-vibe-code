package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Registry bundles every Prometheus collector the platform exposes.
type Registry struct {
	prom *prometheus.Registry

	HTTPRequests    *prometheus.CounterVec
	HTTPDuration    *prometheus.HistogramVec
	WSConnections   prometheus.Gauge
	WSMessagesIn    prometheus.Counter
	WSMessagesOut   prometheus.Counter
	CommandsApplied *prometheus.CounterVec
	EngineEvents    *prometheus.CounterVec
}

// New builds a Registry tagged with the binary mode (`core-api`, `gateway`,
// `engine`, `worker`).
func New(mode string) *Registry {
	prom := prometheus.NewRegistry()
	prom.MustRegister(collectors.NewGoCollector())
	prom.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))

	r := &Registry{
		prom: prom,
		HTTPRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "http_requests_total", Help: "Total HTTP requests handled",
		}, []string{"method", "route", "status"}),
		HTTPDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "http_request_duration_seconds", Help: "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route"}),
		WSConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "ws_connections", Help: "Active WebSocket connections",
		}),
		WSMessagesIn: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "ws_messages_in_total", Help: "Inbound WebSocket messages",
		}),
		WSMessagesOut: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "ws_messages_out_total", Help: "Outbound WebSocket messages",
		}),
		CommandsApplied: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "commands_applied_total", Help: "Engine commands applied (success or stale)",
		}, []string{"command", "result"}),
		EngineEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "supremacy", Subsystem: mode,
			Name: "engine_events_total", Help: "Timeline events processed by type",
		}, []string{"event"}),
	}
	prom.MustRegister(
		r.HTTPRequests, r.HTTPDuration,
		r.WSConnections, r.WSMessagesIn, r.WSMessagesOut,
		r.CommandsApplied, r.EngineEvents,
	)
	return r
}

// Handler returns the http.Handler that exposes /metrics for Prometheus to
// scrape.
func (r *Registry) Handler() http.Handler {
	return promhttp.HandlerFor(r.prom, promhttp.HandlerOpts{Registry: r.prom})
}

// Serve starts a dedicated HTTP listener for /metrics and blocks until ctx
// is cancelled or the server fails.
func (r *Registry) Serve(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", r.Handler())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
