package main

import "testing"

func TestParseSGRMouseWheel(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		code int
		dir  string
		x, y int
	}{
		{name: "up", data: "\x1b[<64;3;7M", code: 64, dir: "up", x: 3, y: 7},
		{name: "down release form", data: "\x1b[<65;4;8m", code: 65, dir: "down", x: 4, y: 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ev, consumed, complete := parseSGRMouse([]byte(tc.data))
			if !complete || consumed != len(tc.data) {
				t.Fatalf("parseSGRMouse complete=%v consumed=%d, want complete and %d", complete, consumed, len(tc.data))
			}
			if ev.Code != tc.code || ev.X != tc.x || ev.Y != tc.y {
				t.Fatalf("event=%+v, want code=%d x=%d y=%d", ev, tc.code, tc.x, tc.y)
			}
			if got, ok := ev.wheel(); !ok || got != tc.dir {
				t.Fatalf("wheel()=(%q,%v), want %q,true", got, ok, tc.dir)
			}
		})
	}
}

func TestParseSGRMouseIncomplete(t *testing.T) {
	for _, data := range []string{"\x1b", "\x1b[", "\x1b[<", "\x1b[<64;3;"} {
		if _, consumed, complete := parseSGRMouse([]byte(data)); complete || consumed != 0 {
			t.Fatalf("parseSGRMouse(%q) complete=%v consumed=%d, want incomplete", data, complete, consumed)
		}
	}
}
