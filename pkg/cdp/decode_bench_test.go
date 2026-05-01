package cdp

import (
	"encoding/binary"
	"testing"
)

// samplePositionV3Datagram constructs a wire-format CDP packet carrying
// one PositionV3 item. Used as a fixed fixture for parser benchmarks.
func samplePositionV3Datagram() []byte {
	buf := make([]byte, HeaderSize+DataItemHeaderSize+30)
	binary.LittleEndian.PutUint32(buf[0:4], MarkerValue)
	binary.LittleEndian.PutUint32(buf[4:8], 0xDEADBEEF)
	copy(buf[8:16], []byte("CDP0002\x00"))
	binary.LittleEndian.PutUint32(buf[16:20], 0x12345678)

	binary.LittleEndian.PutUint16(buf[20:22], 0x0135)
	binary.LittleEndian.PutUint16(buf[22:24], 30)

	p := buf[24:54]
	binary.LittleEndian.PutUint32(p[0:4], 0xABCDEF01)
	binary.LittleEndian.PutUint64(p[4:12], 0x0123456789ABCDEF)
	binary.LittleEndian.PutUint32(p[12:16], uint32(int32(1000)))
	binary.LittleEndian.PutUint32(p[16:20], uint32(int32(2000)))
	binary.LittleEndian.PutUint32(p[20:24], uint32(int32(3000)))
	binary.LittleEndian.PutUint16(p[24:26], 95)
	p[26] = 4
	p[27] = 1
	binary.LittleEndian.PutUint16(p[28:30], 100)
	return buf
}

var decodeSink *Packet

// BenchmarkDecode_PositionV3 isolates the parser cost (no NATS, no JSON):
// header validation + one PositionV3 item decode + Items slice allocation.
// Useful for spotting regressions in the per-item allocation count.
func BenchmarkDecode_PositionV3(b *testing.B) {
	datagram := samplePositionV3Datagram()
	b.SetBytes(int64(len(datagram)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pkt, err := Decode(datagram)
		if err != nil {
			b.Fatal(err)
		}
		decodeSink = pkt
	}
}
