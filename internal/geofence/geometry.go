package geofence

import (
	"errors"
	"fmt"
	"strings"
)

// Point is a 2D position in CDP integer millimeters, matching cdp.PositionV3.X/Y.
//
// Using int32 throughout (instead of float as the Python implementation did)
// eliminates the float-equality bugs in zone.py and matches the wire format
// exactly.
type Point struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

// RGB is a zone's diagnostic color. Carried in geofence events so a future
// LED-bridge can drive tag firmware without looking up the zone again.
type RGB struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

// Zone is one configured polygonal region.
//
// Vertices may be listed clockwise or counter-clockwise; the algorithm is
// orientation-agnostic. The polygon is implicitly closed (last vertex
// connects to first). bbox is precomputed for the cheap-reject prefilter.
//
// Boundary behavior: points exactly on a polygon edge or vertex are
// reported as "inside" or "outside" deterministically for a given input,
// but the choice is implementation-defined and not specified to match any
// particular convention. At UWB millimeter resolution, boundary-precise
// calls aren't physically meaningful.
type Zone struct {
	Name     string
	Slug     string
	Vertices []Point
	Color    RGB
	bbox     bbox
}

type bbox struct {
	minX, minY, maxX, maxY int32
}

// NewZone validates a zone definition and precomputes its bbox.
func NewZone(name string, vertices []Point, color RGB) (*Zone, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("zone name must be non-empty")
	}
	slug := slugifyZoneName(name)
	if slug == "" {
		return nil, fmt.Errorf("zone name %q slugifies to empty", name)
	}
	if len(vertices) < 3 {
		return nil, fmt.Errorf("zone %q: need at least 3 vertices, got %d", name, len(vertices))
	}
	verts := make([]Point, len(vertices))
	copy(verts, vertices)
	return &Zone{
		Name:     name,
		Slug:     slug,
		Vertices: verts,
		Color:    color,
		bbox:     boundingBox(verts),
	}, nil
}

// Contains reports whether p lies inside the zone, with a cheap bbox
// prefilter ahead of the full ray-cast.
func (z *Zone) Contains(p Point) bool {
	if p.X < z.bbox.minX || p.X > z.bbox.maxX ||
		p.Y < z.bbox.minY || p.Y > z.bbox.maxY {
		return false
	}
	return PointInPolygon(p, z.Vertices)
}

// PointInPolygon: W. R. Franklin's ray-cast.
//
// Strict-inequality on edge endpoints — `(yi > py) != (yj > py)` — means a
// point coincident with a vertex shared by two edges is counted by exactly
// one of those edges, eliminating the float-equality "y in y_list" patch
// the Python implementation needed (zone.py:58–61).
//
// The ray-hits-edge-to-the-right test is cross-multiplied to stay in
// integer arithmetic — no division, no epsilon. int64 intermediates
// accommodate the product of two int32 differences. (For the
// astronomically large coordinates that could overflow int64,
// pre-validate vertices and positions to fit in some smaller bound;
// realistic UWB millimeters are well within safe range.)
func PointInPolygon(p Point, vertices []Point) bool {
	n := len(vertices)
	if n < 3 {
		return false
	}
	px, py := int64(p.X), int64(p.Y)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := int64(vertices[i].X), int64(vertices[i].Y)
		xj, yj := int64(vertices[j].X), int64(vertices[j].Y)
		if (yi > py) != (yj > py) &&
			((px-xi)*(yj-yi) < (xj-xi)*(py-yi)) == (yj > yi) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func boundingBox(vertices []Point) bbox {
	bb := bbox{
		minX: vertices[0].X, maxX: vertices[0].X,
		minY: vertices[0].Y, maxY: vertices[0].Y,
	}
	for _, v := range vertices[1:] {
		if v.X < bb.minX {
			bb.minX = v.X
		}
		if v.X > bb.maxX {
			bb.maxX = v.X
		}
		if v.Y < bb.minY {
			bb.minY = v.Y
		}
		if v.Y > bb.maxY {
			bb.maxY = v.Y
		}
	}
	return bb
}

// slugifyZoneName lowercases a zone name and replaces NATS-illegal
// subject-token characters (., >, *, whitespace) with underscores.
// Leading/trailing underscores are trimmed.
func slugifyZoneName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range strings.ToLower(name) {
		switch {
		case r == '.', r == '>', r == '*':
			b.WriteByte('_')
		case r == ' ', r == '\t', r == '\n', r == '\r':
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "_")
}
