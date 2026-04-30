package cdpbridge

import (
	"bytes"
	stdjson "encoding/json"
	"testing"

	goccyjson "github.com/goccy/go-json"
	"github.com/velociti/cdp-go/internal/geofence"
	"github.com/velociti/cdp-go/pkg/cdp"
)

// jsonSink defeats dead-code elimination in the benchmark loops.
var jsonSink []byte

func fixtureEnvelopePositionV3() envelope {
	return envelope{
		Type:     "0x0135",
		TypeName: "PositionV3",
		Packet: envelopePacket{
			Sequence:     0xDEADBEEF,
			SenderSerial: cdp.Serial(0x12345678),
		},
		Data: &cdp.PositionV3{
			SerialNumber: cdp.Serial(0xABCDEF01),
			NetworkTime:  0x0123456789ABCDEF,
			X:            1234567,
			Y:            -987654,
			Z:            3210,
			Quality:      95,
			AnchorCount:  4,
			Flags:        0x01,
			Smoothing:    100,
		},
	}
}

func fixtureEnvelopeAccelerometerV2() envelope {
	return envelope{
		Type:     "0x0139",
		TypeName: "AccelerometerV2",
		Packet: envelopePacket{
			Sequence:     0xCAFEBABE,
			SenderSerial: cdp.Serial(0x12345678),
		},
		Data: &cdp.AccelerometerV2{
			SerialNumber: cdp.Serial(0xABCDEF01),
			NetworkTime:  0x0123456789ABCDEF,
			X:            1024,
			Y:            -2048,
			Z:            16384,
			Scale:        2,
		},
	}
}

func fixtureEnvelopeUnknown() envelope {
	return envelope{
		Type:     "0xFFFF",
		TypeName: "Unknown",
		Packet: envelopePacket{
			Sequence:     0x01020304,
			SenderSerial: cdp.Serial(0x12345678),
		},
		Data: &cdp.Unknown{
			TypeID: 0xFFFF,
			Raw:    []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77},
		},
	}
}

func fixtureGeofenceEvent() geofence.Event {
	return geofence.Event{
		Type:        geofence.EventEnter,
		Tag:         cdp.Serial(0x12345678),
		Zone:        "Loading Bay 3",
		ZoneSlug:    "loading_bay_3",
		InZones:     []string{"Loading Bay 3", "Warehouse Floor", "Building A"},
		NetworkTime: 0x0123456789ABCDEF,
		Position:    geofence.Point{X: 1234567, Y: -987654},
		Color:       geofence.RGB{R: 255, G: 128, B: 64},
	}
}

// benchMarshal runs the same payload through stdlib and goccy so a single
// `go test -bench=.` invocation yields a comparable table.
func benchMarshal(b *testing.B, x any) {
	b.Helper()
	b.Run("stdlib", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			out, err := stdjson.Marshal(x)
			if err != nil {
				b.Fatal(err)
			}
			jsonSink = out
		}
	})
	b.Run("goccy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			out, err := goccyjson.Marshal(x)
			if err != nil {
				b.Fatal(err)
			}
			jsonSink = out
		}
	})
}

func BenchmarkMarshalEnvelopePositionV3(b *testing.B) {
	benchMarshal(b, fixtureEnvelopePositionV3())
}

func BenchmarkMarshalEnvelopeAccelerometerV2(b *testing.B) {
	benchMarshal(b, fixtureEnvelopeAccelerometerV2())
}

func BenchmarkMarshalEnvelopeUnknown(b *testing.B) {
	benchMarshal(b, fixtureEnvelopeUnknown())
}

func BenchmarkMarshalGeofenceEvent(b *testing.B) {
	benchMarshal(b, fixtureGeofenceEvent())
}

// TestEncodersAgreeOnByteOutput pins byte-equivalence between stdlib and
// goccy on every payload the bridge marshals. A future goccy upgrade that
// silently changes default behavior (HTML escape, number precision, etc.)
// will fail this test rather than reach production.
func TestEncodersAgreeOnByteOutput(t *testing.T) {
	cases := []struct {
		name  string
		value any
	}{
		{"envelope_position_v3", fixtureEnvelopePositionV3()},
		{"envelope_accelerometer_v2", fixtureEnvelopeAccelerometerV2()},
		{"envelope_unknown", fixtureEnvelopeUnknown()},
		{"geofence_event", fixtureGeofenceEvent()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			std, err := stdjson.Marshal(tc.value)
			if err != nil {
				t.Fatalf("stdjson.Marshal: %v", err)
			}
			goc, err := goccyjson.Marshal(tc.value)
			if err != nil {
				t.Fatalf("goccyjson.Marshal: %v", err)
			}
			if !bytes.Equal(std, goc) {
				t.Errorf("encoder outputs differ:\nstdlib: %s\ngoccy:  %s", std, goc)
			}
		})
	}
}
