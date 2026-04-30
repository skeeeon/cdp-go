// Package cdpbridge is the binary-specific glue for cdp-nats-bridge:
// it joins a CDP multicast group, decodes each packet's data items via
// pkg/cdp, and publishes them as JSON over NATS.
package cdpbridge

import (
	"context"
	"log/slog"

	"github.com/velociti/cdp-go/internal/broker"
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
		if err := nc.Drain(); err != nil {
			slog.Warn("nats drain", "err", err)
		}
	}()

	packets := make(chan []byte, 256)
	listenErr := make(chan error, 1)
	go func() { listenErr <- listen(ctx, cfg, packets) }()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-listenErr:
			return err
		case data := <-packets:
			if err := publish(nc, cfg.Prefix, data); err != nil {
				slog.Warn("publish_packet", "err", err)
			}
		}
	}
}
