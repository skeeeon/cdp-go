// Command cdp-nats-bridge listens for Ciholas Data Protocol UDP multicast
// traffic, decodes each packet's data items into JSON, and publishes them
// onto NATS.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/skeeeon/cdp-go/internal/cdpbridge"
	"github.com/skeeeon/cdp-go/internal/logger"
)

func main() {
	cfg, err := cdpbridge.LoadConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	logger.Setup(cfg.Logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cdpbridge.Run(ctx, cfg); err != nil {
		slog.Error("bridge stopped with error", "err", err)
		os.Exit(1)
	}
}
