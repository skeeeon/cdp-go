package cdp

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// buildPacket creates a CDP packet with the given header fields and item TLVs.
func buildPacket(seq uint32, sender uint32, items [][]byte) []byte {
	var buf bytes.Buffer
	_ = binary.Write(&buf, binary.LittleEndian, MarkerValue)
	_ = binary.Write(&buf, binary.LittleEndian, seq)
	buf.Write([]byte{'C', 'D', 'P', '0', '0', '0', '2', 0})
	_ = binary.Write(&buf, binary.LittleEndian, sender)
	for _, it := range items {
		buf.Write(it)
	}
	return buf.Bytes()
}

// di builds a single (type, size, payload) TLV.
func di(typeID uint16, payload []byte) []byte {
	out := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint16(out[0:2], typeID)
	binary.LittleEndian.PutUint16(out[2:4], uint16(len(payload)))
	copy(out[4:], payload)
	return out
}

func TestSerialFormatting(t *testing.T) {
	s := Serial(0x01234567)
	if got, want := s.String(), "01:23:4567"; got != want {
		t.Errorf("String: got %q, want %q", got, want)
	}
	if got, want := s.Hex(), "01234567"; got != want {
		t.Errorf("Hex: got %q, want %q", got, want)
	}
	jb, err := s.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if got, want := string(jb), `"01:23:4567"`; got != want {
		t.Errorf("MarshalJSON: got %s, want %s", got, want)
	}
}

func TestDecodeEmptyPacket(t *testing.T) {
	pkt, err := Decode(buildPacket(42, 0xDEADBEEF, nil))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if pkt.Sequence != 42 {
		t.Errorf("sequence: got %d, want 42", pkt.Sequence)
	}
	if pkt.Sender != Serial(0xDEADBEEF) {
		t.Errorf("sender: got %x, want DEADBEEF", uint32(pkt.Sender))
	}
	if len(pkt.Items) != 0 {
		t.Errorf("items: got %d, want 0", len(pkt.Items))
	}
}

func TestDecodeBadMarker(t *testing.T) {
	pkt := buildPacket(1, 1, nil)
	pkt[0] = 0x00
	if _, err := Decode(pkt); err == nil {
		t.Fatal("expected error for bad marker, got nil")
	}
}

func TestDecodeShortPacket(t *testing.T) {
	if _, err := Decode([]byte{0x01, 0x02, 0x03}); err == nil {
		t.Fatal("expected error for short packet, got nil")
	}
}

func TestDecodeUnknownTypeFallsBack(t *testing.T) {
	payload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	pkt, err := Decode(buildPacket(1, 0x01020304, [][]byte{di(0xABCD, payload)}))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if len(pkt.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(pkt.Items))
	}
	it := pkt.Items[0]
	if it.TypeID != 0xABCD {
		t.Errorf("TypeID: got 0x%04X, want 0xABCD", it.TypeID)
	}
	if it.Subject != "unknown_abcd" {
		t.Errorf("Subject: got %q, want unknown_abcd", it.Subject)
	}
	u, ok := it.Payload.(*Unknown)
	if !ok {
		t.Fatalf("Payload: got %T, want *Unknown", it.Payload)
	}
	if !bytes.Equal(u.Raw, payload) {
		t.Errorf("Unknown.Raw: got %x, want %x", u.Raw, payload)
	}
}

func TestDecodeTrailingBytesFails(t *testing.T) {
	// Build a valid packet then append junk.
	pkt := buildPacket(1, 1, nil)
	pkt = append(pkt, 0xFF)
	if _, err := Decode(pkt); err == nil {
		t.Fatal("expected error for trailing bytes, got nil")
	}
}
