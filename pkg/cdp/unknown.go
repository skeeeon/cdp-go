package cdp

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// Unknown is the fallback payload returned for data items whose type ID
// has no registered decoder. The raw payload bytes are preserved so a
// downstream consumer can decode out-of-band if it knows the type.
type Unknown struct {
	TypeID uint16
	Raw    []byte
}

// MarshalJSON emits {"type": "0xNNNN", "raw": "<hex>"}.
func (u *Unknown) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type string `json:"type"`
		Raw  string `json:"raw"`
	}{
		Type: fmt.Sprintf("0x%04X", u.TypeID),
		Raw:  hex.EncodeToString(u.Raw),
	})
}

func unknownName(typeID uint16) string {
	return fmt.Sprintf("Unknown_0x%04X", typeID)
}

func unknownSubject(typeID uint16) string {
	return fmt.Sprintf("unknown_%04x", typeID)
}
