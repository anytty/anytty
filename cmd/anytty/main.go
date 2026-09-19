// Command anytty is the compatibility entry point for the AnyTTY CLI. The
// implementation lives in clients/cli; this binary only forwards the version
// injected at link time.
package main

import (
	"os"

	cli "github.com/anytty/anytty/clients/cli"
)

// version is replaced for release builds with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Main(version))
}
