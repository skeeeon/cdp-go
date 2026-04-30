package geofence

import (
	"log/slog"
	"sort"

	"github.com/velociti/cdp-go/pkg/cdp"
)

// ZoneSet is a stable comparable representation of "which zones contain
// the tag right now": a slice of zone names sorted alphabetically.
type ZoneSet []string

// Equal reports whether two ZoneSets contain the same names. Both must be
// pre-sorted (callers in this package always pass sorted slices).
func (z ZoneSet) Equal(other ZoneSet) bool {
	if len(z) != len(other) {
		return false
	}
	for i := range z {
		if z[i] != other[i] {
			return false
		}
	}
	return true
}

// Engine is the geofence brain. It owns the static zone list and per-tag
// hysteresis state, and emits commit events through a sink.
//
// Engine is NOT goroutine-safe. The bridge already serializes packets
// through a single goroutine reading from a channel, so concurrent
// OnPosition calls don't happen by construction.
type Engine struct {
	zones      []*Zone
	hysteresis int
	sink       EventSink
	tags       map[cdp.Serial]*tagState
}

type tagState struct {
	committed    ZoneSet
	pending      ZoneSet
	pendingCount int
}

// NewEngine constructs an Engine with the given zones, hysteresis count,
// and event sink. hysteresis < 1 is treated as 1 (commit immediately on
// the first observed change).
func NewEngine(zones []*Zone, hysteresis int, sink EventSink) *Engine {
	if hysteresis < 1 {
		hysteresis = 1
	}
	return &Engine{
		zones:      zones,
		hysteresis: hysteresis,
		sink:       sink,
		tags:       make(map[cdp.Serial]*tagState),
	}
}

// OnPosition processes one position update from one tag. Returns true if
// a state transition was committed (and at least one event emitted).
func (e *Engine) OnPosition(serial cdp.Serial, pos *cdp.PositionV3) bool {
	p := Point{X: pos.X, Y: pos.Y}
	proposed := e.contains(p)

	st, ok := e.tags[serial]
	if !ok {
		st = &tagState{}
		e.tags[serial] = st
	}

	switch {
	case proposed.Equal(st.committed):
		// Settled state. Drop any pending.
		st.pending = nil
		st.pendingCount = 0
		return false
	case st.pending == nil || !proposed.Equal(st.pending):
		// New candidate; restart the count.
		st.pending = proposed
		st.pendingCount = 1
	default:
		st.pendingCount++
	}

	if st.pendingCount < e.hysteresis {
		return false
	}

	// Commit.
	e.emit(serial, pos, st.committed, proposed)
	st.committed = proposed
	st.pending = nil
	st.pendingCount = 0
	return true
}

// contains returns the sorted set of zones that contain p.
func (e *Engine) contains(p Point) ZoneSet {
	var hits ZoneSet
	for _, z := range e.zones {
		if z.Contains(p) {
			hits = append(hits, z.Name)
		}
	}
	sort.Strings(hits)
	return hits
}

// emit publishes one event per zone added or removed in the transition,
// exits first then enters. The committed set passed to each event is the
// state *after* the transition.
func (e *Engine) emit(serial cdp.Serial, pos *cdp.PositionV3, prev, next ZoneSet) {
	exits, enters := diffZoneSets(prev, next)

	point := Point{X: pos.X, Y: pos.Y}
	inZones := append([]string(nil), next...) // defensive copy for the event payload

	for _, name := range exits {
		ev := Event{
			Type:        EventExit,
			Tag:         serial,
			Zone:        name,
			ZoneSlug:    e.slugOf(name),
			InZones:     inZones,
			NetworkTime: pos.NetworkTime,
			Position:    point,
			Color:       e.colorOf(name),
		}
		if err := e.sink.Emit(ev); err != nil {
			slog.Warn("geofence emit", "type", "exit", "tag", serial, "zone", name, "err", err)
		}
	}
	for _, name := range enters {
		ev := Event{
			Type:        EventEnter,
			Tag:         serial,
			Zone:        name,
			ZoneSlug:    e.slugOf(name),
			InZones:     inZones,
			NetworkTime: pos.NetworkTime,
			Position:    point,
			Color:       e.colorOf(name),
		}
		if err := e.sink.Emit(ev); err != nil {
			slog.Warn("geofence emit", "type", "enter", "tag", serial, "zone", name, "err", err)
		}
	}
}

// diffZoneSets returns (exits, enters): names in prev but not next, and
// names in next but not prev. Both inputs must be sorted; outputs are sorted.
func diffZoneSets(prev, next ZoneSet) (exits, enters []string) {
	i, j := 0, 0
	for i < len(prev) && j < len(next) {
		switch {
		case prev[i] == next[j]:
			i++
			j++
		case prev[i] < next[j]:
			exits = append(exits, prev[i])
			i++
		default:
			enters = append(enters, next[j])
			j++
		}
	}
	exits = append(exits, prev[i:]...)
	enters = append(enters, next[j:]...)
	return
}

func (e *Engine) slugOf(name string) string {
	for _, z := range e.zones {
		if z.Name == name {
			return z.Slug
		}
	}
	return ""
}

func (e *Engine) colorOf(name string) RGB {
	for _, z := range e.zones {
		if z.Name == name {
			return z.Color
		}
	}
	return RGB{}
}
