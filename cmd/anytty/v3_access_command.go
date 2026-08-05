package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	clientprotocol "github.com/anytty/anytty/client/adapter/protocol"
	clientendpoint "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/spf13/cobra"
)

func v3AccessCommand(socket *string, logFile *string) *cobra.Command {
	command := &cobra.Command{Use: "access", Short: "Inspect and revoke daemon client-bound access"}
	command.AddCommand(v3AccessIdentityCommand(socket, logFile))
	command.AddCommand(v3AccessListCommand(socket, logFile))
	command.AddCommand(v3AccessRevokeCommand(socket, logFile))
	return command
}

func v3AccessIdentityCommand(socket *string, logFile *string) *cobra.Command {
	jsonOutput := false
	command := &cobra.Command{
		Use: "identity", Short: "Show the local daemon global DeviceIdentity", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := dialOrStartV3ClientContext(cmd.Context(), resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), nil)
			if err != nil {
				return err
			}
			defer client.Close()
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			response, err := clientprotocol.VerifyDaemonIdentityResult(cmd.Context(), application, clientendpoint.DaemonIdentity{})
			if err != nil {
				return err
			}
			result := response.GetIdentity()
			view := struct {
				DeviceID          string `json:"device_id"`
				DeviceFingerprint string `json:"device_fingerprint"`
				DevicePublicKey   string `json:"device_public_key"`
			}{result.GetDeviceId(), result.GetDeviceFingerprint(), base64.RawURLEncoding.EncodeToString(result.GetDevicePublicKey())}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Device", Value: view.DeviceID},
				cliField{Label: "Fingerprint", Value: view.DeviceFingerprint},
				cliField{Label: "Public key", Value: view.DevicePublicKey},
			)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func v3AccessListCommand(socket *string, logFile *string) *cobra.Command {
	jsonOutput := false
	command := &cobra.Command{
		Use: "list", Short: "List daemon-local client access grants", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := dialOrStartV3ClientContext(cmd.Context(), resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), nil)
			if err != nil {
				return err
			}
			defer client.Close()
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			response, err := application.ClientAccessList(cmd.Context(), &apipb.ClientAccessListCommand{})
			if err != nil {
				return err
			}
			result := response.GetAccess()
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result.GetRecords())
			}
			if len(result.GetRecords()) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No client access grants")
				return nil
			}
			rows := make([][]string, 0, len(result.GetRecords()))
			for _, record := range result.GetRecords() {
				state := "active"
				if record.GetRevokedAtUnixNano() != 0 {
					state = "revoked"
				}
				rows = append(rows, []string{record.GetGrantId(), record.GetClientLabel(), state, formatTerminalTime(time.Unix(0, record.GetExpiresAtUnixNano()).UTC()), record.GetSubjectKeyFingerprint()})
			}
			return writeCLITable(cmd.OutOrStdout(), []string{"GRANT", "CLIENT", "STATE", "EXPIRES", "SUBJECT"}, rows)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func v3AccessRevokeCommand(socket *string, logFile *string) *cobra.Command {
	jsonOutput := false
	command := &cobra.Command{
		Use: "revoke GRANT_ID", Short: "Revoke one daemon-local client access grant", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			grantID := strings.TrimSpace(args[0])
			if grantID == "" {
				return usageCLIError("grant id is required")
			}
			client, err := dialOrStartV3ClientContext(cmd.Context(), resolveV3Socket(*socket), resolveV3LogFilePath(*logFile), nil)
			if err != nil {
				return err
			}
			defer client.Close()
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			response, err := application.ClientAccessRevoke(cmd.Context(), &apipb.ClientAccessRevokeCommand{Request: &remoteauthpb.ClientAccessRevokeRequest{GrantId: grantID}})
			if err != nil {
				return err
			}
			result := response.GetRecord()
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Revoked client access grant %s for %s\n", result.GetGrantId(), result.GetSubjectKeyFingerprint())
			return nil
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}
