package cdpbridge

import (
	stdjson "encoding/json"
	"testing"
	"time"

	"github.com/velociti/cdp-go/internal/geofence"
	"github.com/velociti/cdp-go/pkg/cdp"
)

func TestNatsGeofenceSink_Emit(t *testing.T) {
	nc := runTestNATS(t)

	sub, err := nc.SubscribeSync("geofence.>")
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	sink := &natsGeofenceSink{nc: nc, prefix: "geofence"}
	ev := geofence.Event{
		Type:        geofence.EventEnter,
		Tag:         cdp.Serial(0x12345678),
		Zone:        "Loading Bay 3",
		ZoneSlug:    "loading_bay_3",
		InZones:     []string{"Loading Bay 3"},
		NetworkTime: 0x0123456789ABCDEF,
		Position:    geofence.Point{X: 1000, Y: 2000},
		Color:       geofence.RGB{R: 255, G: 128, B: 64},
	}
	if err := sink.Emit(ev); err != nil {
		t.Fatalf("Emit: %v", err)
	}

	msg, err := sub.NextMsg(time.Second)
	if err != nil {
		t.Fatalf("NextMsg: %v", err)
	}
	if want := "geofence.enter.12345678.loading_bay_3"; msg.Subject != want {
		t.Errorf("subject: got %q, want %q", msg.Subject, want)
	}

	var got struct {
		Type        string   `json:"type"`
		Tag         string   `json:"tag"`
		Zone        string   `json:"zone"`
		InZones     []string `json:"in_zones"`
		NetworkTime uint64   `json:"network_time"`
		Position    struct {
			X int32 `json:"x"`
			Y int32 `json:"y"`
		} `json:"position"`
	}
	if err := stdjson.Unmarshal(msg.Data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Type != "enter" {
		t.Errorf("type: got %q, want enter", got.Type)
	}
	if got.Zone != "Loading Bay 3" {
		t.Errorf("zone: got %q, want Loading Bay 3", got.Zone)
	}
	if got.Tag != cdp.Serial(0x12345678).String() {
		t.Errorf("tag: got %q, want %q", got.Tag, cdp.Serial(0x12345678).String())
	}
	if len(got.InZones) != 1 || got.InZones[0] != "Loading Bay 3" {
		t.Errorf("in_zones: got %v", got.InZones)
	}
	if got.Position.X != 1000 || got.Position.Y != 2000 {
		t.Errorf("position: got %+v", got.Position)
	}
	if got.NetworkTime != 0x0123456789ABCDEF {
		t.Errorf("network_time: got %x", got.NetworkTime)
	}
}
