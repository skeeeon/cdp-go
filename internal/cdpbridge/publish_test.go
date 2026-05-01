package cdpbridge

import (
	"encoding/binary"
	stdjson "encoding/json"
	"testing"
	"time"

	natstest "github.com/nats-io/nats-server/v2/test"
	"github.com/nats-io/nats.go"
	"github.com/skeeeon/cdp-go/internal/geofence"
	"github.com/skeeeon/cdp-go/pkg/cdp"
)

// runTestNATS spawns an in-process NATS server on a random port, returns a
// connected client, and registers cleanup to shut both down at test end.
// Accepts testing.TB so the same helper works for tests and benchmarks.
func runTestNATS(tb testing.TB) *nats.Conn {
	tb.Helper()
	s := natstest.RunRandClientPortServer()
	tb.Cleanup(s.Shutdown)
	nc, err := nats.Connect(s.ClientURL())
	if err != nil {
		tb.Fatalf("nats.Connect: %v", err)
	}
	tb.Cleanup(nc.Close)
	return nc
}

// buildPositionV3Datagram constructs a minimal wire-format CDP packet
// carrying one PositionV3 data item, suitable for handing to publish().
func buildPositionV3Datagram(seq uint32, sender cdp.Serial, pos cdp.PositionV3) []byte {
	buf := make([]byte, cdp.HeaderSize+cdp.DataItemHeaderSize+30)
	binary.LittleEndian.PutUint32(buf[0:4], cdp.MarkerValue)
	binary.LittleEndian.PutUint32(buf[4:8], seq)
	copy(buf[8:16], []byte("CDP0002\x00"))
	binary.LittleEndian.PutUint32(buf[16:20], uint32(sender))

	binary.LittleEndian.PutUint16(buf[20:22], 0x0135)
	binary.LittleEndian.PutUint16(buf[22:24], 30)

	p := buf[24:54]
	binary.LittleEndian.PutUint32(p[0:4], uint32(pos.SerialNumber))
	binary.LittleEndian.PutUint64(p[4:12], pos.NetworkTime)
	binary.LittleEndian.PutUint32(p[12:16], uint32(pos.X))
	binary.LittleEndian.PutUint32(p[16:20], uint32(pos.Y))
	binary.LittleEndian.PutUint32(p[20:24], uint32(pos.Z))
	binary.LittleEndian.PutUint16(p[24:26], pos.Quality)
	p[26] = pos.AnchorCount
	p[27] = pos.Flags
	binary.LittleEndian.PutUint16(p[28:30], pos.Smoothing)
	return buf
}

// recordingSink captures emitted geofence events for assertion.
type recordingSink struct {
	events []geofence.Event
}

func (r *recordingSink) Emit(ev geofence.Event) error {
	r.events = append(r.events, ev)
	return nil
}

func TestPublish_PositionV3Item(t *testing.T) {
	nc := runTestNATS(t)

	sub, err := nc.SubscribeSync("cdp.position.>")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	sender := cdp.Serial(0x12345678)
	pos := cdp.PositionV3{
		SerialNumber: cdp.Serial(0xABCDEF01),
		NetworkTime:  0x0123456789ABCDEF,
		X:            1000, Y: 2000, Z: 3000,
		Quality: 95, AnchorCount: 4, Flags: 0x01, Smoothing: 100,
	}
	datagram := buildPositionV3Datagram(0xDEADBEEF, sender, pos)
	if err := publish(nc, "cdp", datagram, nil); err != nil {
		t.Fatalf("publish: %v", err)
	}

	msg, err := sub.NextMsg(time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	if want := "cdp.position." + sender.Hex(); msg.Subject != want {
		t.Errorf("subject: got %q, want %q", msg.Subject, want)
	}

	var env struct {
		Type     string `json:"type"`
		TypeName string `json:"type_name"`
		Packet   struct {
			Sequence     uint32 `json:"sequence"`
			SenderSerial string `json:"sender_serial"`
		} `json:"packet"`
		Data struct {
			X       int32  `json:"x"`
			Y       int32  `json:"y"`
			Z       int32  `json:"z"`
			Quality uint16 `json:"quality"`
		} `json:"data"`
	}
	if err := stdjson.Unmarshal(msg.Data, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Type != "0x0135" {
		t.Errorf("type: got %q, want 0x0135", env.Type)
	}
	if env.TypeName != "PositionV3" {
		t.Errorf("type_name: got %q, want PositionV3", env.TypeName)
	}
	if env.Packet.Sequence != 0xDEADBEEF {
		t.Errorf("sequence: got 0x%X, want 0xDEADBEEF", env.Packet.Sequence)
	}
	if env.Packet.SenderSerial != sender.String() {
		t.Errorf("sender_serial: got %q, want %q", env.Packet.SenderSerial, sender.String())
	}
	if env.Data.X != 1000 || env.Data.Y != 2000 || env.Data.Z != 3000 || env.Data.Quality != 95 {
		t.Errorf("data fields: got X=%d Y=%d Z=%d Quality=%d", env.Data.X, env.Data.Y, env.Data.Z, env.Data.Quality)
	}

	// No second message should arrive — the datagram had exactly one item.
	if msg, err := sub.NextMsg(50 * time.Millisecond); err == nil {
		t.Errorf("unexpected second message on subject %q: %s", msg.Subject, msg.Data)
	}
}

func TestPublish_DecodeError(t *testing.T) {
	nc := runTestNATS(t)
	if err := publish(nc, "cdp", []byte("not a valid CDP packet"), nil); err == nil {
		t.Fatal("expected error on malformed datagram, got nil")
	}
}

func TestPublish_FeedsGeofenceEngine(t *testing.T) {
	nc := runTestNATS(t)

	zone, err := geofence.NewZone(
		"test_zone",
		[]geofence.Point{
			{X: 0, Y: 0},
			{X: 10000, Y: 0},
			{X: 10000, Y: 10000},
			{X: 0, Y: 10000},
		},
		geofence.RGB{R: 0xFF},
	)
	if err != nil {
		t.Fatalf("NewZone: %v", err)
	}

	sink := &recordingSink{}
	eng := geofence.NewEngine([]*geofence.Zone{zone}, 1, 0, sink)

	// Position inside the zone — engine should emit one Enter event.
	pos := cdp.PositionV3{
		SerialNumber: cdp.Serial(0xABCDEF01),
		X:            5000, Y: 5000, Z: 0,
	}
	datagram := buildPositionV3Datagram(1, cdp.Serial(0x12345678), pos)
	if err := publish(nc, "cdp", datagram, eng); err != nil {
		t.Fatalf("publish: %v", err)
	}

	if len(sink.events) != 1 {
		t.Fatalf("expected 1 geofence event, got %d", len(sink.events))
	}
	ev := sink.events[0]
	if ev.Type != geofence.EventEnter {
		t.Errorf("event type: got %q, want enter", ev.Type)
	}
	if ev.Zone != "test_zone" {
		t.Errorf("event zone: got %q, want test_zone", ev.Zone)
	}
	if ev.Tag != cdp.Serial(0x12345678) {
		t.Errorf("event tag: got %x, want 0x12345678", uint32(ev.Tag))
	}
}

// BenchmarkPublish_PositionV3 measures the full bridge publish path
// (Decode → envelope marshal → NATS Publish) end-to-end against an
// in-process NATS server. Includes broker overhead, which is realistic.
// nats.Conn.Publish is async/buffered, so this measures enqueue cost
// rather than synchronous delivery — matching production behavior.
func BenchmarkPublish_PositionV3(b *testing.B) {
	nc := runTestNATS(b)
	sender := cdp.Serial(0x12345678)
	pos := cdp.PositionV3{
		SerialNumber: cdp.Serial(0xABCDEF01),
		NetworkTime:  0x0123456789ABCDEF,
		X:            1000, Y: 2000, Z: 3000,
		Quality: 95, AnchorCount: 4, Flags: 0x01, Smoothing: 100,
	}
	datagram := buildPositionV3Datagram(0xDEADBEEF, sender, pos)

	b.SetBytes(int64(len(datagram)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := publish(nc, "cdp", datagram, nil); err != nil {
			b.Fatal(err)
		}
	}
}
