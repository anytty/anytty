// Package share owns the one-time TLS transport used to migrate portable Endpoint configuration between clients.
package share

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/proto/remoteauthpb"
	"google.golang.org/protobuf/proto"
)

const (
	receiverProofProtocol = "anytty.endpoint-share.receiver-proof"
	receiverProofVersion  = uint32(1)
	shareMessageVersion   = uint32(1)
	maxFrameBytes         = endpoint.MaxPortableContractBytes
	shareURIPrefix        = "anytty://share?payload="
)

// ServerOptions 定义一次性 share listener 的公开 locator 与 config-only payload。
// AdvertisedAddresses 必须是接收端可达地址；Listener 的实际 bind 地址不进入身份或授权真值。
type ServerOptions struct {
	Listener            net.Listener
	AdvertisedAddresses []string
	Bundle              *remoteauthpb.ClientEndpointShareBundleV1
	TTL                 time.Duration
	Now                 func() time.Time
}

// Server 持有一次性 TLS share session。
// 成功 receiver proof 只允许消费一次；Close 或过期后 listener、secret 和 bundle 都不得复用。
type Server struct {
	listener  net.Listener
	offer     *remoteauthpb.ShareSessionOffer
	bundle    *remoteauthpb.ClientEndpointShareBundleV1
	now       func() time.Time
	consumed  atomic.Bool
	closeOnce sync.Once
	closeErr  error
}

// NewServer 创建临时 TLS 1.3 listener 和可编码进二维码的 ShareSessionOffer。
func NewServer(options ServerOptions) (*Server, error) {
	if options.Listener == nil || options.Bundle == nil || options.TTL <= 0 || len(options.AdvertisedAddresses) == 0 {
		return nil, fmt.Errorf("share server options are incomplete")
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	if _, err := endpoint.MarshalClientEndpointShareBundle(options.Bundle); err != nil {
		return nil, err
	}
	certificate, pin, err := ephemeralCertificate(options.Now().UTC(), options.TTL)
	if err != nil {
		return nil, err
	}
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return nil, fmt.Errorf("generate share session secret: %w", err)
	}
	tlsListener := tls.NewListener(options.Listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS13})
	offer := &remoteauthpb.ShareSessionOffer{
		SchemaVersion: endpoint.ShareSessionOfferVersion, TransferId: options.Bundle.GetTransferId(),
		ListenerAddresses: append([]string(nil), options.AdvertisedAddresses...), EphemeralCertificateSha256: pin,
		OneTimeSessionSecret: secret, ExpiresAtUnixNano: options.Now().UTC().Add(options.TTL).UnixNano(),
	}
	if _, err := endpoint.MarshalShareSessionOffer(offer); err != nil {
		_ = tlsListener.Close()
		return nil, err
	}
	return &Server{listener: tlsListener, offer: offer, bundle: proto.Clone(options.Bundle).(*remoteauthpb.ClientEndpointShareBundleV1), now: options.Now}, nil
}

// Offer 返回独立的静态二维码 offer；调用方不得把其中 secret 写入日志。
func (server *Server) Offer() *remoteauthpb.ShareSessionOffer {
	if server == nil || server.offer == nil {
		return nil
	}
	return proto.Clone(server.offer).(*remoteauthpb.ShareSessionOffer)
}

// Serve 等待第一个完成 receiver proof 的接收端并发送 bundle。
// 无效连接不会消费 session；成功发送、context 取消、过期或 listener 关闭都会结束本方法。
func (server *Server) Serve(ctx context.Context) error {
	if server == nil || server.listener == nil {
		return fmt.Errorf("share server is closed")
	}
	defer server.Close()
	for {
		if server.now().UnixNano() >= server.offer.GetExpiresAtUnixNano() {
			return fmt.Errorf("share session expired")
		}
		connection, err := server.acceptContext(ctx)
		if err != nil {
			return err
		}
		accepted, sessionErr := server.handle(connection)
		_ = connection.Close()
		if accepted {
			return sessionErr
		}
		if sessionErr != nil && ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

// Close 关闭一次性 listener；方法幂等。
func (server *Server) Close() error {
	if server == nil || server.listener == nil {
		return nil
	}
	server.closeOnce.Do(func() { server.closeErr = server.listener.Close() })
	return server.closeErr
}

func (server *Server) handle(connection net.Conn) (bool, error) {
	deadline := time.Unix(0, server.offer.GetExpiresAtUnixNano())
	_ = connection.SetDeadline(deadline)
	clientEnvelope := &remoteauthpb.ShareSessionClientEnvelope{}
	if err := readFrame(connection, clientEnvelope); err != nil {
		return false, err
	}
	hello := clientEnvelope.GetHello()
	if hello == nil || hello.GetSchemaVersion() != shareMessageVersion || hello.GetTransferId() != server.offer.GetTransferId() ||
		len(hello.GetReceiverPublicKey()) != ed25519.PublicKeySize || len(hello.GetReceiverNonce()) != 32 ||
		!equalBytes(hello.GetOneTimeSessionSecret(), server.offer.GetOneTimeSessionSecret()) {
		_ = writeServerError(connection, "admission_denied", "share receiver admission failed")
		return false, fmt.Errorf("share receiver admission failed")
	}
	senderNonce := make([]byte, 32)
	if _, err := rand.Read(senderNonce); err != nil {
		return false, err
	}
	challenge := &remoteauthpb.ShareSenderChallenge{
		SchemaVersion: shareMessageVersion, TransferId: server.offer.GetTransferId(), ReceiverNonce: append([]byte(nil), hello.GetReceiverNonce()...),
		SenderNonce: senderNonce, ExpiresAtUnixNano: server.offer.GetExpiresAtUnixNano(),
	}
	if err := writeFrame(connection, &remoteauthpb.ShareSessionServerEnvelope{Message: &remoteauthpb.ShareSessionServerEnvelope_Challenge{Challenge: challenge}}); err != nil {
		return false, err
	}
	clientEnvelope.Reset()
	if err := readFrame(connection, clientEnvelope); err != nil {
		return false, err
	}
	proof := clientEnvelope.GetProof()
	input, err := receiverProofInput(server.offer, hello.GetReceiverNonce(), senderNonce)
	if err != nil || proof == nil || proof.GetSchemaVersion() != shareMessageVersion || proof.GetTransferId() != server.offer.GetTransferId() ||
		!ed25519.Verify(ed25519.PublicKey(hello.GetReceiverPublicKey()), input, proof.GetSignature()) {
		_ = writeServerError(connection, "proof_invalid", "share receiver proof is invalid")
		return false, fmt.Errorf("share receiver proof is invalid")
	}
	if !server.consumed.CompareAndSwap(false, true) {
		_ = writeServerError(connection, "already_consumed", "share session was already consumed")
		return false, fmt.Errorf("share session was already consumed")
	}
	if err := writeFrame(connection, &remoteauthpb.ShareSessionServerEnvelope{Message: &remoteauthpb.ShareSessionServerEnvelope_Bundle{Bundle: server.bundle}}); err != nil {
		return true, err
	}
	return true, nil
}

// Receive 连接 offer 中任一 locator，验证临时 TLS certificate pin，并完成 receiver-proof 后返回 config-only bundle。
func Receive(ctx context.Context, offer *remoteauthpb.ShareSessionOffer) (*remoteauthpb.ClientEndpointShareBundleV1, error) {
	payload, err := endpoint.MarshalShareSessionOffer(offer)
	if err != nil {
		return nil, err
	}
	offer, err = endpoint.ParseShareSessionOffer(payload)
	if err != nil {
		return nil, err
	}
	var failures []string
	for _, address := range offer.GetListenerAddresses() {
		bundle, receiveErr := receiveAddress(ctx, address, offer)
		if receiveErr == nil {
			return bundle, nil
		}
		failures = append(failures, receiveErr.Error())
	}
	return nil, fmt.Errorf("receive endpoint share: %s", strings.Join(failures, "; "))
}

func receiveAddress(ctx context.Context, address string, offer *remoteauthpb.ShareSessionOffer) (*remoteauthpb.ClientEndpointShareBundleV1, error) {
	config := &tls.Config{
		MinVersion: tls.VersionTLS13, InsecureSkipVerify: true,
		VerifyConnection: func(state tls.ConnectionState) error {
			if len(state.PeerCertificates) != 1 || certificatePin(state.PeerCertificates[0].Raw) != offer.GetEphemeralCertificateSha256() {
				return fmt.Errorf("share TLS certificate pin mismatch")
			}
			return nil
		},
	}
	dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 5 * time.Second}, Config: config}
	connection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	receiverNonce := make([]byte, 32)
	if _, err := rand.Read(receiverNonce); err != nil {
		return nil, err
	}
	hello := &remoteauthpb.ShareReceiverHello{
		SchemaVersion: shareMessageVersion, TransferId: offer.GetTransferId(), OneTimeSessionSecret: append([]byte(nil), offer.GetOneTimeSessionSecret()...),
		ReceiverPublicKey: publicKey, ReceiverNonce: receiverNonce,
	}
	if err := writeFrame(connection, &remoteauthpb.ShareSessionClientEnvelope{Message: &remoteauthpb.ShareSessionClientEnvelope_Hello{Hello: hello}}); err != nil {
		return nil, err
	}
	serverEnvelope := &remoteauthpb.ShareSessionServerEnvelope{}
	if err := readFrame(connection, serverEnvelope); err != nil {
		return nil, err
	}
	challenge := serverEnvelope.GetChallenge()
	if challenge == nil || challenge.GetSchemaVersion() != shareMessageVersion || challenge.GetTransferId() != offer.GetTransferId() ||
		!equalBytes(challenge.GetReceiverNonce(), receiverNonce) || challenge.GetExpiresAtUnixNano() != offer.GetExpiresAtUnixNano() || len(challenge.GetSenderNonce()) != 32 {
		return nil, serverEnvelopeError(serverEnvelope, "share sender challenge is invalid")
	}
	input, err := receiverProofInput(offer, receiverNonce, challenge.GetSenderNonce())
	if err != nil {
		return nil, err
	}
	proof := &remoteauthpb.ShareReceiverProof{SchemaVersion: shareMessageVersion, TransferId: offer.GetTransferId(), Signature: ed25519.Sign(privateKey, input)}
	if err := writeFrame(connection, &remoteauthpb.ShareSessionClientEnvelope{Message: &remoteauthpb.ShareSessionClientEnvelope_Proof{Proof: proof}}); err != nil {
		return nil, err
	}
	serverEnvelope.Reset()
	if err := readFrame(connection, serverEnvelope); err != nil {
		return nil, err
	}
	if serverEnvelope.GetBundle() == nil {
		return nil, serverEnvelopeError(serverEnvelope, "share server returned no bundle")
	}
	bundlePayload, err := endpoint.MarshalClientEndpointShareBundle(serverEnvelope.GetBundle())
	if err != nil {
		return nil, err
	}
	bundle, err := endpoint.ParseClientEndpointShareBundle(bundlePayload)
	if err != nil {
		return nil, err
	}
	if bundle.GetTransferId() != offer.GetTransferId() {
		return nil, fmt.Errorf("share bundle transfer_id mismatch")
	}
	return bundle, nil
}

// EncodeOfferURI 编码只包含 ShareSessionOffer 的二维码 URI。
func EncodeOfferURI(offer *remoteauthpb.ShareSessionOffer) (string, error) {
	payload, err := endpoint.MarshalShareSessionOffer(offer)
	if err != nil {
		return "", err
	}
	return shareURIPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecodeOfferURI 严格解析二维码 URI 或裸 base64url offer。
func DecodeOfferURI(value string) (*remoteauthpb.ShareSessionOffer, error) {
	encoded := strings.TrimSpace(value)
	if strings.HasPrefix(encoded, shareURIPrefix) {
		encoded = strings.TrimPrefix(encoded, shareURIPrefix)
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("share offer payload is invalid")
	}
	return endpoint.ParseShareSessionOffer(payload)
}

func receiverProofInput(offer *remoteauthpb.ShareSessionOffer, receiverNonce, senderNonce []byte) ([]byte, error) {
	secretHash := sha256.Sum256(offer.GetOneTimeSessionSecret())
	input := &remoteauthpb.ShareReceiverProofInput{
		Protocol: receiverProofProtocol, Version: receiverProofVersion, TransferId: offer.GetTransferId(),
		ReceiverNonce: receiverNonce, SenderNonce: senderNonce, EphemeralCertificateSha256: offer.GetEphemeralCertificateSha256(),
		OneTimeSessionSecretSha256: secretHash[:],
	}
	return (proto.MarshalOptions{Deterministic: true}).Marshal(input)
}

func ephemeralCertificate(now time.Time, ttl time.Duration) (tls.Certificate, string, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	template := &x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "AnyTTY Endpoint Share"},
		NotBefore: now.Add(-time.Minute), NotAfter: now.Add(ttl), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, "", err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: privateKey}, certificatePin(der), nil
}

func certificatePin(der []byte) string {
	digest := sha256.Sum256(der)
	return "sha256:" + base64.RawURLEncoding.EncodeToString(digest[:])
}

func (server *Server) acceptContext(ctx context.Context) (net.Conn, error) {
	type result struct {
		connection net.Conn
		err        error
	}
	ready := make(chan result, 1)
	go func() {
		connection, err := server.listener.Accept()
		ready <- result{connection: connection, err: err}
	}()
	select {
	case <-ctx.Done():
		_ = server.Close()
		return nil, ctx.Err()
	case value := <-ready:
		return value.connection, value.err
	}
}

func writeFrame(writer io.Writer, message proto.Message) error {
	payload, err := (proto.MarshalOptions{Deterministic: true}).Marshal(message)
	if err != nil {
		return err
	}
	if len(payload) == 0 || len(payload) > maxFrameBytes {
		return fmt.Errorf("share frame size is invalid")
	}
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(payload)))
	if err := writeAll(writer, header); err != nil {
		return err
	}
	return writeAll(writer, payload)
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written <= 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

func readFrame(reader io.Reader, message proto.Message) error {
	header := make([]byte, 4)
	if _, err := io.ReadFull(reader, header); err != nil {
		return err
	}
	size := binary.BigEndian.Uint32(header)
	if size == 0 || size > maxFrameBytes {
		return fmt.Errorf("share frame size is invalid")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return err
	}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, message); err != nil {
		return err
	}
	if len(message.ProtoReflect().GetUnknown()) != 0 {
		return fmt.Errorf("share frame contains unknown fields")
	}
	return nil
}

func writeServerError(writer io.Writer, code, message string) error {
	return writeFrame(writer, &remoteauthpb.ShareSessionServerEnvelope{Message: &remoteauthpb.ShareSessionServerEnvelope_Error{Error: &remoteauthpb.ShareSessionError{Code: code, Message: message}}})
}

func serverEnvelopeError(envelope *remoteauthpb.ShareSessionServerEnvelope, fallback string) error {
	if value := envelope.GetError(); value != nil && value.GetMessage() != "" {
		return fmt.Errorf("%s: %s", value.GetCode(), value.GetMessage())
	}
	return fmt.Errorf("%s", fallback)
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range left {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}
