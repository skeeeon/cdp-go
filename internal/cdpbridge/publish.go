package cdpbridge

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/nats-io/nats.go"
	"github.com/velociti/cdp-go/internal/geofence"
	"github.com/velociti/cdp-go/pkg/cdp"
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
	pkt, err := cdp.Decode(data)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	for _, it := range pkt.Items {
		subj := fmt.Sprintf("%s.%s.%s", prefix, it.Subject, pkt.Sender.Hex())
		body, err := json.Marshal(envelope{
			Type:     fmt.Sprintf("0x%04X", it.TypeID),
			TypeName: it.Name,
			Packet: envelopePacket{
				Sequence:     pkt.Sequence,
				SenderSerial: pkt.Sender,
			},
			Data: it.Payload,
		})
		if err != nil {
			slog.Warn("marshal", "subject", subj, "err", err)
			continue
		}
		if err := nc.Publish(subj, body); err != nil {
			// Demote to debug when we already know NATS is down — the
			// broker's DisconnectErrHandler/ReconnectHandler logs
			// already carry the operational signal, so per-publish
			// warns at thousands/sec just flood stderr.
			if nc.Status() == nats.CONNECTED {
				slog.Warn("publish", "subject", subj, "err", err)
			} else {
				slog.Debug("publish (disconnected)", "subject", subj, "err", err)
			}
		}

		if engine != nil {
			if pos, ok := it.Payload.(*cdp.PositionV3); ok {
				engine.OnPosition(pkt.Sender, pos)
			}
		}
	}
	return nil
}
