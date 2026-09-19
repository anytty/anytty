// Command anyttyd runs the AnyTTY daemon in the foreground. It is the daemon
// product-line entry point and delegates to the shared CLI implementation's
// `daemon run` command.
package main

import (
	"os"

	cli "github.com/anytty/anytty/clients/cli"
)

func main() {
	os.Exit(cli.RunDaemon(os.Args[1:]))
}
