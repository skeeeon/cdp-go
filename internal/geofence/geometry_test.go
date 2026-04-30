package geofence

import "testing"

// L-shape: an L-bent hexagon used for several edge cases. Two pairs of
// vertices share a Y coordinate, exercising the "ray grazes shared y"
// case that the Python implementation patched up with float equality.
//
//	(0,10)─────(5,10)
//	  │          │
//	  │          │
//	  │       (5,5)──(10,5)
//	  │                │
//	  │                │
//	(0,0)──────────(10,0)
var lShape = []Point{
	{0, 0}, {10, 0}, {10, 5}, {5, 5}, {5, 10}, {0, 10},
}

var unitSquare = []Point{
	{0, 0}, {10, 0}, {10, 10}, {0, 10},
}

func TestPointInPolygon(t *testing.T) {
	tests := []struct {
		name     string
		vertices []Point
		p        Point
		want     bool
	}{
		// Square interior / exterior.
		{"square interior", unitSquare, Point{5, 5}, true},
		{"square exterior far", unitSquare, Point{50, 50}, false},
		{"square exterior left", unitSquare, Point{-5, 5}, false},
		{"square exterior above", unitSquare, Point{5, 50}, false},

		// Square reversed winding (CCW vs CW). Same answer expected.
		{"square cw winding interior", []Point{{0, 0}, {0, 10}, {10, 10}, {10, 0}}, Point{5, 5}, true},
		{"square cw winding exterior", []Point{{0, 0}, {0, 10}, {10, 10}, {10, 0}}, Point{50, 50}, false},

		// L-shape: the concavity must be reported outside.
		{"L concavity exterior", lShape, Point{8, 8}, false},
		{"L lower arm interior", lShape, Point{3, 3}, true},
		{"L upper arm interior", lShape, Point{3, 7}, true},
		{"L right leg interior", lShape, Point{7, 3}, true},

		// Y-coincident with shared vertices (the Python y_list patch case).
		// Ray at y=5 passes through two shared y-values; algorithm must
		// produce the right count without the float-equality patch.
		{"L y=5 far right exterior", lShape, Point{20, 5}, false},
		{"L y=5 far left exterior", lShape, Point{-5, 5}, false},
		{"L y=5 inside lower-left", lShape, Point{3, 5}, true},
		// y=0 is the bottom edge's y; both y=10 vertices share that level.
		{"L y=10 far right exterior", lShape, Point{20, 10}, false},

		// Boundary: corner vertex of the square. Behavior is stable but
		// not specified to match any particular convention; we pin it.
		{"square corner (0,0) included", unitSquare, Point{0, 0}, true},

		// Boundary: point on horizontal bottom edge. Pinned behavior.
		{"L on bottom edge included", lShape, Point{3, 0}, true},

		// Triangle.
		{"triangle interior", []Point{{0, 0}, {10, 0}, {5, 10}}, Point{5, 3}, true},
		{"triangle exterior above apex", []Point{{0, 0}, {10, 0}, {5, 10}}, Point{5, 20}, false},

		// Pentagon (regular-ish, integer coords).
		{"pentagon interior", []Point{{0, 0}, {10, 0}, {12, 7}, {5, 12}, {-2, 7}}, Point{5, 5}, true},
		{"pentagon exterior", []Point{{0, 0}, {10, 0}, {12, 7}, {5, 12}, {-2, 7}}, Point{20, 5}, false},

		// Polygon with three collinear vertices on one edge.
		{"collinear vertices interior", []Point{{0, 0}, {5, 0}, {10, 0}, {10, 10}, {0, 10}}, Point{5, 5}, true},
		{"collinear vertices exterior", []Point{{0, 0}, {5, 0}, {10, 0}, {10, 10}, {0, 10}}, Point{15, 5}, false},

		// Degenerate.
		{"empty vertices", []Point{}, Point{0, 0}, false},
		{"two vertices", []Point{{0, 0}, {10, 10}}, Point{5, 5}, false},

		// Realistic large coordinates — exercises int64 widening of int32
		// products. 1e8 mm = 100 km, well past any indoor scenario.
		{"large coords interior", []Point{{-100_000_000, -100_000_000}, {100_000_000, -100_000_000}, {100_000_000, 100_000_000}, {-100_000_000, 100_000_000}}, Point{0, 0}, true},
		{"large coords exterior", []Point{{-100_000_000, -100_000_000}, {100_000_000, -100_000_000}, {100_000_000, 100_000_000}, {-100_000_000, 100_000_000}}, Point{200_000_000, 0}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PointInPolygon(tt.p, tt.vertices); got != tt.want {
				t.Errorf("PointInPolygon(%v, %v) = %v, want %v", tt.p, tt.vertices, got, tt.want)
			}
		})
	}
}

func TestZoneContainsBboxPrefilter(t *testing.T) {
	z, err := NewZone("box", unitSquare, RGB{1, 2, 3})
	if err != nil {
		t.Fatalf("NewZone: %v", err)
	}
	if z.Contains(Point{50, 50}) {
		t.Error("expected (50,50) outside bbox to be rejected")
	}
	if !z.Contains(Point{5, 5}) {
		t.Error("expected (5,5) inside zone to be accepted")
	}
}

func TestNewZoneRejectsBadInput(t *testing.T) {
	tests := []struct {
		name     string
		zoneName string
		verts    []Point
	}{
		{"too few vertices", "tiny", []Point{{0, 0}, {1, 1}}},
		{"empty name", "", unitSquare},
		{"name slugifies empty", ".", unitSquare},
		{"whitespace-only name", "   ", unitSquare},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewZone(tt.zoneName, tt.verts, RGB{}); err == nil {
				t.Errorf("expected error for %s", tt.name)
			}
		})
	}
}

func TestSlugifyZoneName(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Paper", "paper"},
		{"Loading Dock", "loading_dock"},
		{"Zone.A", "zone_a"},
		{"a>b*c", "a_b_c"},
		{"  spaced  ", "spaced"},
		{"already_ok", "already_ok"},
	}
	for _, tt := range tests {
		if got := slugifyZoneName(tt.in); got != tt.want {
			t.Errorf("slugifyZoneName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
