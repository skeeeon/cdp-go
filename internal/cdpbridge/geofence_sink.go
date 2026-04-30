package cdpbridge

import (
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/velociti/cdp-go/internal/geofence"
)

// natsGeofenceSink publishes one NATS message per geofence event.
//
// Subject: <prefix>.tag.<event_type>.<tag_serial_hex8>.<zone_slug>
// Body:    JSON-marshaled geofence.Event
type natsGeofenceSink struct {
	nc     *nats.Conn
	prefix string
}

func (s *natsGeofenceSink) Emit(ev geofence.Event) error {
	subj := fmt.Sprintf("%s.tag.%s.%s.%s", s.prefix, ev.Type, ev.TagHex, ev.ZoneSlug)
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return s.nc.Publish(subj, body)
}
