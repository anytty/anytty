package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	endpointdomain "github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/apipb"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/remoteauth"
	"github.com/anytty/anytty/shared/securefs"
	unixtransport "github.com/anytty/anytty/shared/transport/unix"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	updateV3ConnectionRegistry = endpointdomain.UpdateContext
	dialV3PairingTransport     = unixtransport.DialContext
	v3PairHostname             = os.Hostname
	v3PairOutputIsTerminal     = func(output io.Writer) bool {
		file, ok := output.(*os.File)
		return ok && term.IsTerminal(int(file.Fd()))
	}
	v3PairTerminalSize = func(output io.Writer) (columns int, rows int, ok bool) {
		file, ok := output.(*os.File)
		if !ok {
			return 0, 0, false
		}
		columns, rows, err := term.GetSize(int(file.Fd()))
		return columns, rows, err == nil
	}
)

func v3PairCommand(socket *string, logFile *string) *cobra.Command {
	command := &cobra.Command{Use: "pair", Short: "Create or redeem a client-bound daemon pairing claim"}
	command.AddCommand(v3PairCreateCommand(socket, logFile))
	command.AddCommand(v3PairImportCommand())
	command.AddCommand(v3PairInspectCommand())
	return command
}

type pairInspectView struct {
	SchemaVersion     uint32          `json:"schema_version"`
	Kind              string          `json:"kind"`
	DeviceID          string          `json:"device_id"`
	DeviceFingerprint string          `json:"device_fingerprint"`
	ExpiresAt         string          `json:"expires_at"`
	Routes            []pairRouteView `json:"routes"`
}

type pairRouteView struct {
	RouteID             string   `json:"route_id"`
	Kind                string   `json:"kind"`
	SignalingAddresses  []string `json:"signaling_addresses,omitempty"`
	ICETCPAddresses     []string `json:"ice_tcp_addresses,omitempty"`
	AdvertisedAddresses []string `json:"advertised_addresses,omitempty"`
	ServerName          string   `json:"server_name,omitempty"`
	TargetDeviceID      string   `json:"target_device_id,omitempty"`
	DaemonID            string   `json:"daemon_id,omitempty"`
	EdgeID              string   `json:"edge_id,omitempty"`
	PublicEndpoint      string   `json:"public_endpoint,omitempty"`
	RelayMode           string   `json:"relay_mode,omitempty"`
}

func v3PairInspectCommand() *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "inspect <CLAIM_FILE|->", Short: "Verify one-time pairing claim metadata", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := readV3PairingClaim(cmd.Context(), cmd.InOrStdin(), args[0])
			if err != nil {
				return err
			}
			offer, err := remoteauth.ParsePairingClaimOffer(payload, time.Now().UTC())
			clear(payload)
			if err != nil {
				return err
			}
			view := pairInspectView{
				SchemaVersion: offer.GetSchemaVersion(), Kind: "pairing_claim", DeviceID: offer.GetDeviceId(),
				DeviceFingerprint: remoteauth.Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey())), ExpiresAt: formatTerminalTime(time.Unix(0, offer.GetExpiresAtUnixNano()).UTC()),
				Routes: pairingClaimRouteViews(offer),
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Device", Value: view.DeviceID},
				cliField{Label: "Fingerprint", Value: view.DeviceFingerprint},
				cliField{Label: "Expires", Value: view.ExpiresAt},
			)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print redacted machine-readable JSON")
	return command
}

func v3PairCreateCommand(socket *string, logFile *string) *cobra.Command {
	var outputPath string
	var qrOutputPath string
	var rawOutput bool
	var textOutput bool
	var commandOutput bool
	var label string
	var terminalID string
	var ticketTTL time.Duration
	var grantLifetime time.Duration
	var routeSpecs []string
	var directID string
	var directName string
	var signalingAddresses []string
	var iceTCPAddresses []string
	var serverName string
	var sshID string
	var sshName string
	var sshHost string
	var sshPort uint16
	var sshUser string
	var sshHostKeys []string
	command := &cobra.Command{
		Use:   "create",
		Short: "Issue a short-lived one-time pairing claim from the local daemon",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			socketPath := resolveV3Socket(*socket)
			selectedOutputs := 0
			for _, selected := range []bool{rawOutput, textOutput, commandOutput, strings.TrimSpace(outputPath) != "", strings.TrimSpace(qrOutputPath) != ""} {
				if selected {
					selectedOutputs++
				}
			}
			if selectedOutputs > 1 {
				return usageCLIError("pair create --raw, --text, --command, --out, and --qr-file are mutually exclusive")
			}
			client, err := dialOrStartV3ClientContext(cmd.Context(), socketPath, resolveV3LogFilePath(*logFile), nil)
			if err != nil {
				return err
			}
			defer client.Close()
			label = strings.TrimSpace(label)
			if label == "" {
				if hostname, hostnameErr := v3PairHostname(); hostnameErr == nil {
					label = strings.TrimSpace(hostname)
				}
			}
			scope := remoteauth.FullDaemonScope()
			if terminalID = strings.TrimSpace(terminalID); terminalID != "" {
				terminalID, err = localPairTerminalID(terminalID)
				if err != nil {
					return err
				}
				scope = remoteauth.Scope{TerminalID: terminalID}
			}
			if ticketTTL <= 0 || grantLifetime <= 0 {
				return usageCLIError("pair create ticket and grant ttl must be positive")
			}
			application, err := newLocalApplicationSession(client)
			if err != nil {
				return err
			}
			routes, err := v3PairingRoutes(v3PairRouteFlags{
				Routes: routeSpecs, DirectID: directID, DirectName: directName, SignalingAddresses: signalingAddresses,
				ICETCPAddresses: iceTCPAddresses, ServerName: serverName, SSHID: sshID, SSHName: sshName,
				SSHHost: sshHost, SSHPort: sshPort, SSHUser: sshUser, SSHHostKeys: sshHostKeys,
			})
			if err != nil {
				return usageCLIError(err.Error())
			}
			response, err := application.ClientAccessTicketCreate(cmd.Context(), &apipb.ClientAccessTicketCreateCommand{Request: &remoteauthpb.ClientAccessTicketCreateRequest{
				Label: label, Scope: clientAccessScopeToProto(scope), TicketTtlSeconds: int64(ticketTTL / time.Second), GrantLifetimeSeconds: int64(grantLifetime / time.Second),
				Routes: routes,
			}})
			if err != nil {
				return err
			}
			result := response.GetTicket()
			payload := result.GetClaimOffer()
			if len(payload) == 0 {
				return fmt.Errorf("daemon did not return a pairing claim offer")
			}
			if rawOutput {
				_, err = cmd.OutOrStdout().Write(payload)
				return err
			}
			portablePayload := v3PairingBootstrapURI(payload)
			if textOutput {
				_, err = fmt.Fprintln(cmd.OutOrStdout(), portablePayload)
				return err
			}
			if commandOutput {
				return renderV3PairingCommand(cmd.OutOrStdout(), payload)
			}
			if strings.TrimSpace(qrOutputPath) != "" {
				png, pngErr := renderV3PairingPNG(portablePayload)
				if pngErr != nil {
					return pngErr
				}
				if err := renderV3PairingPreview(cmd.OutOrStdout(), payload, time.Unix(0, result.GetExpiresAtUnixNano()).UTC()); err != nil {
					return err
				}
				if err := writeV3PrivateFile(qrOutputPath, png); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Pairing QR PNG written to %s\n", qrOutputPath)
				return nil
			}
			if strings.TrimSpace(outputPath) == "" {
				if !v3PairOutputIsTerminal(cmd.OutOrStdout()) {
					return usageCLIError("pair create requires an interactive terminal; use --command, --text, --qr-file FILE, --raw, or --out FILE")
				}
				return renderV3PairingQR(cmd.OutOrStdout(), payload, time.Unix(0, result.GetExpiresAtUnixNano()).UTC())
			}
			if err := renderV3PairingPreview(cmd.OutOrStdout(), payload, time.Unix(0, result.GetExpiresAtUnixNano()).UTC()); err != nil {
				return err
			}
			if err := writeV3PrivateFile(outputPath, payload); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Pairing claim written to %s\n", outputPath)
			return nil
		},
	}
	command.Flags().StringVar(&outputPath, "out", "", "write the one-time pairing claim to an owner-only file")
	command.Flags().StringVar(&qrOutputPath, "qr-file", "", "write a square pairing QR PNG to an owner-only file")
	command.Flags().BoolVar(&rawOutput, "raw", false, "write the one-time pairing claim to stdout for explicit owner scripting")
	command.Flags().BoolVar(&textOutput, "text", false, "write the portable pairing URI to stdout for copying")
	command.Flags().BoolVar(&commandOutput, "command", false, "write a copyable one-command pairing import")
	command.Flags().StringVar(&label, "label", "", "daemon display label (defaults to this host name)")
	command.Flags().StringVar(&terminalID, "terminal", "", "limit the capability to one terminal instead of daemon-wide access")
	command.Flags().DurationVar(&ticketTTL, "ttl", 10*time.Minute, "one-time ticket lifetime")
	command.Flags().DurationVar(&grantLifetime, "grant-ttl", 90*24*time.Hour, "bound capability lifetime")
	command.Flags().StringArrayVar(&routeSpecs, "route", nil, "pairing Route: direct, ssh, or a strict Route URI (repeatable)")
	command.Flags().StringVar(&directID, "direct-id", "", "stable ID suffix for the parameterized Direct Route")
	command.Flags().StringVar(&directName, "direct-name", "", "display name for the parameterized Direct Route")
	command.Flags().StringArrayVar(&signalingAddresses, "signaling-address", nil, "published Direct signaling HOST:PORT (repeatable; requires --ice-tcp-address)")
	command.Flags().StringArrayVar(&iceTCPAddresses, "ice-tcp-address", nil, "published Direct ICE-TCP HOST:PORT (repeatable; requires --signaling-address)")
	command.Flags().StringVar(&serverName, "server-name", "", "optional Direct server name bound into the signed Route hint")
	command.Flags().StringVar(&sshID, "ssh-id", "", "stable ID suffix for the parameterized SSH Route")
	command.Flags().StringVar(&sshName, "ssh-name", "", "display name for the parameterized SSH Route")
	command.Flags().StringVar(&sshHost, "ssh-host", "", "SSH server host for the parameterized SSH Route")
	command.Flags().Uint16Var(&sshPort, "ssh-port", 0, "SSH server port (defaults to 22)")
	command.Flags().StringVar(&sshUser, "ssh-user", "", "SSH user for the parameterized SSH Route")
	command.Flags().StringArrayVar(&sshHostKeys, "ssh-host-key", nil, "pinned SSH SHA256 host-key fingerprint (repeatable)")
	return command
}

func clientAccessScopeToProto(scope remoteauth.Scope) *remoteauthpb.ClientAccessScope {
	return &remoteauthpb.ClientAccessScope{AllowDaemon: scope.AllowDaemon, TerminalId: scope.TerminalID, MachineEventsOnly: scope.MachineEventsOnly, FileReadMetadata: scope.FileReadMetadata, FileReadContent: scope.FileReadContent, FileWriteContent: scope.FileWriteContent, FileMutate: scope.FileMutate, ManageClientAccess: scope.ManageClientAccess}
}

// renderV3PairingQR 把 daemon 内存持有的短期一次性 claim 编码进高对比度终端二维码。
// 二维码不包含 ticket、scope 或 grant；调用方仍应在扫描后清屏，避免 claim 在有效期内被旁观者使用。
func renderV3PairingQR(output io.Writer, payload []byte, expiresAt time.Time) error {
	portablePayload := v3PairingBootstrapURI(payload)
	code, err := qrcode.New(portablePayload, qrcode.Low)
	if err != nil {
		return fmt.Errorf("encode pairing QR: %w", err)
	}
	bitmap := code.Bitmap()
	if len(bitmap) == 0 {
		return fmt.Errorf("encode pairing QR: empty bitmap")
	}
	var preview strings.Builder
	if err := renderV3PairingPreview(&preview, payload, expiresAt); err != nil {
		return err
	}
	requiredColumns := len(bitmap)
	for _, line := range strings.Split(preview.String(), "\n") {
		if len(line) > requiredColumns {
			requiredColumns = len(line)
		}
	}
	for _, line := range []string{
		"Scan with the AnyTTY App",
		"This QR contains a one-time pairing claim. Clear the terminal after scanning.",
	} {
		if len(line) > requiredColumns {
			requiredColumns = len(line)
		}
	}
	requiredRows := strings.Count(preview.String(), "\n") + 1 + (len(bitmap)+1)/2 + 1
	if columns, rows, ok := v3PairTerminalSize(output); ok && (columns < requiredColumns || rows < requiredRows) {
		return usageCLIError(fmt.Sprintf(
			"pairing QR requires at least %dx%d terminal cells (current %dx%d); enlarge the terminal or use --text or --qr-file FILE",
			requiredColumns, requiredRows, columns, rows,
		))
	}
	if _, err := io.WriteString(output, preview.String()); err != nil {
		return err
	}
	if _, err = io.WriteString(output, "Scan with the AnyTTY App\n"); err != nil {
		return err
	}
	for row := 0; row < len(bitmap); row += 2 {
		for column := range bitmap[row] {
			top := bitmap[row][column]
			bottom := false
			if row+1 < len(bitmap) {
				bottom = bitmap[row+1][column]
			}
			switch {
			case top && bottom:
				_, err = io.WriteString(output, "\x1b[40m ")
			case !top && !bottom:
				_, err = io.WriteString(output, "\x1b[47m ")
			case top:
				_, err = io.WriteString(output, "\x1b[30;47m▀")
			default:
				_, err = io.WriteString(output, "\x1b[30;47m▄")
			}
			if err != nil {
				return err
			}
		}
		if _, err = io.WriteString(output, "\x1b[0m\n"); err != nil {
			return err
		}
	}
	_, err = io.WriteString(output, "\x1b[0mThis QR contains a one-time pairing claim. Clear the terminal after scanning.\n")
	return err
}

func v3PairingBootstrapURI(payload []byte) string {
	return remoteauth.EncodePairingClaimCode(payload)
}

func renderV3PairingCommand(output io.Writer, payload []byte) error {
	offer, err := remoteauth.ParsePairingClaimOffer(payload, time.Now().UTC())
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "anytty pair import --id %s %s\n", shellQuotePairingArgument(offer.GetDeviceId()), shellQuotePairingArgument(v3PairingBootstrapURI(payload)))
	return err
}

func shellQuotePairingArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

// renderV3PairingPNG 生成带 quiet zone 的正方形位图，供无法完整显示终端二维码时离线展示。
func renderV3PairingPNG(portablePayload string) ([]byte, error) {
	code, err := qrcode.New(portablePayload, qrcode.Low)
	if err != nil {
		return nil, fmt.Errorf("encode pairing QR: %w", err)
	}
	png, err := code.PNG(1024)
	if err != nil {
		return nil, fmt.Errorf("encode pairing QR PNG: %w", err)
	}
	return png, nil
}

func renderV3PairingPreview(output io.Writer, payload []byte, expiresAt time.Time) error {
	offer, err := remoteauth.ParsePairingClaimOffer(payload, time.Now().UTC())
	if err != nil {
		return err
	}
	if err := writeCLIFields(output,
		cliField{Label: "Device", Value: offer.GetDeviceId()},
		cliField{Label: "Fingerprint", Value: remoteauth.Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey()))},
		cliField{Label: "Expires", Value: formatTerminalTime(expiresAt)},
	); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "\nRoutes\n"); err != nil {
		return err
	}
	rows := make([][]string, 0, len(offer.GetRoutes()))
	for _, route := range offer.GetRoutes() {
		if direct := route.GetDirectWebrtcTcp(); direct != nil {
			details := fmt.Sprintf("signaling=%s, ice-tcp=%s", direct.GetSignalingAddress(), direct.GetIceTcpAddress())
			rows = append(rows, []string{route.GetRouteId(), "direct", details})
			continue
		}
		if managed := route.GetManagedWebrtc(); managed != nil {
			rows = append(rows, []string{route.GetRouteId(), "cloud", fmt.Sprintf("edge=%s, endpoint=%s", managed.GetEdgeId(), managed.GetPublicEndpoint())})
			continue
		}
		if ssh := route.GetSshWebrtcTcp(); ssh != nil {
			rows = append(rows, []string{route.GetRouteId(), "ssh", fmt.Sprintf("%s@%s:%d", ssh.GetUser(), ssh.GetHost(), ssh.GetPort())})
		}
	}
	return writeCLITable(output, []string{"ID", "KIND", "DETAILS"}, rows)
}

func pairingClaimRouteViews(offer *remoteauthpb.PairingClaimOffer) []pairRouteView {
	views := make([]pairRouteView, 0, len(offer.GetRoutes()))
	for _, route := range offer.GetRoutes() {
		if direct := route.GetDirectWebrtcTcp(); direct != nil {
			views = append(views, pairRouteView{
				RouteID: route.GetRouteId(), Kind: string(endpointdomain.RouteDirectWebRTCTCP),
				SignalingAddresses: []string{direct.GetSignalingAddress()}, ICETCPAddresses: []string{direct.GetIceTcpAddress()}, ServerName: direct.GetServerName(),
			})
			continue
		}
		if managed := route.GetManagedWebrtc(); managed != nil {
			views = append(views, pairRouteView{
				RouteID: route.GetRouteId(), Kind: string(endpointdomain.RouteManagedWebRTC), TargetDeviceID: offer.GetDeviceId(),
				DaemonID: managed.GetDaemonId(), EdgeID: managed.GetEdgeId(), PublicEndpoint: managed.GetPublicEndpoint(), ServerName: managed.GetServerName(),
			})
			continue
		}
		if ssh := route.GetSshWebrtcTcp(); ssh != nil {
			views = append(views, pairRouteView{RouteID: route.GetRouteId(), Kind: string(endpointdomain.RouteSSHWebRTCTCP)})
		}
	}
	return views
}

func localPairTerminalID(target string) (string, error) {
	target = strings.TrimSpace(target)
	if endpointID, terminalID, found := strings.Cut(target, ":"); found {
		if endpointID != string(endpointdomain.DefaultEndpointID) || terminalID == "" || strings.Contains(terminalID, ":") {
			return "", usageCLIError("pair create can only scope the local daemon target local:TERMINAL_ID")
		}
		return terminalID, nil
	}
	if target == "" {
		return "", usageCLIError("terminal target cannot be empty")
	}
	return target, nil
}

func v3PairImportCommand() *cobra.Command {
	var endpointID string
	var label string
	var registryPath string
	var pairingSocket string
	var clientLabel string
	var allowScopeExpansion bool
	command := &cobra.Command{
		Use:   "import <CLAIM_FILE|->",
		Short: "Redeem a one-time claim into client-bound credentials and endpoint registry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := readV3PairingClaim(cmd.Context(), cmd.InOrStdin(), args[0])
			if err != nil {
				return err
			}
			defer clear(payload)
			offer, err := remoteauth.ParsePairingClaimOfferForExchange(payload)
			if err != nil {
				return err
			}
			identity := endpointdomain.DaemonIdentity{DeviceID: offer.GetDeviceId(), DeviceFingerprint: remoteauth.Fingerprint(ed25519.PublicKey(offer.GetDevicePublicKey()))}
			pairingCandidate, err := remoteauth.PairingClaimEndpointCandidate(offer)
			if err != nil {
				return err
			}
			id := endpointdomain.EndpointID(strings.TrimSpace(endpointID))
			endpointLabel := strings.TrimSpace(label)
			if endpointLabel == "" {
				endpointLabel = string(id)
			}
			var endpoint endpointdomain.Endpoint
			var credential remoteauth.ClientAccessCredential
			_, err = updateV3ConnectionRegistry(cmd.Context(), registryPath, true, func(registry endpointdomain.Registry) (endpointdomain.Registry, error) {
				normalized, normalizeErr := registry.Normalize()
				if normalizeErr != nil {
					return endpointdomain.Registry{}, normalizeErr
				}
				actualID := id
				if existing, exists := normalized.Endpoints[id]; exists && !existing.DaemonIdentity.Empty() && existing.DaemonIdentity != identity {
					return endpointdomain.Registry{}, fmt.Errorf("endpoint %q is pinned to a different daemon identity", id)
				} else if !exists {
					for _, existing := range normalized.List() {
						if existing.DaemonIdentity == identity {
							actualID = existing.ID
							break
						}
					}
				}
				grantRef := v3PairingGrantRef(actualID, identity.DeviceID)
				credentials := remoteauth.NewCredentialStore(v3RemoteCredentialDir())
				var exchangedBundle []byte
				bound, bindErr := credentials.PairAndBind(
					cmd.Context(), grantRef, string(actualID), payload, nil, nil,
					remoteauth.BindGrantOptions{AllowScopeExpansion: allowScopeExpansion},
					func(clientIdentity remoteauth.ClientAccessIdentity) (remoteauth.PairingExchangeResult, error) {
						resolvedPairingSocket := strings.TrimSpace(pairingSocket)
						pairingRequest := remoteauth.ClientPairingRequest{
							ExpectedDeviceID: identity.DeviceID, ExpectedDeviceFingerprint: identity.DeviceFingerprint,
							PairingClaimOffer: payload, Identity: clientIdentity, ClientLabel: strings.TrimSpace(clientLabel), ClientProduct: uint32(cloudv1.ClientProduct_CLIENT_PRODUCT_CLI),
						}
						if pairingRequest.ClientLabel == "" {
							pairingRequest.ClientLabel = "anytty-cli"
							if hostname, hostnameErr := v3PairHostname(); hostnameErr == nil && strings.TrimSpace(hostname) != "" {
								pairingRequest.ClientLabel = "anytty-cli@" + strings.TrimSpace(hostname)
							}
						}
						signer, signerErr := remoteauth.NewPrivateClientAccessSigner(clientIdentity)
						if signerErr != nil {
							return remoteauth.PairingExchangeResult{}, signerErr
						}
						pairingRequest.Signer = signer
						if resolvedPairingSocket == "" {
							result, redeemErr := redeemV3RemotePairing(cmd.Context(), actualID, grantRef, pairingCandidate, pairingRequest)
							exchangedBundle = append([]byte(nil), result.Bundle...)
							return result, redeemErr
						}
						binding, bindingErr := remoteauth.LocalUnixChannelBinding(resolvedPairingSocket)
						if bindingErr != nil {
							return remoteauth.PairingExchangeResult{}, bindingErr
						}
						pairingTransport, dialErr := dialV3PairingTransport(cmd.Context(), resolvedPairingSocket)
						if dialErr != nil {
							return remoteauth.PairingExchangeResult{}, fmt.Errorf("connect PairingExchange socket: %w", dialErr)
						}
						defer pairingTransport.Close()
						pairingRequest.ChannelBinding = binding
						result, redeemErr := (remoteauth.ClientPairingHandshake{}).Redeem(cmd.Context(), pairingTransport, pairingRequest)
						exchangedBundle = append([]byte(nil), result.Bundle...)
						return result, redeemErr
					},
				)
				if bindErr != nil {
					return endpointdomain.Registry{}, bindErr
				}
				bundle, _, parseErr := remoteauth.ParsePairingBundleForExchange(exchangedBundle)
				if parseErr != nil {
					return endpointdomain.Registry{}, parseErr
				}
				updated, mergedEndpoint, mergedGrantRef, mergeErr := mergePairingEndpoint(normalized, actualID, endpointLabel, bundle)
				if mergeErr != nil || mergedGrantRef != grantRef {
					if mergeErr != nil {
						return endpointdomain.Registry{}, mergeErr
					}
					return endpointdomain.Registry{}, fmt.Errorf("pairing endpoint resolved a different credential reference")
				}
				endpoint = mergedEndpoint
				credential = bound
				return updated, nil
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Paired endpoint %s with daemon %s using client key %s\n", endpoint.ID, endpoint.DaemonIdentity.DeviceID, credential.Identity.Fingerprint)
			return nil
		},
	}
	command.Flags().StringVar(&endpointID, "id", "", "client-local endpoint id")
	command.Flags().StringVar(&label, "label", "", "override the daemon display label")
	command.Flags().StringVar(&registryPath, "registry", "", "endpoint registry path (default: XDG config dir endpoints.yaml)")
	command.Flags().StringVar(&pairingSocket, "pair-socket", "", "owner-only PairingExchange Unix socket (defaults to local daemon)")
	command.Flags().StringVar(&clientLabel, "client-label", "", "label recorded for this client access key")
	command.Flags().BoolVar(&allowScopeExpansion, "allow-scope-expansion", false, "confirm replacing an existing credential with a broader capability scope")
	_ = command.MarkFlagRequired("id")
	return command
}

func mergePairingEndpoint(
	registry endpointdomain.Registry,
	preferredID endpointdomain.EndpointID,
	label string,
	bundle *remoteauth.PairingBundle,
) (endpointdomain.Registry, endpointdomain.Endpoint, string, error) {
	candidate, err := endpointdomain.EndpointCandidateFromBootstrapBundle(bundle)
	if err != nil {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", err
	}
	identity := candidate.Identity
	if strings.TrimSpace(label) != "" {
		candidate.SuggestedLabel = strings.TrimSpace(label)
	}
	normalized, err := registry.Normalize()
	if err != nil {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", err
	}
	registry = normalized
	actualID := preferredID
	target, targetExists := registry.Endpoints[preferredID]
	if targetExists && !target.DaemonIdentity.Empty() && target.DaemonIdentity != identity {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", fmt.Errorf("endpoint %q is pinned to a different daemon identity", preferredID)
	}
	if !targetExists {
		for _, endpoint := range registry.List() {
			if endpoint.DaemonIdentity == identity {
				actualID = endpoint.ID
				target, targetExists = endpoint, true
				break
			}
		}
	}
	grantRef := v3PairingGrantRef(actualID, identity.DeviceID)
	if targetExists {
		for _, route := range target.Routes {
			if route.Kind != endpointdomain.RouteDirectWebRTCTCP && route.Kind != endpointdomain.RouteSSHWebRTCTCP && route.Kind != endpointdomain.RouteManagedWebRTC || strings.TrimSpace(route.CredentialRef) == "" {
				continue
			}
			if grantRef != v3PairingGrantRef(actualID, identity.DeviceID) && grantRef != route.CredentialRef {
				return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", fmt.Errorf("endpoint %q has conflicting capability credential refs", actualID)
			}
			grantRef = route.CredentialRef
		}
	}
	if !targetExists && len(candidate.Routes) == 0 {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", fmt.Errorf("pairing bundle contains no portable route and no existing endpoint matches daemon %q", identity.DeviceID)
	}
	for index := range candidate.Routes {
		if candidate.Routes[index].Kind == endpointdomain.RouteDirectWebRTCTCP || candidate.Routes[index].Kind == endpointdomain.RouteSSHWebRTCTCP || candidate.Routes[index].Kind == endpointdomain.RouteManagedWebRTC {
			candidate.Routes[index].CredentialRef = grantRef
		}
	}
	input := endpointdomain.EndpointAssemblerInput{Registry: registry, Candidates: []endpointdomain.EndpointCandidate{candidate}}
	if targetExists && target.DaemonIdentity.Empty() {
		input.ConfirmedIdentityBindings = []endpointdomain.ConfirmedIdentityBinding{{EndpointID: actualID, Identity: identity}}
	}
	result, err := endpointdomain.AssembleEndpoints(input)
	if err != nil {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", err
	}
	resolvedID := result.ResolvedEndpointIDs[0]
	if targetExists && resolvedID != actualID {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", fmt.Errorf("pairing identity resolved to endpoint %q instead of confirmed endpoint %q", resolvedID, actualID)
	}
	if !targetExists && resolvedID != actualID {
		endpoint := result.Registry.Endpoints[resolvedID]
		delete(result.Registry.Endpoints, resolvedID)
		endpoint.ID = actualID
		result.Registry.Endpoints[actualID] = endpoint
		if result.Registry.Default == resolvedID {
			result.Registry.Default = actualID
		}
		result.Registry, err = result.Registry.Normalize()
		if err != nil {
			return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", err
		}
		resolvedID = actualID
	}
	endpoint := result.Registry.Endpoints[resolvedID]
	authRoutes := 0
	for routeID, route := range endpoint.Routes {
		if route.Kind != endpointdomain.RouteDirectWebRTCTCP && route.Kind != endpointdomain.RouteSSHWebRTCTCP && route.Kind != endpointdomain.RouteManagedWebRTC {
			continue
		}
		if strings.TrimSpace(route.CredentialRef) != "" && route.CredentialRef != grantRef {
			return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", fmt.Errorf("endpoint %q route %q uses a different capability credential ref", resolvedID, routeID)
		}
		route.CredentialRef = grantRef
		endpoint.Routes[routeID] = route
		authRoutes++
	}
	if authRoutes == 0 {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", fmt.Errorf("endpoint %q has no WebRTC route that can use the paired capability", resolvedID)
	}
	result.Registry.Endpoints[resolvedID] = endpoint
	result.Registry, err = result.Registry.Normalize()
	if err != nil {
		return endpointdomain.Registry{}, endpointdomain.Endpoint{}, "", err
	}
	endpoint = result.Registry.Endpoints[resolvedID]
	return result.Registry, endpoint, grantRef, nil
}

func readV3PairingClaim(ctx context.Context, stdin io.Reader, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if decoded, inline, err := decodeInlineV3Pairing(strings.TrimSpace(path)); inline {
		return decoded, err
	}
	var reader io.Reader
	var file *os.File
	if strings.TrimSpace(path) == "-" {
		reader = stdin
	} else {
		var err error
		file, err = os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open pairing claim: %w", err)
		}
		defer file.Close()
		reader = file
	}
	payload, err := io.ReadAll(io.LimitReader(reader, endpointdomain.MaxPortableContractBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read pairing claim: %w", err)
	}
	if len(payload) == 0 || len(payload) > endpointdomain.MaxPortableContractBytes {
		clear(payload)
		return nil, fmt.Errorf("pairing claim size is invalid")
	}
	if decoded, inline, decodeErr := decodeInlineV3Pairing(strings.TrimSpace(string(payload))); inline {
		clear(payload)
		if decodeErr != nil {
			clear(decoded)
			return nil, decodeErr
		}
		payload = decoded
	}
	return payload, nil
}

func decodeInlineV3Pairing(text string) ([]byte, bool, error) {
	if strings.HasPrefix(text, remoteauth.PairingClaimCodePrefix) {
		decoded, err := remoteauth.DecodePairingClaimCode(text)
		if err != nil || len(decoded) == 0 || len(decoded) > endpointdomain.MaxPortableContractBytes {
			clear(decoded)
			return nil, true, fmt.Errorf("pairing claim code is invalid")
		}
		return decoded, true, nil
	}
	return nil, false, nil
}

func v3PairingGrantRef(endpointID endpointdomain.EndpointID, deviceID string) string {
	digest := sha256.Sum256([]byte(string(endpointID) + "\x00" + strings.TrimSpace(deviceID)))
	return "managed-" + hex.EncodeToString(digest[:12])
}

func writeV3PrivateFile(path string, payload []byte) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("output path is required")
	}
	parent := filepath.Dir(path)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return err
	}
	if err := securefs.SecureDirectory(parent); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".pairing-*.json")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := securefs.SecureFile(temporaryPath); err != nil {
		return err
	}
	if _, err := temporary.Write(payload); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := filepublish.Rename(temporaryPath, path); err != nil {
		return err
	}
	if err := filepublish.SyncDirectory(parent); err != nil {
		return err
	}
	committed = true
	return nil
}
