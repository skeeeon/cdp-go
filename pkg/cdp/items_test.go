package cdp

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"math"
	"strings"
	"testing"
)

// le is a small builder that appends little-endian values into b.
type le struct{ b []byte }

func (l *le) u8(v uint8) *le  { l.b = append(l.b, v); return l }
func (l *le) i8(v int8) *le   { l.b = append(l.b, byte(v)); return l }
func (l *le) u16(v uint16) *le {
	l.b = binary.LittleEndian.AppendUint16(l.b, v)
	return l
}
func (l *le) i16(v int16) *le {
	l.b = binary.LittleEndian.AppendUint16(l.b, uint16(v))
	return l
}
func (l *le) u32(v uint32) *le {
	l.b = binary.LittleEndian.AppendUint32(l.b, v)
	return l
}
func (l *le) i32(v int32) *le {
	l.b = binary.LittleEndian.AppendUint32(l.b, uint32(v))
	return l
}
func (l *le) u64(v uint64) *le {
	l.b = binary.LittleEndian.AppendUint64(l.b, v)
	return l
}
func (l *le) f32(v float32) *le {
	l.b = binary.LittleEndian.AppendUint32(l.b, math.Float32bits(v))
	return l
}
func (l *le) bytes(p []byte) *le {
	l.b = append(l.b, p...)
	return l
}

func TestDecodePositionV3(t *testing.T) {
	payload := (&le{}).
		u32(0x01020304). // serial_number
		u64(0x1122334455667788). // network_time
		i32(-1234). // x
		i32(5678).  // y
		i32(900).   // z
		u16(9500).  // quality
		u8(4).      // anchor_count
		u8(0x80).   // flags
		u16(7).     // smoothing
		b
	if len(payload) != 30 {
		t.Fatalf("fixture wrong size: %d != 30", len(payload))
	}
	pkt, err := Decode(buildPacket(1, 0x99887766, [][]byte{di(0x0135, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(pkt.Items) != 1 {
		t.Fatalf("items: %d", len(pkt.Items))
	}
	it := pkt.Items[0]
	if it.Subject != "position" {
		t.Errorf("Subject: %s", it.Subject)
	}
	p := it.Payload.(*PositionV3)
	if p.SerialNumber != Serial(0x01020304) {
		t.Errorf("Serial: %x", uint32(p.SerialNumber))
	}
	if p.NetworkTime != 0x1122334455667788 {
		t.Errorf("NetworkTime: %x", p.NetworkTime)
	}
	if p.X != -1234 || p.Y != 5678 || p.Z != 900 {
		t.Errorf("XYZ: %d %d %d", p.X, p.Y, p.Z)
	}
	if p.Quality != 9500 || p.AnchorCount != 4 || p.Flags != 0x80 || p.Smoothing != 7 {
		t.Errorf("trailing fields: q=%d ac=%d f=%x s=%d", p.Quality, p.AnchorCount, p.Flags, p.Smoothing)
	}
}

func TestDecodeAccelerometerV2(t *testing.T) {
	payload := (&le{}).
		u32(0x10000001).
		u64(123456789).
		i32(-100).
		i32(200).
		i32(300).
		u8(8).
		b
	pkt, err := Decode(buildPacket(2, 1, [][]byte{di(0x0139, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	a := pkt.Items[0].Payload.(*AccelerometerV2)
	if a.SerialNumber != Serial(0x10000001) || a.NetworkTime != 123456789 ||
		a.X != -100 || a.Y != 200 || a.Z != 300 || a.Scale != 8 {
		t.Errorf("got %+v", a)
	}
}

func TestDecodeQuaternionV2(t *testing.T) {
	payload := (&le{}).
		u32(0xAABBCCDD).
		u64(1).
		i32(1 << 30).
		i32(0).
		i32(0).
		i32(0).
		u8(2).
		b
	pkt, err := Decode(buildPacket(0, 1, [][]byte{di(0x013D, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	q := pkt.Items[0].Payload.(*QuaternionV2)
	if q.X != 1<<30 || q.QuaternionType != 2 {
		t.Errorf("got %+v", q)
	}
}

func TestDecodeAnchorHealthV5(t *testing.T) {
	// Two bad paired anchors.
	payload := (&le{}).
		u32(0xA0A0A0A0). // serial
		u8(1).            // interface_id
		u32(100).         // ticks_reported
		u32(200).         // timed_rxs
		u32(300).         // beacons_reported
		u32(0).           // beacons_discarded
		u32(0).           // beacons_late
		u16(9000).        // average_quality
		u8(60).           // report_period
		u8(0).            // interanchor_comms_error_code
		// bad_paired_anchors:
		u32(0xB1B1B1B1).u8(1).
		u32(0xB2B2B2B2).u8(2).
		b

	pkt, err := Decode(buildPacket(1, 1, [][]byte{di(0x014A, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	h := pkt.Items[0].Payload.(*AnchorHealthV5)
	if h.SerialNumber != Serial(0xA0A0A0A0) || h.AverageQuality != 9000 || h.ReportPeriod != 60 {
		t.Errorf("got %+v", h)
	}
	if len(h.BadPairedAnchors) != 2 {
		t.Fatalf("bad_paired_anchors: %d", len(h.BadPairedAnchors))
	}
	if h.BadPairedAnchors[0].SerialNumber != Serial(0xB1B1B1B1) || h.BadPairedAnchors[0].InterfaceID != 1 {
		t.Errorf("anchor[0]: %+v", h.BadPairedAnchors[0])
	}
	if h.BadPairedAnchors[1].SerialNumber != Serial(0xB2B2B2B2) || h.BadPairedAnchors[1].InterfaceID != 2 {
		t.Errorf("anchor[1]: %+v", h.BadPairedAnchors[1])
	}
}

func TestDecodeAnchorPositionStatusV3(t *testing.T) {
	payload := (&le{}).
		u32(0x10101010).
		u64(0x2020202020202020).
		// status entry 1
		u32(0x30303030).u8(1).u8(0).i16(-100).i16(-200).u16(8500).
		// status entry 2
		u32(0x40404040).u8(2).u8(3).i16(-150).i16(-250).u16(7500).
		b

	pkt, err := Decode(buildPacket(1, 1, [][]byte{di(0x0136, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	a := pkt.Items[0].Payload.(*AnchorPositionStatusV3)
	if a.TagSerialNumber != Serial(0x10101010) {
		t.Errorf("tag: %x", uint32(a.TagSerialNumber))
	}
	if len(a.AnchorStatusArray) != 2 {
		t.Fatalf("array len: %d", len(a.AnchorStatusArray))
	}
	if a.AnchorStatusArray[0].Quality != 8500 || a.AnchorStatusArray[1].Quality != 7500 {
		t.Errorf("qualities: %+v", a.AnchorStatusArray)
	}
}

func TestDecodeDeviceHardwareStatusV2(t *testing.T) {
	payload := (&le{}).
		u32(0xAABBCCDD).
		u32(1024).
		u32(0x0F).
		u16(120).
		u8(80).
		i8(-5).
		u8(45).
		u8(0xC9). u8(0x06). // two error pattern bytes
		b

	pkt, err := Decode(buildPacket(1, 1, [][]byte{di(0x0138, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	d := pkt.Items[0].Payload.(*DeviceHardwareStatusV2)
	if d.BatteryPercentage != 80 || d.Temperature != -5 || d.ProcessorUsage != 45 {
		t.Errorf("got %+v", d)
	}
	if len(d.ErrorPatterns) != 2 || d.ErrorPatterns[0].Pattern != 0xC9 || d.ErrorPatterns[1].Pattern != 0x06 {
		t.Errorf("error_patterns: %+v", d.ErrorPatterns)
	}
}

func TestDecodePolarCoordinatesV1(t *testing.T) {
	payload := (&le{}).
		u32(1).
		u64(2).
		u32(1500).
		f32(45.5).
		f32(-12.25).
		u16(9999).
		u8(3).
		u8(0).
		u16(7).
		b
	pkt, err := Decode(buildPacket(1, 1, [][]byte{di(0x0164, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	p := pkt.Items[0].Payload.(*PolarCoordinatesV1)
	if p.Theta != 45.5 || p.Phi != -12.25 || p.Rho != 1500 || p.Quality != 9999 {
		t.Errorf("got %+v", p)
	}
}

func TestDecodeDeviceNamesNullStrip(t *testing.T) {
	// "anchor-1\x00\x00\x00" → "anchor-1"
	name := []byte{'a', 'n', 'c', 'h', 'o', 'r', '-', '1', 0, 0, 0}
	payload := (&le{}).u32(0x12345678).bytes(name).b
	pkt, err := Decode(buildPacket(1, 1, [][]byte{di(0x013F, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	d := pkt.Items[0].Payload.(*DeviceNames)
	if d.Name != "anchor-1" {
		t.Errorf("Name: %q", d.Name)
	}
}

func TestEnvelopeMarshalsAsExpected(t *testing.T) {
	// Round-trip: build a real PositionV3, marshal it, sanity-check JSON shape.
	p := &PositionV3{
		SerialNumber: Serial(0x01234567),
		NetworkTime:  100,
		X:            10, Y: 20, Z: 30,
		Quality: 9000, AnchorCount: 4, Flags: 0, Smoothing: 5,
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	js := string(b)
	for _, want := range []string{
		`"serial_number":"01:23:4567"`,
		`"network_time":100`,
		`"x":10`, `"y":20`, `"z":30`,
		`"quality":9000`, `"anchor_count":4`, `"smoothing":5`,
	} {
		if !strings.Contains(js, want) {
			t.Errorf("JSON missing %s\n  got: %s", want, js)
		}
	}
}

func TestUnknownEnvelopeJSON(t *testing.T) {
	u := &Unknown{TypeID: 0xDEAD, Raw: []byte{0xCA, 0xFE}}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	want := `{"type":"0xDEAD","raw":"cafe"}`
	if !bytes.Equal(b, []byte(want)) {
		t.Errorf("got %s, want %s", b, want)
	}
}
