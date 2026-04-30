// Package broker is the shared NATS connection helper for every binary
// in this repo. It owns the option-builder + connect logic; callers get
// a *nats.Conn back and Publish on it directly.
package broker

import (
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/velociti/cdp-go/internal/config"
)

// Connect builds the nats.Option list from cfg and dials the server.
//
// Defaults that match nats.go's own defaults are passed through unchanged
// so we don't override them inadvertently. Connection-state changes are
// logged via slog so operators can see disconnects/reconnects.
func Connect(cfg config.Broker) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.Name(cfg.Name),
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.ReconnectJitter(cfg.ReconnectJitter, cfg.ReconnectJitter),
		nats.PingInterval(cfg.PingInterval),
		nats.MaxPingsOutstanding(cfg.MaxPingsOut),
		nats.DrainTimeout(cfg.DrainTimeout),
		nats.FlusherTimeout(cfg.FlushTimeout),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			slog.Warn("nats disconnected", "err", err)
		}),
		nats.ReconnectHandler(func(c *nats.Conn) {
			slog.Info("nats reconnected", "url", c.ConnectedUrl())
		}),
		nats.ClosedHandler(func(c *nats.Conn) {
			slog.Info("nats closed", "last_err", c.LastError())
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			slog.Warn("nats async error", "err", err)
		}),
	}

	if cfg.User != "" {
		opts = append(opts, nats.UserInfo(cfg.User, cfg.Password))
	}
	if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	}
	if cfg.CredsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.CredsFile))
	}
	if cfg.NkeySeedFile != "" {
		nkeyOpt, err := nats.NkeyOptionFromSeed(cfg.NkeySeedFile)
		if err != nil {
			return nil, fmt.Errorf("broker: nkey seed: %w", err)
		}
		opts = append(opts, nkeyOpt)
	}
	if cfg.TLSCA != "" {
		opts = append(opts, nats.RootCAs(cfg.TLSCA))
	}
	if cfg.TLSCert != "" && cfg.TLSKey != "" {
		opts = append(opts, nats.ClientCert(cfg.TLSCert, cfg.TLSKey))
	}
	if cfg.TLSInsecure {
		opts = append(opts, nats.Secure(&tls.Config{InsecureSkipVerify: true}))
	}
	if cfg.NoEcho {
		opts = append(opts, nats.NoEcho())
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("broker: connect: %w", err)
	}
	slog.Info("nats connected", "url", nc.ConnectedUrl())
	return nc, nil
}
