// Package metrics defines the bridge's Prometheus metrics and the HTTP
// endpoint that exposes them.
//
// All metric vars are registered once at package init via promauto into the
// default registry, so the standard go_* and process_* collectors come along
// for free. Counter increments are atomic and cheap enough to leave on the
// hot path unconditionally; the HTTP endpoint is opt-in via cfg.Metrics.Addr.
package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	DatagramsReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cdp_datagrams_received_total",
		Help: "Total UDP datagrams received from the multicast group.",
	})
	DecodeErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cdp_decode_errors_total",
		Help: "Total CDP datagrams that failed to decode.",
	})
	ItemsPublished = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cdp_items_published_total",
		Help: "Total CDP data items successfully published to NATS.",
	})
	PublishErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "cdp_publish_errors_total",
		Help: "Total NATS publish or marshal failures, across item and geofence streams.",
	})
	GeofenceEvents = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "cdp_geofence_events_total",
		Help: "Total geofence events successfully published, labeled by type.",
	}, []string{"type"})

	NATSConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cdp_nats_connected",
		Help: "1 if the NATS connection reports CONNECTED, 0 otherwise.",
	})
	PacketQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cdp_packet_queue_depth",
		Help: "Current depth of the listener-to-publisher channel.",
	})
	GeofenceTagsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "cdp_geofence_tags_active",
		Help: "Number of tags currently tracked by the geofence engine.",
	})
)

// Serve runs an HTTP server on addr exposing /metrics. Returns when ctx is
// canceled or ListenAndServe fails. Bind failures do not kill the bridge;
// the caller logs and continues.
func Serve(ctx context.Context, addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

// PollGauges periodically samples each non-nil closure and updates the
// corresponding gauge. Returns when ctx is canceled. One Tick per second is
// plenty for operational gauges and avoids touching shared state on the hot
// path.
func PollGauges(ctx context.Context, queueDepth, geofenceTags func() int, natsConnected func() bool) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if queueDepth != nil {
				PacketQueueDepth.Set(float64(queueDepth()))
			}
			if geofenceTags != nil {
				GeofenceTagsActive.Set(float64(geofenceTags()))
			}
			if natsConnected != nil {
				if natsConnected() {
					NATSConnected.Set(1)
				} else {
					NATSConnected.Set(0)
				}
			}
		}
	}
}

