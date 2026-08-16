//go:build darwin || linux || windows

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	clouddaemon "github.com/anytty/anytty/cloud/daemon"
	cloudprotocol "github.com/anytty/anytty/cloud/protocol"
	"github.com/anytty/anytty/cloud/ticket"
	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
	"github.com/anytty/anytty/shared/remoteauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestCloudEdgeListRealDaemonE2E(t *testing.T) {
	binary := buildAnyTTYBinaryForTest(t)
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))

	identity, err := remoteauth.LoadOrCreateLocalIdentity(v3RemoteIdentityDir())
	if err != nil {
		t.Fatal(err)
	}
	const daemonID = "11111111-1111-4111-8111-111111111111"
	const accountID = "22222222-2222-4222-8222-222222222222"
	edgeLocator := startCloudEdgeListE2EEdge(t, daemonID, "edge-e2e-1", "E2E Edge")
	controller := startCloudEdgeListE2EController(t, &cloudEdgeListE2EEnrollmentService{
		identity: identity, daemonID: daemonID, accountID: accountID, locator: edgeLocator,
		challenge: bytes.Repeat([]byte{0x71}, remoteauth.DeviceIdentityChallengeBytes),
	})
	controllerCA := filepath.Join(root, "controller-ca.pem")
	if err := os.WriteFile(controllerCA, controller.caPEM, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_ADDRESS", controller.address)
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_SERVER_NAME", controller.serverName)
	t.Setenv("ANYTTY_CLOUD_CONTROLLER_CA", controllerCA)

	record := cloudEdgeListE2ERecord(t, daemonID, accountID, identity, edgeLocator)
	if err := clouddaemon.SaveRecord(v3CloudEnrollmentRecordPath(), record); err != nil {
		t.Fatal(err)
	}

	socketPath := filepath.Join(root, "anytty.sock")
	logPath := filepath.Join(root, "anytty.log")
	t.Cleanup(func() {
		stop := exec.Command(binary, "--socket", socketPath, "--log-file", logPath, "daemon", "stop")
		stop.Env = os.Environ()
		_, _ = stop.CombinedOutput()
	})

	_ = executeCloudEdgeE2EBinary(t, binary, logPath, "--socket", socketPath, "--log-file", logPath, "daemon", "start", "--json")
	output := executeCloudEdgeE2EBinary(t, binary, logPath, "--socket", socketPath, "--log-file", logPath, "cloud", "edge", "list")
	if !strings.Contains(output, "E2E Edge") || !strings.Contains(output, "edge-e2e-1") {
		t.Fatalf("cloud edge list output did not include measured Edge:\n%s", output)
	}
	if strings.Contains(output, "server closed the stream") {
		t.Fatalf("cloud edge list leaked a broken gRPC stream error:\n%s", output)
	}
}

func executeCloudEdgeE2EBinary(t *testing.T, binary, logPath string, args ...string) string {
	t.Helper()
	command := exec.Command(binary, args...)
	command.Env = os.Environ()
	output, err := command.CombinedOutput()
	if err != nil {
		logs, _ := os.ReadFile(logPath)
		t.Fatalf("anytty %s: %v\noutput:\n%s\nlog:\n%s", strings.Join(args, " "), err, output, logs)
	}
	return string(output)
}

type cloudEdgeListE2EEnrollmentService struct {
	cloudv1.UnimplementedEnrollmentServiceServer
	identity             remoteauth.Identity
	challenge            []byte
	daemonID, accountID  string
	locator              *cloudv1.EdgeLocator
	measurementRefreshes atomic.Int32
}

func (service *cloudEdgeListE2EEnrollmentService) BeginDaemonBindingRefresh(context.Context, *cloudv1.BeginDaemonBindingRefreshRequest) (*cloudv1.IdentityChallenge, error) {
	return &cloudv1.IdentityChallenge{
		ChallengeId: "edge-list-e2e-refresh", Challenge: append([]byte(nil), service.challenge...),
		ExpiresAt: timestamppb.New(time.Now().Add(time.Minute)),
	}, nil
}

func (service *cloudEdgeListE2EEnrollmentService) CompleteDaemonBindingRefresh(_ context.Context, request *cloudv1.CompleteDaemonBindingRefreshRequest) (*cloudv1.RefreshDaemonBindingResponse, error) {
	if request.GetChallengeId() != "edge-list-e2e-refresh" ||
		remoteauth.VerifyDeviceIdentityProof(service.challenge, service.identity.DeviceID, service.identity.Fingerprint, service.identity.PublicKey, request.GetDeviceProof()) != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid DeviceIdentity proof")
	}
	for _, measurement := range request.GetEdgeMeasurements() {
		if measurement.GetEdgeId() == service.locator.GetEdgeId() && measurement.GetReachable() {
			service.measurementRefreshes.Add(1)
		}
	}
	binding, err := cloudEdgeListE2EBinding(service.daemonID, service.accountID, service.identity, service.locator)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	selection := &cloudv1.DaemonEdgeSelection{
		DaemonId: service.daemonID, CurrentEdgeId: service.locator.GetEdgeId(), SelectedEdgeId: service.locator.GetEdgeId(),
		PreferenceRevision: 1, EvaluatedAt: timestamppb.Now(),
		Candidates: []*cloudv1.DaemonEdgeCandidate{{
			Locator: proto.Clone(service.locator).(*cloudv1.EdgeLocator), Online: true, Eligible: true, Current: true,
			AgentCount: 1, Capacity: 32, Status: "可用",
		}},
	}
	return &cloudv1.RefreshDaemonBindingResponse{
		Daemon: &cloudv1.DaemonRecord{
			DaemonId: service.daemonID, AccountId: service.accountID, DeviceId: service.identity.DeviceID, DeviceFingerprint: service.identity.Fingerprint,
			State: cloudv1.DaemonState_DAEMON_STATE_ACTIVE, StateRevision: 1,
		},
		DaemonBinding: binding, EdgeLocator: proto.Clone(service.locator).(*cloudv1.EdgeLocator), EdgeSelection: selection,
	}, nil
}

type cloudEdgeListE2EAgentGateway struct {
	cloudv1.UnimplementedAgentGatewayServer
	daemonID, edgeID string
}

func (gateway *cloudEdgeListE2EAgentGateway) Connect(stream cloudv1.AgentGateway_ConnectServer) error {
	now := time.Now().UTC().Add(-time.Second)
	challenge := &cloudv1.EdgeChallenge{
		Nonce: bytes.Repeat([]byte{0x72}, ticket.EdgeChallengeNonceSize), EdgeId: gateway.edgeID, EdgeBootId: "edge-list-e2e-boot", StreamId: "edge-list-e2e-stream",
		IssuedAt: timestamppb.New(now), ExpiresAt: timestamppb.New(now.Add(ticket.EdgeChallengeLifetime)), Target: cloudv1.EdgeChallengeTarget_EDGE_CHALLENGE_TARGET_AGENT_GATEWAY,
	}
	if err := stream.Send(&cloudv1.EdgeCommand{
		ProtocolVersion: cloudprotocol.AgentGatewayVersion, MessageId: "challenge", SenderId: gateway.edgeID, BootId: challenge.GetEdgeBootId(), ConnectionId: challenge.GetStreamId(),
		StreamSeq: 1, SentAt: challenge.GetIssuedAt(), Payload: &cloudv1.EdgeCommand_Challenge{Challenge: challenge},
	}); err != nil {
		return err
	}
	hello, err := stream.Recv()
	if err != nil {
		return err
	}
	if err := stream.Send(&cloudv1.EdgeCommand{
		ProtocolVersion: cloudprotocol.AgentGatewayVersion, MessageId: "ready", SenderId: gateway.edgeID, BootId: challenge.GetEdgeBootId(), ConnectionId: hello.GetConnectionId(),
		StreamSeq: 2, SentAt: timestamppb.Now(), Payload: &cloudv1.EdgeCommand_Ready{Ready: &cloudv1.AgentReady{
			Generation: 1, Heartbeat: &cloudv1.HeartbeatPolicy{Interval: durationpb.New(time.Hour), Timeout: durationpb.New(2 * time.Hour)},
			DaemonState: &cloudv1.DaemonStateRecord{DaemonId: gateway.daemonID, State: cloudv1.DaemonState_DAEMON_STATE_ACTIVE, StateRevision: 1},
		}},
	}); err != nil {
		return err
	}
	for {
		if _, err := stream.Recv(); err != nil {
			return err
		}
	}
}

type cloudEdgeListE2EEndpoint struct {
	address, serverName string
	caPEM               []byte
}

func startCloudEdgeListE2EController(t *testing.T, service cloudv1.EnrollmentServiceServer) cloudEdgeListE2EEndpoint {
	return startCloudEdgeListE2ETLSServer(t, "controller-edge-list-e2e.test", func(server *grpc.Server) {
		cloudv1.RegisterEnrollmentServiceServer(server, service)
	})
}

func startCloudEdgeListE2EEdge(t *testing.T, daemonID, edgeID, edgeName string) *cloudv1.EdgeLocator {
	endpoint := startCloudEdgeListE2ETLSServer(t, "edge-list-e2e.test", func(server *grpc.Server) {
		cloudv1.RegisterAgentGatewayServer(server, &cloudEdgeListE2EAgentGateway{daemonID: daemonID, edgeID: edgeID})
		healthServer := health.NewServer()
		healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
		grpc_health_v1.RegisterHealthServer(server, healthServer)
	})
	return &cloudv1.EdgeLocator{
		EdgeId: edgeID, Name: edgeName, Region: "E2E", PublicEndpoint: endpoint.address, ServerName: endpoint.serverName,
		CaCertificatePem: append([]byte(nil), endpoint.caPEM...),
	}
}

func startCloudEdgeListE2ETLSServer(t *testing.T, serverName string, register func(*grpc.Server)) cloudEdgeListE2EEndpoint {
	t.Helper()
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: serverName + " CA"},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true,
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	rootDER, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	root, err := x509.ParseCertificate(rootDER)
	if err != nil {
		t.Fatal(err)
	}
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: serverName}, DNSNames: []string{serverName},
		NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, root, &leafKey.PublicKey, rootKey)
	if err != nil {
		t.Fatal(err)
	}
	leafKeyDER, err := x509.MarshalPKCS8PrivateKey(leafKey)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := tls.X509KeyPair(
		pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDER}),
		pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: leafKeyDER}),
	)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer(grpc.Creds(credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS13, Certificates: []tls.Certificate{certificate},
	})))
	register(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})
	return cloudEdgeListE2EEndpoint{
		address: listener.Addr().String(), serverName: serverName,
		caPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootDER}),
	}
}

func cloudEdgeListE2ERecord(t *testing.T, daemonID, accountID string, identity remoteauth.Identity, locator *cloudv1.EdgeLocator) clouddaemon.EnrollmentRecord {
	t.Helper()
	binding, err := cloudEdgeListE2EBinding(daemonID, accountID, identity, locator)
	if err != nil {
		t.Fatal(err)
	}
	bindingPayload, err := proto.MarshalOptions{Deterministic: true}.Marshal(binding)
	if err != nil {
		t.Fatal(err)
	}
	locatorPayload, err := proto.MarshalOptions{Deterministic: true}.Marshal(locator)
	if err != nil {
		t.Fatal(err)
	}
	return clouddaemon.EnrollmentRecord{
		Version: 2, DaemonID: daemonID, AccountID: accountID, DaemonBinding: bindingPayload, EdgeLocator: locatorPayload, EnrolledAt: time.Now().UTC(),
	}
}

func cloudEdgeListE2EBinding(daemonID, accountID string, identity remoteauth.Identity, locator *cloudv1.EdgeLocator) (*cloudv1.SignedEnvelope, error) {
	locatorPayload, err := proto.MarshalOptions{Deterministic: true}.Marshal(locator)
	if err != nil {
		return nil, err
	}
	digest := sha256.Sum256(locatorPayload)
	claims, err := proto.MarshalOptions{Deterministic: true}.Marshal(&cloudv1.DaemonBindingClaims{
		BindingId: "edge-list-e2e-binding", DaemonId: daemonID, AccountId: accountID, EdgeId: locator.GetEdgeId(),
		DeviceId: identity.DeviceID, DevicePublicKey: append([]byte(nil), identity.PublicKey...), EdgeLocatorSha256: digest[:],
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(daemonID) == "" || strings.TrimSpace(accountID) == "" || locator.GetEdgeId() == "" {
		return nil, errors.New("incomplete binding test fixture")
	}
	return &cloudv1.SignedEnvelope{KeyId: identity.Fingerprint, Payload: claims, Signature: []byte(fmt.Sprintf("test:%s", locator.GetEdgeId()))}, nil
}
