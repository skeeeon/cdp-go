package cdpbridge

import (
	"fmt"
	"log/slog"

	json "github.com/goccy/go-json"
	"github.com/nats-io/nats.go"
	"github.com/skeeeon/cdp-go/internal/geofence"
	"github.com/skeeeon/cdp-go/internal/metrics"
	"github.com/skeeeon/cdp-go/pkg/cdp"
)

// envelope is the JSON object published per data item.
type envelope struct {
	Type     string         `json:"type"`
	TypeName string         `json:"type_name"`
	Packet   envelopePacket `json:"packet"`
	Data     any            `json:"data"`
}

type envelopePacket struct {
	Sequence     uint32     `json:"sequence"`
	SenderSerial cdp.Serial `json:"sender_serial"`
}

// publish decodes one CDP datagram and emits one NATS message per data item.
// When engine is non-nil, each PositionV3 item is also fed to the geofence
// engine after publication.
//
// Decode errors are returned (per-datagram failure); per-item publish errors
// are logged but do not abort the rest of the items in the same datagram.
func publish(nc *nats.Conn, prefix string, data []byte, engine *geofence.Engine) error {
	metrics.DatagramsReceived.Inc()
	pkt, err := cdp.Decode(data)
	if err != nil {
		metrics.DecodeErrors.Inc()
		return fmt.Errorf("decode: %w", err)
	}

	senderHex := pkt.Sender.Hex()
	for _, it := range pkt.Items {
		subj := fmt.Sprintf("%s.%s.%s", prefix, it.Subject, senderHex)
		body, err := json.Marshal(envelope{
			Type:     it.TypeHex,
			TypeName: it.Name,
			Packet: envelopePacket{
				Sequence:     pkt.Sequence,
				SenderSerial: pkt.Sender,
			},
			Data: it.Payload,
		})
		if err != nil {
			metrics.PublishErrors.Inc()
			slog.Warn("marshal", "subject", subj, "err", err)
			continue
		}
		if err := nc.Publish(subj, body); err != nil {
			metrics.PublishErrors.Inc()
			// Demote to debug when we already know NATS is down — the
			// broker's DisconnectErrHandler/ReconnectHandler logs
			// already carry the operational signal, so per-publish
			// warns at thousands/sec just flood stderr.
			if nc.Status() == nats.CONNECTED {
				slog.Warn("publish", "subject", subj, "err", err)
			} else {
				slog.Debug("publish (disconnected)", "subject", subj, "err", err)
			}
		} else {
			metrics.ItemsPublished.Inc()
		}

		if engine != nil {
			if pos, ok := it.Payload.(*cdp.PositionV3); ok {
				engine.OnPosition(pkt.Sender, pos)
			}
		}
	}
	return nil
}
