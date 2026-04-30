package geofence

import (
	"fmt"
	"time"
)

// Config is the geofence feature's configuration. Embed it inside a
// binary's top-level Config under the `geofence` YAML key. An empty Zones
// slice disables the feature: the engine isn't constructed and no
// geofence code runs in the per-packet path.
type Config struct {
	// Hysteresis: how many consecutive PositionV3 packets a proposed
	// zone-membership change must hold before being committed. <= 1
	// commits immediately on any observed change.
	Hysteresis int `yaml:"hysteresis"`

	// Prefix is the NATS subject prefix for geofence events. Default "geofence".
	Prefix string `yaml:"prefix"`

	// TagTTL: drop a tag's per-tag state if no position update has
	// arrived for this long. Zero disables the sweep. Wall-clock
	// duration — NOT CDP NetworkTime — because NetworkTime is a UWB
	// sync clock that resets on tag reboot.
	TagTTL time.Duration `yaml:"tag_ttl"`

	// Zones is the static list of polygons. Coordinates are int32
	// millimeters, matching cdp.PositionV3.X/Y exactly.
	Zones []ZoneConfig `yaml:"zones"`
}

// ZoneConfig is one polygon entry from YAML.
//
// Coordinates are int32 millimeters — a deliberate divergence from the
// Python implementation's float coords. CDP positions are integer mm on
// the wire, so keeping the same units throughout eliminates any
// coordinate-scaling foot-gun.
type ZoneConfig struct {
	Name     string     `yaml:"name"`
	Vertices [][2]int32 `yaml:"vertices"`
	RGB      [3]uint8   `yaml:"rgb"`
}

// Enabled reports whether the feature should be activated.
func (c Config) Enabled() bool { return len(c.Zones) > 0 }

// Build validates every zone in the config and returns the constructed
// *Zone list. Rejects: non-positive hysteresis (negative; 0 is OK and
// means "commit immediately"), zones with < 3 vertices, zones whose name
// slugifies to empty, and duplicate slugs across zones.
func (c Config) Build() ([]*Zone, error) {
	if c.Hysteresis < 0 {
		return nil, fmt.Errorf("geofence: hysteresis must be >= 0, got %d", c.Hysteresis)
	}
	if c.TagTTL < 0 {
		return nil, fmt.Errorf("geofence: tag_ttl must be >= 0, got %s", c.TagTTL)
	}
	zones := make([]*Zone, 0, len(c.Zones))
	seenSlug := make(map[string]string, len(c.Zones))
	for i, zc := range c.Zones {
		verts := make([]Point, len(zc.Vertices))
		for j, v := range zc.Vertices {
			verts[j] = Point{X: v[0], Y: v[1]}
		}
		z, err := NewZone(zc.Name, verts, RGB{R: zc.RGB[0], G: zc.RGB[1], B: zc.RGB[2]})
		if err != nil {
			return nil, fmt.Errorf("geofence: zones[%d]: %w", i, err)
		}
		if prev, dup := seenSlug[z.Slug]; dup {
			return nil, fmt.Errorf("geofence: zones[%d]: name %q has duplicate slug %q (also produced by %q)",
				i, z.Name, z.Slug, prev)
		}
		seenSlug[z.Slug] = z.Name
		zones = append(zones, z)
	}
	return zones, nil
}
