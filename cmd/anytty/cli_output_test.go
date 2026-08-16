package main

import (
	"bytes"
	"testing"
)

func TestWriteCLITableAlignsUnicodeByTerminalCells(t *testing.T) {
	var output bytes.Buffer
	err := writeCLITable(&output, []string{"ID", "LABEL", "STATE"}, [][]string{
		{"a", "中文", "on"},
		{"long", "x", "off"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "ID    LABEL  STATE\n" +
		"a     中文   on\n" +
		"long  x      off\n"
	if output.String() != want {
		t.Fatalf("table output:\n%q\nwant:\n%q", output.String(), want)
	}
}

func TestWriteCLIOutputSanitizesCellsAndAlignsFields(t *testing.T) {
	var output bytes.Buffer
	if err := writeCLITable(&output, nil, [][]string{{"first\nline", ""}, {"二", "ok"}}); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "first line  -\n二          ok\n"; got != want {
		t.Fatalf("table output = %q, want %q", got, want)
	}

	output.Reset()
	if err := writeCLIFields(&output,
		cliField{Label: "ID", Value: "studio"},
		cliField{Label: "连接方式", Value: "direct"},
	); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "ID        studio\n连接方式  direct\n"; got != want {
		t.Fatalf("field output = %q, want %q", got, want)
	}
}

func TestWriteCLITableUsesRecordsWhenTerminalIsNarrow(t *testing.T) {
	var output bytes.Buffer
	if err := writeCLITableAtWidth(&output, []string{"ID", "LABEL"}, [][]string{
		{"device-long-id", "阿里云服务"},
		{"local", "工作站"},
	}, 24); err != nil {
		t.Fatal(err)
	}
	want := "ID     device-long-id\n" +
		"LABEL  阿里云服务\n\n" +
		"ID     local\n" +
		"LABEL  工作站\n"
	if output.String() != want {
		t.Fatalf("narrow table output:\n%q\nwant:\n%q", output.String(), want)
	}
}

func TestWriteCLIFieldsWrapsLongValuesAndFixedRowsAlign(t *testing.T) {
	var output bytes.Buffer
	if err := writeCLIFieldsAtWidth(&output, 16, cliField{Label: "LABEL", Value: "abcdefghijklmnop"}); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "LABEL  abcdefghi\n       jklmnop\n"; got != want {
		t.Fatalf("wrapped fields = %q, want %q", got, want)
	}

	output.Reset()
	if err := writeCLIFixedRow(&output, []int{5, 4}, "ID", "类型", "TARGET"); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), "ID     类型  TARGET\n"; got != want {
		t.Fatalf("fixed row = %q, want %q", got, want)
	}
}
