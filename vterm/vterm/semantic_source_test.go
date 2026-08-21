package vterm

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	charmvt "github.com/anytty/anytty/vterm/internal/vt"
	uv "github.com/charmbracelet/ultraviolet"
)

func TestSemanticSourceApplyPTYWriteEmitsOrderedTransaction(t *testing.T) {
	source := NewSemanticSource(12, 2, 100, nil)
	tx, err := source.ApplyPTYWrite([]byte("one\r\ntwo\r\nthree"))
	if err != nil {
		t.Fatalf("apply pty write: %v", err)
	}
	if tx.Seq != 1 || tx.Size != (TerminalSemanticSize{Cols: 12, Rows: 2}) {
		t.Fatalf("unexpected transaction header: %#v", tx)
	}
	if len(tx.Ops) == 0 {
		t.Fatalf("transaction must expose ordered semantic ops: %#v", tx)
	}
	if len(tx.PrimaryScrollOut) != 0 {
		t.Fatalf("ordinary primary write should keep scroll-out payload in ordered ops only: %#v", tx.PrimaryScrollOut)
	}
	if tx.PrimaryFrame != nil || tx.AltFrame != nil {
		t.Fatalf("ordinary primary write should not carry frame side proof: %#v", tx)
	}
	if tx.SourceDamage.SizeCols != 12 || tx.SourceDamage.SizeRows != 2 {
		t.Fatalf("transaction should keep source damage summary for diagnostics: %#v", tx.SourceDamage)
	}
	if len(tx.SourceDamage.Ops) != 0 || len(tx.SourceDamage.ScrollbackAppend) != 0 {
		t.Fatalf("ordinary transaction must not duplicate source damage payload: %#v", tx.SourceDamage)
	}
}

func TestR390SemanticSourceTransfersDamageOpsWithoutCrossWriteAlias(t *testing.T) {
	source := NewSemanticSource(24, 4, 100, nil)
	first, err := source.ApplyPTYWrite([]byte("first line\r\n"))
	if err != nil {
		t.Fatalf("apply first write: %v", err)
	}
	if len(first.Ops) == 0 || (len(first.Ops[0].Cells) == 0 && len(first.Ops[0].Runs) == 0) {
		t.Fatalf("expected first write to expose ordered ops, got %#v", first)
	}
	if len(first.Ops[0].Runs) > 0 {
		first.Ops[0].Runs[0].Text = "X"
	} else {
		first.Ops[0].Cells[0].Content = "X"
	}

	second, err := source.ApplyPTYWrite([]byte("second line\r\n"))
	if err != nil {
		t.Fatalf("apply second write: %v", err)
	}
	if len(second.Ops) == 0 || (len(second.Ops[0].Cells) == 0 && len(second.Ops[0].Runs) == 0) {
		t.Fatalf("expected second write to expose ordered ops, got %#v", second)
	}
	if got := semanticOpTextForSourceTest(second.Ops[0]); got == "X" {
		t.Fatalf("mutating one transaction must not alias future semantic damage payload, got %#v", second.Ops[0])
	}
}

func TestR450SemanticSourceFastASCIISGRKeepsEvictionsAndRuns(t *testing.T) {
	source := NewSemanticSource(16, 3, 0, nil)
	var raw strings.Builder
	for i := 1; i <= 8; i++ {
		raw.WriteString("row")
		raw.WriteString(strconv.Itoa(i))
		raw.WriteString(" \x1b[31mred\x1b[0m\r\n")
	}

	tx, err := source.ApplyPTYWrite([]byte(raw.String()))
	if err != nil {
		t.Fatalf("apply fast ascii/sgr write: %v", err)
	}
	if len(tx.PrimaryScrollOut) != 0 {
		t.Fatalf("ordinary fast path write must keep PrimaryScrollOut gated: %#v", tx.PrimaryScrollOut)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	if len(got) < 6 || got[0] != "row1 red" {
		t.Fatalf("fast ascii/sgr path must keep eviction proof, got %v", got)
	}
	seenStyledRun := false
	for _, op := range tx.Ops {
		if op.Code != ScreenOpWriteSpan || len(op.Runs) == 0 {
			continue
		}
		for _, run := range op.Runs {
			if run.Text == "red" && run.Style.FG != "" {
				seenStyledRun = true
			}
		}
	}
	if !seenStyledRun {
		t.Fatalf("fast ascii/sgr path must keep styled text runs, ops=%#v", tx.Ops)
	}
}

func TestR450LineHistorySemanticSourceDropsOrdinaryTextOps(t *testing.T) {
	source := NewLineHistorySemanticSource(16, 3, 0, nil)
	var raw strings.Builder
	for i := 1; i <= 8; i++ {
		raw.WriteString("row")
		raw.WriteString(strconv.Itoa(i))
		raw.WriteString(" \x1b[31mred\x1b[0m\r\n")
	}

	tx, err := source.ApplyPTYWrite([]byte(raw.String()))
	if err != nil {
		t.Fatalf("apply linehist fast ascii/sgr write: %v", err)
	}
	if len(tx.Ops) != 0 {
		t.Fatalf("linehist production source must not retain ordinary ordered text ops, got %#v", tx.Ops)
	}
	got := evictedTextsForTest(tx.EvictedRows)
	if len(got) < 6 || got[0] != "row1 red" {
		t.Fatalf("linehist production source must keep eviction proof, got %v", got)
	}

	clearTx, err := source.ApplyPTYWrite([]byte("\x1b[3Jafter"))
	if err != nil {
		t.Fatalf("apply ED3: %v", err)
	}
	if !clearTx.ClearScrollback {
		t.Fatalf("linehist production source must keep ED3 clear-scrollback boundary: %#v", clearTx)
	}
}

func TestLineHistorySemanticSourceSingleWriteMatchesSplitLiveAndHistory(t *testing.T) {
	combined := NewLineHistorySemanticSource(12, 3, 0, nil)
	referenceLive := New(12, 3, 0, nil)
	referenceHistory := NewLineHistorySemanticSource(12, 3, 0, nil)
	writes := [][]byte{
		[]byte("one\r\ntwo\r\nthree\r\nfour"),
		[]byte("\x1b[?2026h\r\nfive\x1b[?2026l"),
		[]byte("\x1b[?1049h\x1b[2Jalternate\x1b[?1049l"),
		[]byte("\x1b[3Jafter"),
	}

	for index, raw := range writes {
		gotTx, gotDamage, err := combined.ApplyPTYWriteWithLiveDamage(raw)
		if err != nil {
			t.Fatalf("combined write %d: %v", index, err)
		}
		_, err, wantDamage := referenceLive.WriteForLatestFrame(raw)
		if err != nil {
			t.Fatalf("reference live write %d: %v", index, err)
		}
		wantTx, err := referenceHistory.ApplyPTYWrite(raw)
		if err != nil {
			t.Fatalf("reference history write %d: %v", index, err)
		}

		if gotDamage.IncrementalRowsReliable != wantDamage.IncrementalRowsReliable ||
			!reflect.DeepEqual(gotDamage.DirectDamageTouchedRows, wantDamage.DirectDamageTouchedRows) ||
			!reflect.DeepEqual(gotDamage.RowCopies, wantDamage.RowCopies) {
			t.Fatalf("combined live damage %d differs: got=%#v want=%#v", index, gotDamage, wantDamage)
		}
		assertLineHistoryTransactionEquivalent(t, index, gotTx, wantTx)
		if got, want := combined.VTerm().ScreenContent(), referenceLive.ScreenContent(); !reflect.DeepEqual(got, want) {
			t.Fatalf("combined screen %d differs: got=%#v want=%#v", index, got, want)
		}
		if got, want := combined.VTerm().CursorState(), referenceLive.CursorState(); got != want {
			t.Fatalf("combined cursor %d differs: got=%#v want=%#v", index, got, want)
		}
		if got, want := combined.VTerm().Modes(), referenceLive.Modes(); got != want {
			t.Fatalf("combined modes %d differs: got=%#v want=%#v", index, got, want)
		}
	}
}

func TestLineHistorySemanticSourceCombinedPathWritesEmulatorOnce(t *testing.T) {
	previous := safeEmulatorWriteForLineHistoryDamage
	writes := 0
	safeEmulatorWriteForLineHistoryDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
		writes++
		return previous(emu, data)
	}
	t.Cleanup(func() { safeEmulatorWriteForLineHistoryDamage = previous })

	source := NewLineHistorySemanticSource(12, 3, 0, nil)
	if _, _, err := source.ApplyPTYWriteWithLiveDamage([]byte("one\r\ntwo\r\nthree")); err != nil {
		t.Fatalf("combined write: %v", err)
	}
	if writes != 1 {
		t.Fatalf("combined path wrote emulator %d times, want exactly once", writes)
	}
}

func assertLineHistoryTransactionEquivalent(t *testing.T, index int, got TerminalSemanticTransaction, want TerminalSemanticTransaction) {
	t.Helper()
	if got.Seq != want.Seq || got.Size != want.Size || got.ClearScrollback != want.ClearScrollback ||
		got.AltEntered != want.AltEntered || got.AltExited != want.AltExited ||
		got.SynchronizedBegin != want.SynchronizedBegin || got.SynchronizedActive != want.SynchronizedActive ||
		got.SynchronizedEnd != want.SynchronizedEnd || len(got.Ops) != len(want.Ops) {
		t.Fatalf("combined transaction header %d differs: got=%#v want=%#v", index, got, want)
	}
	gotRows := evictedTextsForTest(got.EvictedRows)
	wantRows := evictedTextsForTest(want.EvictedRows)
	if !reflect.DeepEqual(gotRows, wantRows) {
		t.Fatalf("combined evictions %d differ: got=%v want=%v", index, gotRows, wantRows)
	}
}

func TestSemanticSourceApplyPTYWriteEmitsModeBoundaries(t *testing.T) {
	source := NewSemanticSource(24, 4, 100, nil)
	syncTx, err := source.ApplyPTYWrite([]byte("\x1b[?2026hhello\x1b[?2026l"))
	if err != nil {
		t.Fatalf("apply sync write: %v", err)
	}
	if !syncTx.SynchronizedBegin || !syncTx.SynchronizedEnd {
		t.Fatalf("transaction must expose synchronized output boundaries: %#v", syncTx)
	}

	altTx, err := source.ApplyPTYWrite([]byte("\x1b[?1049halt\x1b[?1049l"))
	if err != nil {
		t.Fatalf("apply alt write: %v", err)
	}
	if !altTx.AltEntered || !altTx.AltExited {
		t.Fatalf("transaction must expose alt enter/exit boundaries: %#v", altTx)
	}
}

func TestSemanticSourceSynchronizedActiveSpansSplitTransactions(t *testing.T) {
	source := NewSemanticSource(12, 3, 100, nil)
	begin, err := source.ApplyPTYWrite([]byte("\x1b[?2026h"))
	if err != nil {
		t.Fatalf("apply sync begin: %v", err)
	}
	payload, err := source.ApplyPTYWrite([]byte("payload"))
	if err != nil {
		t.Fatalf("apply sync payload: %v", err)
	}
	end, err := source.ApplyPTYWrite([]byte("\x1b[?2026l"))
	if err != nil {
		t.Fatalf("apply sync end: %v", err)
	}
	if !begin.SynchronizedBegin || !begin.SynchronizedActive {
		t.Fatalf("begin transaction should expose active synchronized mode, got %#v", begin)
	}
	if payload.SynchronizedBegin || payload.SynchronizedEnd || !payload.SynchronizedActive {
		t.Fatalf("payload transaction should remain inside synchronized mode, got %#v", payload)
	}
	if !end.SynchronizedEnd || end.SynchronizedActive {
		t.Fatalf("end transaction should close synchronized mode, got %#v", end)
	}
}

func TestSemanticSourceSynchronizedScrollOutKeepsPayloadRuns(t *testing.T) {
	source := NewSemanticSource(12, 3, 100, nil)
	raw := "\x1b[?2026h"
	for i := 1; i <= 8; i++ {
		raw += "line0" + string(rune('0'+i)) + "\r\n"
	}
	raw += "\x1b[?2026l"

	tx, err := source.ApplyPTYWrite([]byte(raw))
	if err != nil {
		t.Fatalf("apply synchronized output: %v", err)
	}
	if !tx.SynchronizedBegin || !tx.SynchronizedEnd {
		t.Fatalf("expected synchronized boundaries, got %#v", tx)
	}
	if len(tx.PrimaryScrollOut) < 6 {
		t.Fatalf("expected scroll-out proof for lines beyond screen, got %#v", tx.PrimaryScrollOut)
	}
	if got := semanticScrollOutText(tx.PrimaryScrollOut[0]); got != "line01" {
		t.Fatalf("scroll-out proof must preserve text payload, got %q in %#v", got, tx.PrimaryScrollOut[0])
	}
	if tx.PrimaryFrame == nil || len(tx.PrimaryFrame.Rows) < 2 || cellsTextForSemanticTest(tx.PrimaryFrame.Rows[0]) != "line07" || cellsTextForSemanticTest(tx.PrimaryFrame.Rows[1]) != "line08" {
		t.Fatalf("expected latest frame to remain final screen, got %#v", tx.PrimaryFrame)
	}
}

func TestR335SemanticSourceExposesFullReplaceTouchedRows(t *testing.T) {
	source := NewSemanticSource(80, 24, 100, nil)
	prevWithDamage := safeEmulatorWriteWithSemanticDamage
	safeEmulatorWriteWithSemanticDamage = func(emu *charmvt.SafeEmulator, data []byte) (int, error, []charmvt.Damage, bool) {
		n, err := emu.Write(data)
		cells := make([]uv.Cell, emu.Width())
		for col := range cells {
			cells[col] = uv.Cell{Content: "x", Width: 1}
		}
		damages := make([]charmvt.Damage, emu.Height())
		for row := 0; row < emu.Height(); row++ {
			damages[row] = charmvt.SpanDamage{X: 0, Y: row, Cells: cells}
		}
		return n, err, damages, true
	}
	t.Cleanup(func() {
		safeEmulatorWriteWithSemanticDamage = prevWithDamage
	})

	tx, err := source.ApplyPTYWrite([]byte("x"))
	if err != nil {
		t.Fatalf("apply broad direct damage: %v", err)
	}
	if !tx.RequiresFullReplace || tx.FullReplaceReason != "broad_direct_cell_damage" {
		t.Fatalf("expected broad full replace transaction, got %#v", tx)
	}
	if len(tx.PrimaryFrameTouchedRows) != 24 || tx.PrimaryFrameTouchedRows[0] != 0 || tx.PrimaryFrameTouchedRows[23] != 23 {
		t.Fatalf("semantic transaction must carry primary frame touched rows, got %#v", tx.PrimaryFrameTouchedRows)
	}
}

func TestR417SemanticSourceAttachesFrameForPrimaryPseudoTUIEdit(t *testing.T) {
	source := NewSemanticSource(20, 6, 100, nil)
	if _, err := source.ApplyPTYWrite([]byte("shell prompt\r\n")); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	tx, err := source.ApplyPTYWrite([]byte("\r\x1b[Kcodex edit"))
	if err != nil {
		t.Fatalf("apply CR+EL primary edit: %v", err)
	}
	if tx.PrimaryFrame == nil {
		t.Fatalf("primary edit transaction must attach primary frame proof, got %#v", tx)
	}
	if len(tx.PrimaryFrameTouchedRows) == 0 {
		t.Fatalf("primary edit transaction must expose touched rows, got %#v", tx)
	}
	if got := cellsTextForSemanticTest(tx.PrimaryFrame.Rows[1]); got != "codex edit" {
		t.Fatalf("primary frame should contain edited current line, got %q tx=%#v", got, tx)
	}
}

func TestR417SemanticSourceFrameAttachCoversPrimaryEditOps(t *testing.T) {
	cases := []struct {
		name string
		op   DamageOp
	}{
		{name: "clear to eol", op: DamageOp{Code: ScreenOpClearToEOL, Row: 1, Col: 0}},
		{name: "clear rect", op: DamageOp{Code: ScreenOpClearRect, Rect: DamageRect{X: 0, Y: 1, Width: 10, Height: 1}}},
		{name: "scroll rect", op: DamageOp{Code: ScreenOpScrollRect, Rect: DamageRect{X: 0, Y: 1, Width: 10, Height: 3}, Dy: -1}},
		{name: "copy rect", op: DamageOp{Code: ScreenOpCopyRect, Src: DamageRect{X: 0, Y: 1, Width: 10, Height: 1}, DstX: 0, DstY: 3}},
		{name: "relative cursor", op: DamageOp{Code: ScreenOpControl, Control: "cuu", Mode: 1}},
		{name: "line edit", op: DamageOp{Code: ScreenOpControl, Control: "dch", Row: 2, Col: 4, Mode: 1}},
		{name: "line insert", op: DamageOp{Code: ScreenOpControl, Control: "il", Row: 2, Mode: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !terminalSemanticShouldAttachFrame(WriteDamage{SemanticOps: []DamageOp{tc.op}}, TerminalSemanticTransaction{}) {
				t.Fatalf("semantic source should attach frame for %s op %#v", tc.name, tc.op)
			}
		})
	}
}

func TestR418SemanticSourceTouchedRowsPreferOrderedPrimaryEditOps(t *testing.T) {
	damage := WriteDamage{DirectDamageTouchedRows: []int{0, 1, 2, 3, 4, 5}}
	ops := []TerminalSemanticOp{
		{Code: ScreenOpControl, Control: "cuu", Row: 3, Mode: 1},
		{Code: ScreenOpControl, Control: "el", Row: 3, Col: 0, Mode: 2},
		{Code: ScreenOpWriteSpan, Row: 3, Col: 0, Runs: []CellRun{{Text: "codex edit"}}},
	}

	rows := terminalSemanticPrimaryFrameTouchedRows(damage, ops, TerminalSemanticSize{Cols: 20, Rows: 6})
	if got, want := intsForSemanticSourceTest(rows), "3"; got != want {
		t.Fatalf("ordered primary edit should own only semantic touched rows, got %s want %s", got, want)
	}
}

func TestR418SemanticSourceTouchedRowsPreserveFullReplaceDirectProof(t *testing.T) {
	damage := WriteDamage{
		RequiresFullReplace:     true,
		DirectDamageTouchedRows: []int{0, 1, 2, 3, 4, 5},
	}
	ops := []TerminalSemanticOp{{Code: ScreenOpWriteSpan, Row: 3, Col: 0, Runs: []CellRun{{Text: "one row"}}}}

	rows := terminalSemanticPrimaryFrameTouchedRows(damage, ops, TerminalSemanticSize{Cols: 20, Rows: 6})
	if got, want := intsForSemanticSourceTest(rows), "0,1,2,3,4,5"; got != want {
		t.Fatalf("full replace must preserve direct touched-row proof, got %s want %s", got, want)
	}
}

func TestR418SemanticSourceTouchedRowsCoverLineAndRectRanges(t *testing.T) {
	cases := []struct {
		name string
		ops  []TerminalSemanticOp
		want string
	}{
		{
			name: "line insert range",
			ops:  []TerminalSemanticOp{{Code: ScreenOpControl, Control: "il", Row: 2, Bottom: 5, Mode: 1}},
			want: "2,3,4",
		},
		{
			name: "copy rect destination rows",
			ops:  []TerminalSemanticOp{{Code: ScreenOpCopyRect, Src: DamageRect{X: 0, Y: 1, Width: 10, Height: 2}, DstX: 0, DstY: 4}},
			want: "4,5",
		},
		{
			name: "clear screen",
			ops:  []TerminalSemanticOp{{Code: ScreenOpControl, Control: "ed", Mode: 2}},
			want: "0,1,2,3,4,5",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rows := terminalSemanticPrimaryFrameTouchedRows(WriteDamage{}, tc.ops, TerminalSemanticSize{Cols: 20, Rows: 6})
			if got := intsForSemanticSourceTest(rows); got != tc.want {
				t.Fatalf("unexpected touched rows for %s: got %s want %s", tc.name, got, tc.want)
			}
		})
	}
}

func TestR328SemanticSourceDistinguishesEraseDisplayFromClearScrollback(t *testing.T) {
	ed2 := NewSemanticSource(12, 3, 100, nil)
	if _, err := ed2.ApplyPTYWrite([]byte("old1\r\nold2")); err != nil {
		t.Fatalf("seed ED2 source: %v", err)
	}
	ed2Tx, err := ed2.ApplyPTYWrite([]byte("\x1b[2Jnew"))
	if err != nil {
		t.Fatalf("apply ED2: %v", err)
	}
	if ed2Tx.ClearScrollback {
		t.Fatalf("ED2 saves visible screen to scrollback; it must not clear authoritative history: %#v", ed2Tx)
	}
	gotED2Proof := ""
	for _, proof := range ed2Tx.PrimaryScrollOut {
		gotED2Proof += semanticScrollOutText(proof)
	}
	if gotED2Proof != "old1old2" {
		t.Fatalf("ED2 should expose clear-time scrollback proof for every visible row, got %q in %#v", gotED2Proof, ed2Tx.PrimaryScrollOut)
	}
	if len(ed2Tx.PrimaryScrollOut) != 2 || !ed2Tx.PrimaryScrollOut[0].RowSet || ed2Tx.PrimaryScrollOut[0].Row != 0 || !ed2Tx.PrimaryScrollOut[1].RowSet || ed2Tx.PrimaryScrollOut[1].Row != 1 {
		t.Fatalf("ED2 clear-time proof must preserve original viewport rows, got %#v", ed2Tx.PrimaryScrollOut)
	}

	ed3 := NewSemanticSource(12, 3, 100, nil)
	if _, err := ed3.ApplyPTYWrite([]byte("old1\r\nold2")); err != nil {
		t.Fatalf("seed ED3 source: %v", err)
	}
	ed3Tx, err := ed3.ApplyPTYWrite([]byte("\x1b[3Jnew"))
	if err != nil {
		t.Fatalf("apply ED3: %v", err)
	}
	if !ed3Tx.ClearScrollback {
		t.Fatalf("ED3 must expose explicit clear-scrollback boundary to core history: %#v", ed3Tx)
	}
	if len(ed3Tx.EvictedRows) != 0 {
		t.Fatalf("ED3 alone must not fabricate primary eviction proof, got %#v", ed3Tx.EvictedRows)
	}
	screenJoined := strings.Join(trimmedScreenRowsText(ed3.VTerm().ScreenContent().Cells), "|")
	if !strings.Contains(screenJoined, "old1") || !strings.Contains(screenJoined, "old2new") {
		t.Fatalf("ED3 must keep current viewport content, got %q", screenJoined)
	}
}

func TestSemanticSourceResizeEmitsTransaction(t *testing.T) {
	source := NewSemanticSource(8, 2, 100, nil)
	if _, err := source.ApplyPTYWrite([]byte("abcdef\n")); err != nil {
		t.Fatalf("seed write: %v", err)
	}
	tx, err := source.Resize(TerminalSemanticSize{Cols: 4, Rows: 3})
	if err != nil {
		t.Fatalf("resize: %v", err)
	}
	if tx.Seq != 2 || tx.Size != (TerminalSemanticSize{Cols: 4, Rows: 3}) {
		t.Fatalf("unexpected resize transaction header: %#v", tx)
	}
	if !tx.RequiresFullReplace || tx.FullReplaceReason != "resize" {
		t.Fatalf("resize transaction should mark full replace boundary: %#v", tx)
	}
	if tx.PrimaryFrame == nil || tx.PrimaryFrame.Cols != 4 {
		t.Fatalf("resize transaction should include current primary frame projection: %#v", tx)
	}
}

func semanticScrollOutText(row TerminalSemanticScrollOut) string {
	var out string
	for _, cell := range row.Cells {
		out += cell.Content
	}
	for _, run := range row.Runs {
		out += run.Text
	}
	return strings.TrimRight(out, " ")
}

func cellsTextForSemanticTest(cells []TerminalSemanticCell) string {
	var out string
	for _, cell := range cells {
		out += cell.Content
	}
	return out
}

func intsForSemanticSourceTest(values []int) string {
	if len(values) == 0 {
		return ""
	}
	var out strings.Builder
	for index, value := range values {
		if index > 0 {
			out.WriteString(",")
		}
		out.WriteString(strconv.Itoa(value))
	}
	return out.String()
}

func semanticOpTextForSourceTest(op DamageOp) string {
	var out string
	for _, run := range op.Runs {
		out += run.Text
	}
	for _, cell := range op.Cells {
		out += cell.Content
	}
	return strings.TrimRight(out, " ")
}

func BenchmarkR450SemanticSourceFastASCIISGR100K(b *testing.B) {
	payload := r450FastASCIIBenchmarkPayload(100000)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		source := NewSemanticSource(80, 24, 0, nil)
		tx, err := source.ApplyPTYWrite(payload)
		if err != nil {
			b.Fatalf("apply fast ascii/sgr payload: %v", err)
		}
		if len(tx.EvictedRows) == 0 {
			b.Fatalf("expected evicted rows for benchmark payload")
		}
		b.ReportMetric(float64(len(tx.EvictedRows)), "evicted_rows")
	}
}

func BenchmarkR450LineHistorySemanticSourceFastASCIISGR100K(b *testing.B) {
	payload := r450FastASCIIBenchmarkPayload(100000)
	b.SetBytes(int64(len(payload)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		source := NewLineHistorySemanticSource(80, 24, 0, nil)
		tx, err := source.ApplyPTYWrite(payload)
		if err != nil {
			b.Fatalf("apply linehist fast ascii/sgr payload: %v", err)
		}
		if len(tx.EvictedRows) == 0 {
			b.Fatalf("expected evicted rows for benchmark payload")
		}
		if len(tx.Ops) != 0 {
			sample := tx.Ops
			if len(sample) > 4 {
				sample = sample[:4]
			}
			b.Fatalf("linehist source retained ordered text ops: %#v", sample)
		}
		b.ReportMetric(float64(len(tx.EvictedRows)), "evicted_rows")
	}
}

func r450FastASCIIBenchmarkPayload(lines int) []byte {
	var raw strings.Builder
	raw.Grow(lines * 64)
	for i := 1; i <= lines; i++ {
		raw.WriteString("R450 ")
		raw.WriteString(strconv.Itoa(i))
		raw.WriteString(" \x1b[32mOK\x1b[0m bounded-history-memory-pressure")
		raw.WriteString("\r\n")
	}
	return []byte(raw.String())
}
