//go:build anytty_dev_commands

package main

import "github.com/spf13/cobra"

func init() {
	addDevelopmentCommands = func(command *cobra.Command, socket, logFile, configPath *string) {
		command.AddCommand(v3Command(socket, logFile, configPath))
	}
}
