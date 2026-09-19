package conformance_test

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/anytty/anytty/clients/tui/conformance"
	wire "github.com/anytty/anytty/proto/ui"
)

func loadFixtures(t *testing.T) []conformance.Fixture {
	t.Helper()
	fixtures, err := conformance.LoadFixtures("fixtures.jsonl")
	if err != nil {
		t.Fatalf("load fixtures: %v", err)
	}
	if len(fixtures) < 10 {
		t.Fatalf("only %d fixtures; the suite must keep at least 10 cases", len(fixtures))
	}
	seen := map[string]bool{}
	for _, fixture := range fixtures {
		if err := fixture.Validate(); err != nil {
			t.Errorf("invalid fixture: %v", err)
		}
		if seen[fixture.Name] {
			t.Errorf("duplicate fixture name %q", fixture.Name)
		}
		seen[fixture.Name] = true
	}
	return fixtures
}

// TestOfficialGoSDKPasses is the M1 gate: the reference Go SDK must pass
// every fixture, and the fixtures must actually exercise both directions.
func TestOfficialGoSDKPasses(t *testing.T) {
	fixtures := loadFixtures(t)
	results := conformance.Run(fixtures, conformance.NewRefFactory(), 5*time.Second)
	failed := 0
	for _, result := range results {
		if !result.Passed {
			failed++
			t.Errorf("fixture %s failed at step %d: %v", result.Name, result.Steps, result.Err)
		}
	}
	if failed > 0 {
		t.Fatalf("%d/%d fixtures failed", failed, len(results))
	}
	t.Logf("official Go SDK: %d/%d fixtures passed", len(results), len(results))
}

// TestFixturesCoverEveryEvent pins the "typical sequences" requirement: the
// suite must cover every EVENT kind, both program frame types, epoch reset,
// view_rejected and response correlation.
func TestFixturesCoverEveryEvent(t *testing.T) {
	fixtures := loadFixtures(t)
	kinds := map[string]bool{}
	hasView, hasResult, hasHello, hasResponse, hasRaw := false, false, false, false, false
	for _, fixture := range fixtures {
		for _, step := range fixture.Steps {
			if step.Send != nil {
				if step.Send.Hello != nil {
					hasHello = true
				}
				if step.Send.Event != nil {
					kinds[step.Send.Event.Kind] = true
				}
				if step.Send.Response != nil {
					hasResponse = true
				}
				if step.Send.Raw != "" {
					hasRaw = true
				}
			}
			if step.Expect != nil {
				for _, frame := range step.Expect.Frames {
					if frame.View != nil {
						hasView = true
					}
					if frame.Result != nil {
						hasResult = true
					}
				}
			}
		}
	}
	for _, kind := range []string{"key", "paste", "mouse", "wheel", "resize", "sources", "notice", "component", "view_rejected"} {
		if !kinds[kind] {
			t.Errorf("no fixture sends event kind %q", kind)
		}
	}
	for name, present := range map[string]bool{
		"hello": hasHello, "response": hasResponse, "raw": hasRaw,
		"view": hasView, "result": hasResult,
	} {
		if !present {
			t.Errorf("no fixture exercises %s", name)
		}
	}
}

// TestBrokenFixtureFails makes sure validation catches malformed fixtures.
func TestBrokenFixtureFails(t *testing.T) {
	cases := []conformance.Fixture{
		{Name: "empty"},
		{Name: "two actions", Steps: []conformance.Step{{
			Send:   &conformance.SendStep{Hello: &conformance.HelloSpec{}},
			Expect: &conformance.ExpectStep{},
		}}},
		{Name: "bad raw", Steps: []conformance.Step{{Send: &conformance.SendStep{Raw: "zz"}}}},
	}
	for _, fixture := range cases {
		if err := fixture.Validate(); err == nil {
			t.Errorf("fixture %q: validation passed, want error", fixture.Name)
		}
	}
}

// TestFrameSamples decodes every documented binary sample and compares the
// semantic JSON, so frames.jsonl stays a usable oracle for other languages.
func TestFrameSamples(t *testing.T) {
	file, err := os.Open("frames.jsonl")
	if err != nil {
		t.Fatalf("open frames.jsonl: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "//") {
			continue
		}
		var sample struct {
			Name   string          `json:"name"`
			Type   string          `json:"type"`
			Hex    string          `json:"hex"`
			File   string          `json:"file"`
			Decode json.RawMessage `json:"decode"`
		}
		if err := json.Unmarshal([]byte(text), &sample); err != nil {
			t.Fatalf("frames.jsonl:%d: %v", line, err)
		}
		frame, err := hex.DecodeString(sample.Hex)
		if err != nil {
			t.Fatalf("%s: hex: %v", sample.Name, err)
		}
		if sample.File != "" {
			onDisk, err := os.ReadFile(sample.File)
			if err != nil {
				t.Fatalf("%s: read %s: %v", sample.Name, sample.File, err)
			}
			if !bytes.Equal(onDisk, frame) {
				t.Errorf("%s: %s does not match the documented hex", sample.Name, sample.File)
			}
		}
		frameType, payload, err := wire.DecodeFrame(frame, wire.RoleProgram, 0)
		if err != nil {
			t.Fatalf("%s: decode frame: %v", sample.Name, err)
		}
		if frameType.String() != sample.Type {
			t.Errorf("%s: frame type %s, want %s", sample.Name, frameType, sample.Type)
		}
		message, err := wire.UnmarshalPayload(frameType, payload)
		if err != nil {
			t.Fatalf("%s: decode payload: %v", sample.Name, err)
		}
		gotJSON, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(message)
		if err != nil {
			t.Fatalf("%s: marshal: %v", sample.Name, err)
		}
		if !jsonEqual(t, gotJSON, sample.Decode) {
			t.Errorf("%s: decoded JSON mismatch\n  got  %s\n  want %s", sample.Name, gotJSON, sample.Decode)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
}

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var left, right any
	if err := json.Unmarshal(a, &left); err != nil {
		t.Fatalf("json a: %v", err)
	}
	if err := json.Unmarshal(b, &right); err != nil {
		t.Fatalf("json b: %v", err)
	}
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}
