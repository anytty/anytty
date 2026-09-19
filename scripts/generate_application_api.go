// Command generate_application_api regenerates the four application command
// codegen outputs from proto/access/apipb/application_commands.csv.
//
// Usage:
//
//	go run ./scripts/generate_application_api.go          # write outputs
//	go run ./scripts/generate_application_api.go -check   # verify no drift
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/anytty/anytty/internal/appcodegen"
)

func main() {
	check := flag.Bool("check", false, "verify generated files are up to date without writing")
	flag.Parse()
	if err := run(*check); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(check bool) error {
	files, err := appcodegen.Generate(".")
	if err != nil {
		return err
	}
	for _, file := range files {
		if check {
			current, err := os.ReadFile(file.Path)
			if err != nil {
				return err
			}
			if !bytes.Equal(current, file.Content) {
				return fmt.Errorf("%s is out of date; run go run ./scripts/generate_application_api.go", file.Path)
			}
			continue
		}
		if err := os.WriteFile(file.Path, file.Content, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", file.Path, err)
		}
	}
	return nil
}
