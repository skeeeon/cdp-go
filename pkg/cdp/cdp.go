// Package cdp decodes Ciholas Data Protocol UDP packets into typed Go values.
//
// CDP is a little-endian UDP wire format used by Ciholas RTLS hardware to
// stream positions, IMU data, anchor health, and device telemetry. A packet
// has a 20-byte header followed by zero or more (type, size, payload)
// data items.
//
// This is a port of the parts of cdp-py we need; only a subset of the ~98
// data item types is decoded into typed structs. Unknown types are returned
// as Unknown values carrying their raw payload.
package cdp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// HeaderSize is the fixed CDP packet header length.
const HeaderSize = 20

// DataItemHeaderSize is the (type, size) prefix length on each data item.
const DataItemHeaderSize = 4

// MarkerValue is the magic value at the start of every CDP packet.
const MarkerValue uint32 = 0x3230434C

// Recognized values for the 8-byte string field in the header.
var (
	stringCDP0002 = [8]byte{'C', 'D', 'P', '0', '0', '0', '2', 0}
	stringLCMSelf = [8]byte{'L', 'C', 'M', '_', 'S', 'E', 'L', 'F'}
)

// Serial is a 32-bit Ciholas device serial number.
//
// String formats as "XX:XX:XXXX" (the canonical display format). Hex returns
// an 8-character lowercase concatenated form suitable for use as a NATS
// subject token (NATS subjects forbid colons).
type Serial uint32

// String renders the serial as "XX:XX:XXXX".
func (s Serial) String() string {
	return fmt.Sprintf("%02X:%02X:%04X", uint32(s)>>24, (uint32(s)>>16)&0xff, uint32(s)&0xffff)
}

// Hex renders the serial as 8 lowercase hex characters with no separators.
func (s Serial) Hex() string {
	return fmt.Sprintf("%08x", uint32(s))
}

// MarshalJSON emits the canonical "XX:XX:XXXX" string form.
func (s Serial) MarshalJSON() ([]byte, error) {
	return []byte(`"` + s.String() + `"`), nil
}

// Item is one decoded data item from a packet.
//
// Payload is a pointer to the typed struct registered for TypeID (e.g.
// *PositionV3), or *Unknown if no decoder is registered. TypeHex is the
// "0xNNNN" string form of TypeID, precomputed so per-item publishers can
// avoid Sprintf on the hot path.
type Item struct {
	TypeID  uint16
	TypeHex string
	Name    string
	Subject string
	Payload any
}

// Packet is a decoded CDP packet.
type Packet struct {
	Sequence uint32
	Sender   Serial
	Items    []Item
}

// registryEntry holds the metadata + decoder for one known type ID.
type registryEntry struct {
	name    string
	subject string
	decode  func(payload []byte) (any, error)
}

// registry maps CDP data-item type IDs to their decoder. It is populated
// once at package init from the table in items.go.
var registry = map[uint16]registryEntry{}

// typeHexes caches the "0xNNNN" string form for every registered type ID,
// populated in init() after the registry table is built. Decode reads
// these to avoid an Sprintf per known item on the hot path. Unknown
// (unregistered) types fall through to fmt.Sprintf in unknownTypeHex.
var typeHexes = map[uint16]string{}

// Decode parses a single CDP UDP datagram.
//
// Returns an error for an unrecognized marker, an unrecognized header
// string, or a truncated packet. Unknown data item types are not errors —
// they appear as *Unknown values in the returned Items slice.
func Decode(buf []byte) (*Packet, error) {
	if len(buf) < HeaderSize {
		return nil, fmt.Errorf("cdp: packet too short: %d bytes", len(buf))
	}

	marker := binary.LittleEndian.Uint32(buf[0:4])
	if marker != MarkerValue {
		return nil, fmt.Errorf("cdp: bad marker 0x%08X", marker)
	}

	var str [8]byte
	copy(str[:], buf[8:16])
	if str != stringCDP0002 && str != stringLCMSelf {
		return nil, fmt.Errorf("cdp: bad header string %q", str[:])
	}

	// Cap is the upper bound on items (a zero-payload item is the smallest
	// possible), clamped to a sensible ceiling so a pathological packet
	// can't allocate a huge slice up front.
	itemsCap := (len(buf) - HeaderSize) / DataItemHeaderSize
	if itemsCap > 32 {
		itemsCap = 32
	}
	pkt := &Packet{
		Sequence: binary.LittleEndian.Uint32(buf[4:8]),
		Sender:   Serial(binary.LittleEndian.Uint32(buf[16:20])),
		Items:    make([]Item, 0, itemsCap),
	}

	idx := HeaderSize
	for len(buf)-idx >= DataItemHeaderSize {
		typeID := binary.LittleEndian.Uint16(buf[idx : idx+2])
		size := int(binary.LittleEndian.Uint16(buf[idx+2 : idx+4]))
		idx += DataItemHeaderSize

		if len(buf)-idx < size {
			return nil, fmt.Errorf("cdp: truncated data item type=0x%04X size=%d remaining=%d",
				typeID, size, len(buf)-idx)
		}
		payload := buf[idx : idx+size]
		idx += size

		entry, ok := registry[typeID]
		if !ok {
			pkt.Items = append(pkt.Items, Item{
				TypeID:  typeID,
				TypeHex: unknownTypeHex(typeID),
				Name:    unknownName(typeID),
				Subject: unknownSubject(typeID),
				Payload: &Unknown{TypeID: typeID, Raw: append([]byte(nil), payload...)},
			})
			continue
		}

		decoded, err := entry.decode(payload)
		if err != nil {
			return nil, fmt.Errorf("cdp: decode type=0x%04X: %w", typeID, err)
		}
		pkt.Items = append(pkt.Items, Item{
			TypeID:  typeID,
			TypeHex: typeHexes[typeID],
			Name:    entry.name,
			Subject: entry.subject,
			Payload: decoded,
		})
	}

	if len(buf)-idx != 0 {
		return nil, fmt.Errorf("cdp: %d trailing bytes after last data item", len(buf)-idx)
	}

	return pkt, nil
}

// readers — small helpers used by the per-item decoders. These are
// the only shared decoding helpers; everything else is hand-rolled
// in the per-item functions to keep things obvious.

// errShort is returned by per-item decoders when the payload is shorter
// than the fixed-size portion of the item.
var errShort = errors.New("payload too short")

func u8(b []byte, off int) uint8   { return b[off] }
func i8(b []byte, off int) int8    { return int8(b[off]) }
func u16(b []byte, off int) uint16 { return binary.LittleEndian.Uint16(b[off : off+2]) }
func i16(b []byte, off int) int16  { return int16(binary.LittleEndian.Uint16(b[off : off+2])) }
func u32(b []byte, off int) uint32 { return binary.LittleEndian.Uint32(b[off : off+4]) }
func i32(b []byte, off int) int32  { return int32(binary.LittleEndian.Uint32(b[off : off+4])) }
func u64(b []byte, off int) uint64 { return binary.LittleEndian.Uint64(b[off : off+8]) }

func f32(b []byte, off int) float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(b[off : off+4]))
}
