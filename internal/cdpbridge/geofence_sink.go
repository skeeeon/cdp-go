package cdpbridge

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/velociti/cdp-go/internal/geofence"
)

// natsGeofenceSink publishes one NATS message per geofence event.
//
// Subject: <prefix>.<event_type>.<tag_serial_hex8>.<zone_slug>
// Body:    JSON-marshaled geofence.Event
//
// Note: when NATS is disconnected, Emit drops events silently and
// returns nil. The engine still commits the underlying state
// transition, so consumers will observe a gap in the event stream
// across a NATS outage. That gap is pre-existing and intentional —
// fixing it would require buffering committed transitions until
// publish succeeds, which is outside the scope of this layer.
type natsGeofenceSink struct {
	nc     *nats.Conn
	prefix string
}

func (s *natsGeofenceSink) Emit(ev geofence.Event) error {
	subj := fmt.Sprintf("%s.%s.%s.%s", s.prefix, ev.Type, ev.Tag.Hex(), ev.ZoneSlug)
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	if err := s.nc.Publish(subj, body); err != nil {
		if s.nc.Status() == nats.CONNECTED {
			return err
		}
		// Disconnected: drop quietly. The broker logs disconnect
		// state changes; flooding warns per packet adds no signal.
	}
	return nil
}
