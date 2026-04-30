// Package geofence detects when CDP-tracked tags enter and exit configured
// 2D polygon zones, with count-based hysteresis to suppress single-packet
// flicker.
//
// The package is pure logic: it consumes decoded *cdp.PositionV3 values
// and emits Events through an EventSink. Wiring those events to NATS (or
// any other transport) is the caller's responsibility.
package geofence

import "github.com/velociti/cdp-go/pkg/cdp"

// EventType is the kind of zone-membership transition that produced an Event.
type EventType string

const (
	EventEnter EventType = "enter"
	EventExit  EventType = "exit"
)

// Event is one committed zone-membership transition for a single tag.
//
// InZones is the full committed zone set after the transition, sorted by
// name. A consumer can read the latest membership directly off any event
// without maintaining its own state.
type Event struct {
	Type        EventType  `json:"type"`
	Tag         cdp.Serial `json:"tag"`
	TagHex      string     `json:"-"`
	Zone        string     `json:"zone"`
	ZoneSlug    string     `json:"-"`
	InZones     []string   `json:"in_zones"`
	NetworkTime uint64     `json:"network_time"`
	Position    Point      `json:"position"`
	Color       RGB        `json:"color"`
}

// EventSink is the geofence engine's output port.
type EventSink interface {
	Emit(ev Event) error
}
