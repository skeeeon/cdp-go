package geofence

import (
	"reflect"
	"testing"
	"time"

	"github.com/velociti/cdp-go/pkg/cdp"
)

// recordingSink captures every emitted event, in order. Used by every
// engine test; reset via .reset() between scenarios.
type recordingSink struct {
	events []Event
}

func (r *recordingSink) Emit(ev Event) error {
	r.events = append(r.events, ev)
	return nil
}

func (r *recordingSink) reset() { r.events = nil }

// types extracts just the event types in order. Useful for asserting
// "exit X then enter Y" without rewriting the full event payload.
func (r *recordingSink) types() []EventType {
	out := make([]EventType, len(r.events))
	for i, e := range r.events {
		out[i] = e.Type
	}
	return out
}

// zoneNames extracts the Zone field of each event, in order.
func (r *recordingSink) zoneNames() []string {
	out := make([]string, len(r.events))
	for i, e := range r.events {
		out[i] = e.Zone
	}
	return out
}

// twoSquares: two non-overlapping zones used for most tests. A is to the
// left, B is to the right.
func twoSquares(t *testing.T) []*Zone {
	t.Helper()
	a, err := NewZone("A", []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}, RGB{255, 0, 0})
	if err != nil {
		t.Fatalf("NewZone A: %v", err)
	}
	b, err := NewZone("B", []Point{{20, 0}, {30, 0}, {30, 10}, {20, 10}}, RGB{0, 255, 0})
	if err != nil {
		t.Fatalf("NewZone B: %v", err)
	}
	return []*Zone{a, b}
}

// overlappingSquares: two zones that overlap in the middle, used for
// multi-zone-membership tests. A spans x=[0,10], B spans x=[5,15].
func overlappingSquares(t *testing.T) []*Zone {
	t.Helper()
	a, err := NewZone("A", []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}, RGB{255, 0, 0})
	if err != nil {
		t.Fatalf("NewZone A: %v", err)
	}
	b, err := NewZone("B", []Point{{5, 0}, {15, 0}, {15, 10}, {5, 10}}, RGB{0, 255, 0})
	if err != nil {
		t.Fatalf("NewZone B: %v", err)
	}
	return []*Zone{a, b}
}

// pos builds a synthetic PositionV3 at (x, y) for tag tests.
func pos(x, y int32) *cdp.PositionV3 {
	return &cdp.PositionV3{X: x, Y: y, NetworkTime: 100}
}

func TestEngineHysteresisN1CommitsImmediately(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 1, 0, sink)
	tag := cdp.Serial(0xAABBCCDD)

	e.OnPosition(tag, pos(5, 5)) // inside A
	if got := sink.types(); !reflect.DeepEqual(got, []EventType{EventEnter}) {
		t.Fatalf("expected one enter on first packet with N=1, got %v", got)
	}
	if sink.events[0].Zone != "A" {
		t.Errorf("expected enter A, got %s", sink.events[0].Zone)
	}
}

func TestEngineHysteresisN0BehavesLikeN1(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 0, 0, sink)
	tag := cdp.Serial(0xAABBCCDD)

	e.OnPosition(tag, pos(5, 5))
	if got := sink.types(); !reflect.DeepEqual(got, []EventType{EventEnter}) {
		t.Fatalf("N=0 should commit immediately, got %v", got)
	}
}

func TestEngineHysteresisN5StableCommit(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 5, 0, sink)
	tag := cdp.Serial(1)

	for i := 0; i < 4; i++ {
		e.OnPosition(tag, pos(5, 5))
		if len(sink.events) != 0 {
			t.Fatalf("packet %d: premature commit", i+1)
		}
	}
	e.OnPosition(tag, pos(5, 5)) // 5th
	if got := sink.types(); !reflect.DeepEqual(got, []EventType{EventEnter}) {
		t.Fatalf("expected one enter after 5 stable packets, got %v", got)
	}
}

func TestEngineHysteresisPrematureSwitchResets(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 5, 0, sink)
	tag := cdp.Serial(1)

	// Three packets in A.
	for i := 0; i < 3; i++ {
		e.OnPosition(tag, pos(5, 5))
	}
	// Two packets outside any zone — proposed becomes []. That's a new
	// candidate state, so the count for "in A" is dropped, and []
	// starts at count=1 then 2.
	for i := 0; i < 2; i++ {
		e.OnPosition(tag, pos(50, 50))
	}
	if len(sink.events) != 0 {
		t.Fatalf("no commit expected; got %v", sink.events)
	}
}

func TestEngineHysteresisOscillationNeverCommits(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 3, 0, sink)
	tag := cdp.Serial(1)

	// A, B, A, B, A, B alternating — pending always becomes the new
	// proposed, count never reaches 3.
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			e.OnPosition(tag, pos(5, 5)) // inside A
		} else {
			e.OnPosition(tag, pos(25, 5)) // inside B
		}
	}
	if len(sink.events) != 0 {
		t.Fatalf("oscillation must not commit; got %v", sink.events)
	}
}

func TestEngineMultiZoneOverlap(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(overlappingSquares(t), 1, 0, sink)
	tag := cdp.Serial(1)

	e.OnPosition(tag, pos(7, 5)) // in both A and B (A=[0,10], B=[5,15])
	if len(sink.events) != 2 {
		t.Fatalf("expected 2 enter events, got %d: %v", len(sink.events), sink.events)
	}
	for _, ev := range sink.events {
		if ev.Type != EventEnter {
			t.Errorf("expected enter, got %s", ev.Type)
		}
	}
	got := sink.zoneNames()
	if !reflect.DeepEqual(got, []string{"A", "B"}) {
		t.Errorf("expected [A B], got %v", got)
	}
	// Both events should report in_zones = [A, B].
	for _, ev := range sink.events {
		if !reflect.DeepEqual([]string(ev.InZones), []string{"A", "B"}) {
			t.Errorf("expected in_zones=[A B], got %v", ev.InZones)
		}
	}
}

func TestEngineExitThenReEnter(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 1, 0, sink)
	tag := cdp.Serial(1)

	e.OnPosition(tag, pos(5, 5))   // enter A
	e.OnPosition(tag, pos(50, 50)) // exit A
	e.OnPosition(tag, pos(5, 5))   // re-enter A

	if got := sink.types(); !reflect.DeepEqual(got, []EventType{EventEnter, EventExit, EventEnter}) {
		t.Fatalf("expected enter,exit,enter; got %v", got)
	}
}

func TestEngineSubsetTransitionExitOnly(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(overlappingSquares(t), 1, 0, sink)
	tag := cdp.Serial(1)

	e.OnPosition(tag, pos(7, 5)) // commits enter A and enter B
	sink.reset()

	e.OnPosition(tag, pos(3, 5)) // now only in A: should emit exit B only
	if got := sink.types(); !reflect.DeepEqual(got, []EventType{EventExit}) {
		t.Fatalf("expected single exit, got %v", got)
	}
	if sink.events[0].Zone != "B" {
		t.Errorf("expected exit B, got exit %s", sink.events[0].Zone)
	}
	if !reflect.DeepEqual([]string(sink.events[0].InZones), []string{"A"}) {
		t.Errorf("expected in_zones=[A], got %v", sink.events[0].InZones)
	}
}

func TestEngineExitOrderBeforeEnter(t *testing.T) {
	// Committed [A] -> proposed [B]: must emit exit A *then* enter B.
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 1, 0, sink)
	tag := cdp.Serial(1)

	e.OnPosition(tag, pos(5, 5))  // commit enter A
	sink.reset()
	e.OnPosition(tag, pos(25, 5)) // jump to B

	if got := sink.types(); !reflect.DeepEqual(got, []EventType{EventExit, EventEnter}) {
		t.Fatalf("expected exit before enter, got %v", got)
	}
	if sink.events[0].Zone != "A" {
		t.Errorf("expected exit A, got %s", sink.events[0].Zone)
	}
	if sink.events[1].Zone != "B" {
		t.Errorf("expected enter B, got %s", sink.events[1].Zone)
	}
}

func TestEngineTagsAreIndependent(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 3, 0, sink)
	stable := cdp.Serial(1)
	osc := cdp.Serial(2)

	// Drive both tags together: stable always in A, osc alternates.
	for i := 0; i < 9; i++ {
		e.OnPosition(stable, pos(5, 5))
		if i%2 == 0 {
			e.OnPosition(osc, pos(5, 5))
		} else {
			e.OnPosition(osc, pos(25, 5))
		}
	}

	// Stable tag should commit enter A exactly once. Oscillator never.
	stableEvents := 0
	for _, ev := range sink.events {
		if ev.Tag == stable {
			stableEvents++
		}
		if ev.Tag == osc {
			t.Errorf("oscillating tag should not commit, but got: %v", ev)
		}
	}
	if stableEvents != 1 {
		t.Errorf("expected 1 commit for stable tag, got %d", stableEvents)
	}
}

func TestEngineEventCarriesPositionAndColor(t *testing.T) {
	sink := &recordingSink{}
	e := NewEngine(twoSquares(t), 1, 0, sink)
	tag := cdp.Serial(0x01020304)

	p := &cdp.PositionV3{X: 5, Y: 7, NetworkTime: 999}
	e.OnPosition(tag, p)

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	ev := sink.events[0]
	if ev.Position != (Point{5, 7}) {
		t.Errorf("position: got %v, want {5,7}", ev.Position)
	}
	if ev.NetworkTime != 999 {
		t.Errorf("network_time: got %d, want 999", ev.NetworkTime)
	}
	if ev.Tag != tag {
		t.Errorf("tag: got %v, want %v", ev.Tag, tag)
	}
	if ev.Color != (RGB{255, 0, 0}) {
		t.Errorf("color: got %v, want {255 0 0}", ev.Color)
	}
	if ev.ZoneSlug != "a" {
		t.Errorf("zone_slug: got %q, want a", ev.ZoneSlug)
	}
}

// fakeClock returns whatever time the caller assigns to its current
// field. Tests advance time by replacing current.
type fakeClock struct{ current time.Time }

func (c *fakeClock) Now() time.Time { return c.current }

func TestEnginePrunesStaleTags(t *testing.T) {
	sink := &recordingSink{}
	clk := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	e := NewEngine(twoSquares(t), 1, time.Hour, sink)
	e.now = clk.Now
	e.cleanupEvery = 1 // run sweep on every OnPosition for test brevity

	// Tag #1 seen at t=0.
	e.OnPosition(cdp.Serial(1), pos(5, 5))
	if e.TagCount() != 1 {
		t.Fatalf("expected 1 tag, got %d", e.TagCount())
	}

	// Advance 2 hours. Tag #2 packet arrives — should trigger sweep that
	// removes the now-stale tag #1.
	clk.current = clk.current.Add(2 * time.Hour)
	e.OnPosition(cdp.Serial(2), pos(5, 5))
	if e.TagCount() != 1 {
		t.Errorf("expected 1 tag after sweep, got %d", e.TagCount())
	}
	// The remaining tag is #2.
	if _, ok := e.tags[cdp.Serial(2)]; !ok {
		t.Error("expected tag 2 to remain")
	}
	if _, ok := e.tags[cdp.Serial(1)]; ok {
		t.Error("expected tag 1 to be pruned")
	}
}

func TestEngineDoesNotPruneActiveTags(t *testing.T) {
	sink := &recordingSink{}
	clk := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	e := NewEngine(twoSquares(t), 1, time.Hour, sink)
	e.now = clk.Now
	e.cleanupEvery = 1

	e.OnPosition(cdp.Serial(1), pos(5, 5))

	// Each minute, both tags get a packet. After 90 minutes, neither
	// should have been swept (both seen within TTL of each other).
	for i := 0; i < 90; i++ {
		clk.current = clk.current.Add(time.Minute)
		e.OnPosition(cdp.Serial(1), pos(5, 5))
		e.OnPosition(cdp.Serial(2), pos(5, 5))
	}
	if e.TagCount() != 2 {
		t.Errorf("expected 2 tags, got %d", e.TagCount())
	}
}

func TestEngineTagTTLZeroDisablesCleanup(t *testing.T) {
	sink := &recordingSink{}
	clk := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	e := NewEngine(twoSquares(t), 1, 0, sink) // TTL=0 disables sweep
	e.now = clk.Now
	e.cleanupEvery = 1

	e.OnPosition(cdp.Serial(1), pos(5, 5))
	clk.current = clk.current.Add(48 * time.Hour) // far past any reasonable TTL
	e.OnPosition(cdp.Serial(2), pos(5, 5))

	if e.TagCount() != 2 {
		t.Errorf("TTL=0 must not prune; got %d tags", e.TagCount())
	}
}

func TestEnginePruneSkipsClockGoingBackward(t *testing.T) {
	sink := &recordingSink{}
	clk := &fakeClock{current: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	e := NewEngine(twoSquares(t), 1, time.Hour, sink)
	e.now = clk.Now
	e.cleanupEvery = 1

	// Tag seen "in the future".
	e.OnPosition(cdp.Serial(1), pos(5, 5))

	// Clock jumps backward (e.g. wall-clock NTP correction). Sweep
	// should NOT delete the entry just because lastSeen.After(now).
	clk.current = clk.current.Add(-24 * time.Hour)
	e.OnPosition(cdp.Serial(2), pos(5, 5))
	if e.TagCount() != 2 {
		t.Errorf("expected sweep to skip future-stamped tag; got %d", e.TagCount())
	}
}
