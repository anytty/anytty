package frameaudit

import (
	"fmt"
	"testing"
)

func TestParseDumpParsesLengthDelimitedEntries(t *testing.T) {
	payloadA := []byte("\x1b[Habc")
	payloadB := []byte("\x1b[Habc")
	dump := buildDumpForTest(
		DumpEntry{Kind: "direct_frame", Timestamp: "2026-04-15T00:00:00Z", Payload: payloadA},
		DumpEntry{Kind: "direct_frame", Timestamp: "2026-04-15T00:00:01Z", Payload: payloadB},
	)

	entries, err := ParseDump(dump)
	if err != nil {
		t.Fatalf("parse dump: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if got := string(entries[0].Payload); got != string(payloadA) {
		t.Fatalf("unexpected first payload %q", got)
	}
	if got := string(entries[1].Payload); got != string(payloadB) {
		t.Fatalf("unexpected second payload %q", got)
	}
}

func TestAuditEntriesFlagsRedundantFrameAsNoop(t *testing.T) {
	dump := buildDumpForTest(
		DumpEntry{Kind: "direct_frame", Timestamp: "2026-04-15T00:00:00Z", Payload: []byte("\x1b[Habc")},
		DumpEntry{Kind: "direct_frame", Timestamp: "2026-04-15T00:00:01Z", Payload: []byte("\x1b[Habc")},
	)

	entries, err := ParseDump(dump)
	if err != nil {
		t.Fatalf("parse dump: %v", err)
	}
	report, err := AuditEntries(entries, 10, 4)
	if err != nil {
		t.Fatalf("audit entries: %v", err)
	}
	if len(report.Entries) != 2 {
		t.Fatalf("expected 2 entry stats, got %d", len(report.Entries))
	}
	if report.Entries[0].Noop {
		t.Fatal("expected first entry to change screen")
	}
	if !report.Entries[1].Noop {
		t.Fatalf("expected second entry to be noop, got %#v", report.Entries[1])
	}
	if report.Summary.Noops != 1 {
		t.Fatalf("expected 1 noop in summary, got %#v", report.Summary)
	}
}

func buildDumpForTest(entries ...DumpEntry) []byte {
	var dump []byte
	for _, entry := range entries {
		header := fmt.Sprintf("--- %s %s len=%d ---\n", entry.Kind, entry.Timestamp, len(entry.Payload))
		dump = append(dump, []byte(header)...)
		dump = append(dump, entry.Payload...)
		dump = append(dump, '\n')
	}
	return dump
}
