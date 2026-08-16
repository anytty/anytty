package render

import "testing"

func TestBoxGlyphConnectionMapCoversSplitJunctions(t *testing.T) {
	cases := []struct {
		name        string
		connections uint8
		want        string
	}{
		{name: "left tee", connections: boxConnUp | boxConnDown | boxConnRight, want: "├"},
		{name: "right tee", connections: boxConnUp | boxConnDown | boxConnLeft, want: "┤"},
		{name: "top tee", connections: boxConnDown | boxConnLeft | boxConnRight, want: "┬"},
		{name: "bottom tee", connections: boxConnUp | boxConnLeft | boxConnRight, want: "┴"},
		{name: "cross", connections: boxConnUp | boxConnDown | boxConnLeft | boxConnRight, want: "┼"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := boxGlyphForConnections(tt.connections)
			if !ok || got != tt.want {
				t.Fatalf("unexpected glyph got=%q ok=%v want=%q", got, ok, tt.want)
			}
			if connections, ok := boxConnectionsForGlyph(got); !ok || connections != tt.connections {
				t.Fatalf("unexpected reverse mapping glyph=%q connections=%b ok=%v", got, connections, ok)
			}
		})
	}
}
