package main

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed THIRD_PARTY_NOTICES.txt
var thirdPartyNotices string

func v3LicensesCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "licenses",
		Short: "print third-party licenses bundled with this anytty build",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(thirdPartyNotices))
			return err
		},
	}
}
