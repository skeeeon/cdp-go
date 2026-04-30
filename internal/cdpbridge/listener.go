package cdpbridge

import (
	"context"
	"fmt"
	"log/slog"
	"net"
)

// listen joins the configured CDP multicast group and forwards each
// datagram to out. Each forwarded slice is a fresh copy, safe to retain.
//
// Returns when ctx is canceled or an unrecoverable read error occurs.
func listen(ctx context.Context, cfg *Config, out chan<- []byte) error {
	groupIP := net.ParseIP(cfg.Group)
	if groupIP == nil {
		return fmt.Errorf("listener: invalid group IP %q", cfg.Group)
	}
	if !groupIP.IsMulticast() {
		return fmt.Errorf("listener: group %q is not a multicast address", cfg.Group)
	}

	var iface *net.Interface
	if cfg.Iface != "" {
		i, err := net.InterfaceByName(cfg.Iface)
		if err != nil {
			return fmt.Errorf("listener: interface %q: %w", cfg.Iface, err)
		}
		iface = i
	}

	conn, err := net.ListenMulticastUDP("udp4", iface, &net.UDPAddr{IP: groupIP, Port: cfg.Port})
	if err != nil {
		return fmt.Errorf("listener: ListenMulticastUDP: %w", err)
	}
	defer conn.Close()

	if err := conn.SetReadBuffer(1 << 20); err != nil {
		slog.Warn("listener: SetReadBuffer", "err", err)
	}

	slog.Info("listening", "group", cfg.Group, "port", cfg.Port, "iface", cfg.Iface)

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	buf := make([]byte, 65536)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("listener: ReadFromUDP: %w", err)
		}
		datagram := make([]byte, n)
		copy(datagram, buf[:n])

		select {
		case out <- datagram:
		case <-ctx.Done():
			return nil
		}
	}
}
