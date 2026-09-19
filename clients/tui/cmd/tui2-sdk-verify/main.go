// Command tui2-sdk-verify runs the TUI v2 SDK conformance fixtures against a
// candidate layout program, so any SDK (official or generated) can prove it
// speaks the protocol:
//
//	tui2-sdk-verify                        # official Go SDK, in process
//	tui2-sdk-verify --cmd "python3 sdk/python/conformance.py"
//	tui2-sdk-verify --cmd "node sdk/ts/conformance.js"
//
// Exit code 0 means every fixture passed; 1 means at least one failed. The
// runner never trusts the candidate: it decodes every VIEW/RESULT frame and
// checks epoch/rev/claim/text/method/params semantically (PROTOCOL §0-§6).
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/anytty/anytty/clients/tui/conformance"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	flags := flag.NewFlagSet("tui2-sdk-verify", flag.ContinueOnError)
	fixturesPath := flags.String("fixtures", "tui2/conformance/fixtures.jsonl", "fixture file (JSON Lines)")
	command := flags.String("cmd", "", "candidate program command (default: the official Go SDK, in process)")
	timeout := flags.Duration("timeout", 10*time.Second, "per-fixture timeout")
	list := flags.Bool("list", false, "list fixture names and exit")
	only := flags.String("only", "", "run only fixtures whose name contains this substring")
	if err := flags.Parse(argv); err != nil {
		return 2
	}
	fixtures, err := conformance.LoadFixtures(*fixturesPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "tui2-sdk-verify:", err)
		return 2
	}
	for _, fixture := range fixtures {
		if err := fixture.Validate(); err != nil {
			fmt.Fprintln(os.Stderr, "tui2-sdk-verify:", err)
			return 2
		}
	}
	if *list {
		for _, fixture := range fixtures {
			fmt.Printf("%s\t%s\n", fixture.Name, fixture.Desc)
		}
		return 0
	}
	if *only != "" {
		filtered := fixtures[:0]
		for _, fixture := range fixtures {
			if strings.Contains(fixture.Name, *only) {
				filtered = append(filtered, fixture)
			}
		}
		fixtures = filtered
	}
	var factory conformance.Factory
	if *command == "" {
		factory = conformance.NewRefFactory()
	} else {
		parsed, err := splitCommand(*command)
		if err != nil {
			fmt.Fprintln(os.Stderr, "tui2-sdk-verify:", err)
			return 2
		}
		factory = conformance.CommandFactory{Argv: parsed}
	}
	results := conformance.Run(fixtures, factory, *timeout)
	fmt.Print(conformance.Summary(results))
	for _, result := range results {
		if !result.Passed {
			return 1
		}
	}
	return 0
}

// splitCommand splits a command line into argv honoring single/double quotes
// and backslash escapes; it never invokes a shell (shared shape with the
// host's -shell parsing so users see one rule).
func splitCommand(s string) ([]string, error) {
	var argv []string
	var current strings.Builder
	inWord := false
	quote := rune(0)
	escaped := false
	for _, r := range s {
		switch {
		case escaped:
			current.WriteRune(r)
			inWord = true
			escaped = false
		case r == '\\' && quote != '\'':
			escaped = true
			inWord = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
			inWord = true
		case r == ' ' || r == '\t' || r == '\n':
			if inWord {
				argv = append(argv, current.String())
				current.Reset()
				inWord = false
			}
		default:
			current.WriteRune(r)
			inWord = true
		}
	}
	if escaped {
		return nil, fmt.Errorf("trailing backslash")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unbalanced quote")
	}
	if inWord {
		argv = append(argv, current.String())
	}
	if len(argv) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return argv, nil
}
