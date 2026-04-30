package geofence

import (
	"strings"
	"testing"
)

func TestConfigEnabled(t *testing.T) {
	if (Config{}).Enabled() {
		t.Error("empty config should be disabled")
	}
	c := Config{Zones: []ZoneConfig{{Name: "x", Vertices: [][2]int32{{0, 0}, {1, 0}, {1, 1}}}}}
	if !c.Enabled() {
		t.Error("config with zones should be enabled")
	}
}

func TestConfigBuildSuccess(t *testing.T) {
	c := Config{
		Zones: []ZoneConfig{
			{
				Name:     "Paper",
				Vertices: [][2]int32{{0, 0}, {10, 0}, {10, 10}, {0, 10}},
				RGB:      [3]uint8{255, 255, 0},
			},
			{
				Name:     "Loading Dock",
				Vertices: [][2]int32{{20, 0}, {30, 0}, {25, 10}},
				RGB:      [3]uint8{0, 0, 255},
			},
		},
	}
	zones, err := c.Build()
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got %d", len(zones))
	}
	if zones[0].Name != "Paper" || zones[0].Slug != "paper" {
		t.Errorf("zone 0: got name=%q slug=%q", zones[0].Name, zones[0].Slug)
	}
	if zones[1].Slug != "loading_dock" {
		t.Errorf("zone 1 slug: got %q, want loading_dock", zones[1].Slug)
	}
}

func TestConfigBuildRejects(t *testing.T) {
	tests := []struct {
		name    string
		zones   []ZoneConfig
		wantErr string
	}{
		{
			name: "too few vertices",
			zones: []ZoneConfig{
				{Name: "x", Vertices: [][2]int32{{0, 0}, {1, 1}}},
			},
			wantErr: "vertices",
		},
		{
			name: "empty name",
			zones: []ZoneConfig{
				{Name: "", Vertices: [][2]int32{{0, 0}, {1, 0}, {1, 1}}},
			},
			wantErr: "name",
		},
		{
			name: "name slugifies to empty",
			zones: []ZoneConfig{
				{Name: "...", Vertices: [][2]int32{{0, 0}, {1, 0}, {1, 1}}},
			},
			wantErr: "slugifies",
		},
		{
			name: "duplicate slug",
			zones: []ZoneConfig{
				{Name: "Paper", Vertices: [][2]int32{{0, 0}, {1, 0}, {1, 1}}},
				{Name: "paper", Vertices: [][2]int32{{0, 0}, {1, 0}, {1, 1}}},
			},
			wantErr: "duplicate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Config{Zones: tt.zones}
			_, err := c.Build()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestConfigRejectsNegativeHysteresis(t *testing.T) {
	c := Config{
		Hysteresis: -1,
		Zones:      []ZoneConfig{{Name: "x", Vertices: [][2]int32{{0, 0}, {1, 0}, {1, 1}}}},
	}
	if _, err := c.Build(); err == nil {
		t.Error("expected negative hysteresis to be rejected")
	}
}
