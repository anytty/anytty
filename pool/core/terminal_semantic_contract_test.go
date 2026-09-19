package core

import (
	"testing"

	vterm "github.com/anytty/anytty/vterm/vterm"
)

func TestR302CoreConsumesVTermSemanticTransactionContract(t *testing.T) {
	source := vterm.NewSemanticSource(12, 2, 100, nil)
	var _ TerminalSemanticSource = source

	tx, err := source.ApplyPTYWrite([]byte("one\r\ntwo\r\nthree"))
	if err != nil {
		t.Fatalf("apply PTY write: %v", err)
	}
	if tx.Size != (TerminalSemanticSize{Cols: 12, Rows: 2}) {
		t.Fatalf("transaction should preserve PTY size, got %#v", tx.Size)
	}
	if len(tx.Ops) == 0 {
		t.Fatalf("transaction must expose ordered ops: %#v", tx)
	}
	if len(tx.PrimaryScrollOut) != 0 {
		t.Fatalf("ordinary primary output must not duplicate ordered ops as transaction scroll-out proof: %#v", tx.PrimaryScrollOut)
	}
	if tx.PrimaryFrame != nil || tx.AltFrame != nil {
		t.Fatalf("ordinary primary output history truth should stay in ordered ops, got frame side proof: %#v", tx)
	}
}
