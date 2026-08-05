package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	shareadapter "github.com/anytty/anytty/client/adapter/share"
	systemadapter "github.com/anytty/anytty/client/adapter/system"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var endpointShareLANAddresses = systemadapter.PrivateLANIPv4Addresses

func newEndpointShareCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var listenAddress string
	var advertisedAddresses []string
	var ttl time.Duration
	var raw bool
	command := &cobra.Command{
		Use: "share ID", Short: "Share portable Endpoint routes and policy through a one-time TLS session", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			registry, err := runtime.load()
			if err != nil {
				return err
			}
			target, ok := registry.Endpoints[endpointdomain.EndpointID(strings.TrimSpace(args[0]))]
			if !ok {
				return &cliError{code: 3, message: fmt.Sprintf("endpoint %s was not found", args[0])}
			}
			listener, err := net.Listen("tcp", strings.TrimSpace(listenAddress))
			if err != nil {
				return fmt.Errorf("listen endpoint share: %w", err)
			}
			addresses, err := endpointShareAdvertisedAddresses(listener.Addr().String(), advertisedAddresses)
			if err != nil {
				_ = listener.Close()
				return usageCLIError(err.Error())
			}
			transferID, err := endpointShareRandomID()
			if err != nil {
				_ = listener.Close()
				return err
			}
			bundle, err := endpointdomain.NewClientEndpointShareBundle(target, transferID, time.Now(), ttl)
			if err != nil {
				_ = listener.Close()
				return err
			}
			server, err := shareadapter.NewServer(shareadapter.ServerOptions{Listener: listener, AdvertisedAddresses: addresses, Bundle: bundle, TTL: ttl})
			if err != nil {
				_ = listener.Close()
				return err
			}
			defer server.Close()
			uri, err := shareadapter.EncodeOfferURI(server.Offer())
			if err != nil {
				return err
			}
			if raw {
				fmt.Fprintln(cmd.OutOrStdout(), uri)
			} else if err := renderEndpointShareQR(cmd.OutOrStdout(), uri, target, addresses, time.Now().Add(ttl)); err != nil {
				return err
			}
			if err := server.Serve(cmd.Context()); err != nil {
				return fmt.Errorf("serve endpoint share: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Endpoint share completed")
			return nil
		},
	}
	command.Flags().StringVar(&listenAddress, "listen", "0.0.0.0:41130", "one-time TLS listener address")
	command.Flags().StringArrayVar(&advertisedAddresses, "address", nil, "receiver-reachable TLS HOST:PORT (repeatable)")
	command.Flags().DurationVar(&ttl, "ttl", 10*time.Minute, "share session lifetime")
	command.Flags().BoolVar(&raw, "raw", false, "print the share URI instead of a QR code")
	command.AddCommand(newEndpointShareReceiveCommand(runtime))
	return command
}

func newEndpointShareReceiveCommand(runtime *endpointCommandRuntime) *cobra.Command {
	var yes bool
	command := &cobra.Command{
		Use: "receive OFFER", Short: "Receive, preview, and atomically import an Endpoint share", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			offer, err := shareadapter.DecodeOfferURI(args[0])
			if err != nil {
				return usageCLIError(err.Error())
			}
			bundle, err := shareadapter.Receive(cmd.Context(), offer)
			if err != nil {
				return err
			}
			candidate, err := endpointdomain.EndpointCandidateFromShareBundle(bundle)
			if err != nil {
				return err
			}
			registry, err := runtime.load()
			if err != nil {
				return err
			}
			diff, err := endpointdomain.PreviewShare(registry, candidate)
			if err != nil {
				return err
			}
			if err := renderEndpointShareDiff(cmd.OutOrStdout(), diff); err != nil {
				return err
			}
			if !yes {
				confirmed, err := confirmEndpointShare(cmd)
				if err != nil {
					return err
				}
				if !confirmed {
					return fmt.Errorf("endpoint share import cancelled")
				}
			}
			var importedID endpointdomain.EndpointID
			if err := runtime.update(cmd.Context(), true, func(current endpointdomain.Registry) (endpointdomain.Registry, error) {
				assembled, err := endpointdomain.AssembleEndpoints(endpointdomain.EndpointAssemblerInput{Registry: current, Candidates: []endpointdomain.EndpointCandidate{candidate}})
				if err != nil {
					return endpointdomain.Registry{}, err
				}
				importedID = assembled.ResolvedEndpointIDs[0]
				return assembled.Registry, nil
			}); err != nil {
				return err
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Endpoint", Value: string(importedID)},
				cliField{Label: "Status", Value: "imported"},
				cliField{Label: "Access", Value: "config only"},
			)
		},
	}
	command.Flags().BoolVar(&yes, "yes", false, "confirm the displayed Route and policy diff")
	return command
}

func endpointShareAdvertisedAddresses(bound string, overrides []string) ([]string, error) {
	if len(overrides) > 0 {
		return normalizeDirectPairingAddresses(overrides)
	}
	host, port, err := net.SplitHostPort(bound)
	if err != nil {
		return nil, err
	}
	if !isWildcardHost(host) {
		return normalizeDirectPairingAddresses([]string{bound})
	}
	hosts, err := endpointShareLANAddresses()
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no private LAN address is available; provide --address")
	}
	addresses := make([]string, 0, len(hosts))
	for _, value := range hosts {
		addresses = append(addresses, net.JoinHostPort(value, port))
	}
	return normalizeDirectPairingAddresses(addresses)
}

func endpointShareRandomID() (string, error) {
	value := make([]byte, 18)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate endpoint share transfer id: %w", err)
	}
	return "share-" + base64.RawURLEncoding.EncodeToString(value), nil
}

func renderEndpointShareQR(output io.Writer, uri string, target endpointdomain.Endpoint, addresses []string, expiresAt time.Time) error {
	if err := writeCLIFields(output,
		cliField{Label: "Endpoint", Value: target.Label},
		cliField{Label: "Device", Value: target.DaemonIdentity.DeviceID},
		cliField{Label: "Fingerprint", Value: target.DaemonIdentity.DeviceFingerprint},
		cliField{Label: "Addresses", Value: strings.Join(addresses, ", ")},
		cliField{Label: "Expires", Value: formatTerminalTime(expiresAt.UTC())},
	); err != nil {
		return err
	}
	code, err := qrcode.New(uri, qrcode.Medium)
	if err != nil {
		return err
	}
	bitmap := code.Bitmap()
	for row := 0; row < len(bitmap); row += 2 {
		for column := range bitmap[row] {
			top := bitmap[row][column]
			bottom := row+1 < len(bitmap) && bitmap[row+1][column]
			switch {
			case top && bottom:
				_, _ = io.WriteString(output, "█")
			case top:
				_, _ = io.WriteString(output, "▀")
			case bottom:
				_, _ = io.WriteString(output, "▄")
			default:
				_, _ = io.WriteString(output, " ")
			}
		}
		_, _ = io.WriteString(output, "\n")
	}
	return nil
}

func renderEndpointShareDiff(output io.Writer, diff endpointdomain.ShareDiff) error {
	connectMode := "unchanged"
	if diff.ConnectModeChanged {
		connectMode = "changed"
	}
	selectionPolicy := "unchanged"
	if diff.SelectionPolicyChanged {
		selectionPolicy = "changed"
	}
	if err := writeCLIFields(output,
		cliField{Label: "Endpoint", Value: string(diff.EndpointID)},
		cliField{Label: "Device", Value: diff.Identity.DeviceID},
		cliField{Label: "Fingerprint", Value: diff.Identity.DeviceFingerprint},
		cliField{Label: "Connect mode", Value: connectMode},
		cliField{Label: "Selection policy", Value: selectionPolicy},
		cliField{Label: "Authorization", Value: "required after config import"},
	); err != nil {
		return err
	}
	if _, err := io.WriteString(output, "\nRoutes\n"); err != nil {
		return err
	}
	rows := make([][]string, 0, len(diff.Routes))
	for _, route := range diff.Routes {
		rows = append(rows, []string{string(route.RouteID), string(route.Kind), string(route.Action)})
	}
	return writeCLITable(output, []string{"ID", "KIND", "ACTION"}, rows)
}

func confirmEndpointShare(cmd *cobra.Command) (bool, error) {
	input, ok := cmd.InOrStdin().(*os.File)
	if !ok || !term.IsTerminal(int(input.Fd())) {
		return false, fmt.Errorf("endpoint share import requires --yes when stdin is not interactive")
	}
	fmt.Fprint(cmd.OutOrStdout(), "Import this config-only Endpoint? [y/N] ")
	var answer string
	if _, err := fmt.Fscanln(input, &answer); err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(answer), "y") || strings.EqualFold(strings.TrimSpace(answer), "yes"), nil
}
