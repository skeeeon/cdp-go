// Package cdpbridge is the binary-specific glue for cdp-nats-bridge:
// it joins a CDP multicast group, decodes each packet's data items via
// pkg/cdp, and publishes them as JSON over NATS.
package cdpbridge

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/skeeeon/cdp-go/internal/broker"
	"github.com/skeeeon/cdp-go/internal/geofence"
	"github.com/skeeeon/cdp-go/internal/metrics"
)

// Run is the bridge orchestrator. It dials NATS, listens on the multicast
// group, and publishes each decoded packet's items.
//
// Returns when ctx is canceled (clean shutdown) or when listener/connect
// setup fails.
func Run(ctx context.Context, cfg *Config) error {
	nc, err := broker.Connect(cfg.Broker)
	if err != nil {
		return err
	}
	defer func() {
		// FlushTimeout blocks until pending messages are sent or the
		// timeout fires; Drain is async and inappropriate for a
		// publish-only client.
		if err := nc.FlushTimeout(cfg.Broker.FlushTimeout); err != nil {
			slog.Warn("nats flush", "err", err)
		}
		nc.Close()
	}()

	var engine *geofence.Engine
	if cfg.Geofence.Enabled() {
		zones, err := cfg.Geofence.Build()
		if err != nil {
			return fmt.Errorf("geofence: %w", err)
		}
		engine = geofence.NewEngine(zones, cfg.Geofence.Hysteresis, cfg.Geofence.TagTTL,
			&natsGeofenceSink{nc: nc, prefix: cfg.Geofence.Prefix})
		slog.Info("geofence enabled",
			"zones", len(zones),
			"hysteresis", cfg.Geofence.Hysteresis,
			"tag_ttl", cfg.Geofence.TagTTL,
			"prefix", cfg.Geofence.Prefix)
	}

	packets := make(chan []byte, 2048)
	listenErr := make(chan error, 1)
	go func() { listenErr <- listen(ctx, cfg, packets) }()

	if cfg.Metrics.Addr != "" {
		var tagCount func() int
		if engine != nil {
			tagCount = engine.TagCount
		}
		go metrics.PollGauges(ctx,
			func() int { return len(packets) },
			tagCount,
			func() bool { return nc.Status() == nats.CONNECTED },
		)
		go func() {
			if err := metrics.Serve(ctx, cfg.Metrics.Addr); err != nil {
				// Metrics failure is non-fatal: the bridge keeps
				// pumping CDP→NATS even if the scrape endpoint is dead.
				slog.Error("metrics server", "addr", cfg.Metrics.Addr, "err", err)
			}
		}()
		slog.Info("metrics endpoint", "addr", cfg.Metrics.Addr, "path", "/metrics")
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-listenErr:
			return err
		case data := <-packets:
			if err := publish(nc, cfg.Prefix, data, engine); err != nil {
				slog.Warn("publish_packet", "err", err)
			}
		}
	}
}
