package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
)

func writeTerminalFormat(writer io.Writer, source string, views []terminalView) error {
	tmpl, err := template.New("terminal-format").Option("missingkey=error").Parse(source)
	if err != nil {
		return usageCLIError(fmt.Sprintf("invalid --format template: %v", err))
	}
	for _, view := range views {
		var rendered bytes.Buffer
		if err := tmpl.Execute(&rendered, terminalFormatData(view)); err != nil {
			return usageCLIError(fmt.Sprintf("invalid --format field: %v", err))
		}
		if _, err := writer.Write(rendered.Bytes()); err != nil {
			return err
		}
		if !strings.HasSuffix(rendered.String(), "\n") {
			if _, err := io.WriteString(writer, "\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func terminalFormatData(view terminalView) map[string]any {
	return map[string]any{
		"target": view.Target, "endpoint_id": view.EndpointID, "terminal_id": view.TerminalID,
		"name": view.Name, "command": append([]string(nil), view.Command...), "command_text": strings.Join(view.Command, " "),
		"cwd": view.CWD, "tags": cloneTerminalTags(view.Tags), "state": view.State,
		"cols": view.Cols, "rows": view.Rows, "created_at": view.CreatedAt, "exit_code": view.ExitCode, "exited_at": view.ExitedAt,
	}
}
