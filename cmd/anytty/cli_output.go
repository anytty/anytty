package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"golang.org/x/term"
)

const cliColumnGap = 2

type cliField struct {
	Label string
	Value string
}

// writeCLITable renders human output with terminal-cell-aware columns.
func writeCLITable(writer io.Writer, header []string, rows [][]string) error {
	return writeCLITableAtWidth(writer, header, rows, cliTerminalWidth(writer))
}

func writeCLITableAtWidth(writer io.Writer, header []string, rows [][]string, maxWidth int) error {
	columnCount := len(header)
	for _, row := range rows {
		if len(row) > columnCount {
			columnCount = len(row)
		}
	}
	if columnCount == 0 {
		return nil
	}

	lines := make([][]string, 0, len(rows)+1)
	if len(header) > 0 {
		lines = append(lines, normalizeCLIRow(header, columnCount))
	}
	for _, row := range rows {
		lines = append(lines, normalizeCLIRow(row, columnCount))
	}

	widths := make([]int, columnCount)
	for _, line := range lines {
		for column, cell := range line {
			if width := xansi.StringWidth(cell); width > widths[column] {
				widths[column] = width
			}
		}
	}
	if len(header) > 0 && len(rows) > 0 && maxWidth > 0 && cliTableWidth(widths) > maxWidth {
		return writeCLIRecords(writer, lines[0], lines[1:], maxWidth)
	}
	for _, line := range lines {
		var output strings.Builder
		for column, cell := range line {
			output.WriteString(cell)
			if column < columnCount-1 {
				padding := widths[column] - xansi.StringWidth(cell) + cliColumnGap
				output.WriteString(strings.Repeat(" ", padding))
			}
		}
		output.WriteByte('\n')
		if _, err := io.WriteString(writer, output.String()); err != nil {
			return err
		}
	}
	return nil
}

// writeCLIFields aligns labels for compact human-readable detail output.
func writeCLIFields(writer io.Writer, fields ...cliField) error {
	return writeCLIFieldsAtWidth(writer, cliTerminalWidth(writer), fields...)
}

func writeCLIFieldsAtWidth(writer io.Writer, maxWidth int, fields ...cliField) error {
	width := 0
	for index := range fields {
		fields[index].Label = cleanCLICell(fields[index].Label)
		fields[index].Value = cleanCLICell(fields[index].Value)
		if labelWidth := xansi.StringWidth(fields[index].Label); labelWidth > width {
			width = labelWidth
		}
	}
	for _, field := range fields {
		padding := width - xansi.StringWidth(field.Label) + cliColumnGap
		prefix := field.Label + strings.Repeat(" ", padding)
		if maxWidth <= 0 || xansi.StringWidth(prefix)+xansi.StringWidth(field.Value) <= maxWidth {
			if _, err := fmt.Fprintf(writer, "%s%s\n", prefix, field.Value); err != nil {
				return err
			}
			continue
		}
		available := maxWidth - xansi.StringWidth(prefix)
		if available >= 8 {
			if err := writeCLIWrappedValue(writer, prefix, field.Value, available); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintln(writer, field.Label); err != nil {
			return err
		}
		available = maxWidth - cliColumnGap
		if available < 1 {
			available = 1
		}
		if err := writeCLIWrappedValue(writer, strings.Repeat(" ", cliColumnGap), field.Value, available); err != nil {
			return err
		}
	}
	return nil
}

func writeCLIFixedRow(writer io.Writer, widths []int, cells ...string) error {
	var output strings.Builder
	for column, cell := range cells {
		cell = cleanCLICell(cell)
		output.WriteString(cell)
		if column >= len(cells)-1 {
			continue
		}
		width := 0
		if column < len(widths) {
			width = widths[column]
		}
		padding := max(width-xansi.StringWidth(cell), 0) + cliColumnGap
		output.WriteString(strings.Repeat(" ", padding))
	}
	output.WriteByte('\n')
	_, err := io.WriteString(writer, output.String())
	return err
}

func writeCLIRecords(writer io.Writer, header []string, rows [][]string, maxWidth int) error {
	for rowIndex, row := range rows {
		fields := make([]cliField, 0, len(header))
		for column, label := range header {
			fields = append(fields, cliField{Label: label, Value: row[column]})
		}
		if err := writeCLIFieldsAtWidth(writer, maxWidth, fields...); err != nil {
			return err
		}
		if rowIndex < len(rows)-1 {
			if _, err := io.WriteString(writer, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeCLIWrappedValue(writer io.Writer, prefix, value string, width int) error {
	lines := strings.Split(xansi.Hardwrap(value, width, false), "\n")
	indent := strings.Repeat(" ", xansi.StringWidth(prefix))
	for index, line := range lines {
		linePrefix := indent
		if index == 0 {
			linePrefix = prefix
		}
		if _, err := fmt.Fprintf(writer, "%s%s\n", linePrefix, line); err != nil {
			return err
		}
	}
	return nil
}

func cliTableWidth(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	if len(widths) > 1 {
		total += (len(widths) - 1) * cliColumnGap
	}
	return total
}

func cliTerminalWidth(writer io.Writer) int {
	file, ok := writer.(*os.File)
	if !ok || !term.IsTerminal(int(file.Fd())) {
		return 0
	}
	width, _, err := term.GetSize(int(file.Fd()))
	if err != nil || width < 20 {
		return 0
	}
	return width
}

func normalizeCLIRow(row []string, columnCount int) []string {
	normalized := make([]string, columnCount)
	for column := range normalized {
		if column < len(row) {
			normalized[column] = cleanCLICell(row[column])
		} else {
			normalized[column] = "-"
		}
	}
	return normalized
}

func cleanCLICell(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ", "\t", " ").Replace(value)
	if value == "" {
		return "-"
	}
	return value
}
