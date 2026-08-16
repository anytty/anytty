package enginehost

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	"github.com/anytty/anytty/client/port"
	"github.com/anytty/anytty/proto/bindingpb"
	golangssh "golang.org/x/crypto/ssh"
)

const platformSSHCredentialPrefix = "ssh-platform-"

// ProvisionSSHCredential 为 Go registry 中的 SSH Route 创建或复用平台不可导出 signer。
// 只有平台 key 与 opaque registry 两步都成功后才发布新 snapshot；新 key 的失败事务会显式回滚。
func (host *Host) ProvisionSSHCredential(ctx context.Context, request *bindingpb.SSHCredentialProvisionRequest) (*bindingpb.SSHCredentialProvisionResult, error) {
	endpointID := endpoint.EndpointID(strings.TrimSpace(request.GetEndpointId()))
	routeID := endpoint.RouteID(strings.TrimSpace(request.GetRouteId()))
	host.registryMu.Lock()
	defer host.registryMu.Unlock()
	current, err := host.loadRegistryLocked(ctx)
	if err != nil {
		return nil, err
	}
	target, ok := current.Endpoints[endpointID]
	if !ok {
		return nil, fmt.Errorf("endpoint %q does not exist", endpointID)
	}
	route, ok := target.Route(routeID)
	if !ok || route.Kind != endpoint.RouteSSHWebRTCTCP {
		return nil, fmt.Errorf("endpoint %q route %q is not an SSH WebRTC TCP route", endpointID, routeID)
	}
	if route.CredentialDescriptor == nil || route.CredentialDescriptor.Kind != endpoint.CredentialSSHPrivateKey {
		return nil, fmt.Errorf("SSH route %q requires an ssh-private-key credential descriptor", routeID)
	}
	credentialRef := strings.TrimSpace(route.SSHCredentialRef)
	if credentialRef == "" {
		credentialRef = platformSSHCredentialRef(endpointID, routeID)
	} else if !strings.HasPrefix(credentialRef, platformSSHCredentialPrefix) {
		return nil, fmt.Errorf("SSH route %q is bound to a credential owned by another platform", routeID)
	}
	record, publicKey, err := platformSSHCredential{broker: host.options.Broker}.lookup(ctx, credentialRef, true)
	if err != nil {
		return nil, err
	}
	next, err := cloneRegistry(current)
	if err != nil {
		return nil, rollbackSSHCredential(host.options.Broker, record, err)
	}
	target = next.Endpoints[endpointID]
	route = target.Routes[routeID]
	route.SSHCredentialRef = credentialRef
	target.Routes[routeID] = route
	next.Endpoints[endpointID] = target
	next, err = next.Normalize()
	if err != nil {
		return nil, rollbackSSHCredential(host.options.Broker, record, err)
	}
	wireRegistry, err := host.storeRegistryLocked(ctx, next, nil)
	if err != nil {
		return nil, rollbackSSHCredential(host.options.Broker, record, err)
	}
	wireEndpoint, err := endpoint.EndpointToProto(next.Endpoints[endpointID])
	if err != nil {
		return nil, err
	}
	sshPublic, err := golangssh.NewPublicKey(publicKey)
	if err != nil {
		return nil, fmt.Errorf("encode platform SSH public key: %w", err)
	}
	return &bindingpb.SSHCredentialProvisionResult{
		Endpoint: wireEndpoint, Registry: wireRegistry, CredentialRef: credentialRef,
		AuthorizedKey:  strings.TrimSpace(string(golangssh.MarshalAuthorizedKey(sshPublic))),
		KeyFingerprint: golangssh.FingerprintSHA256(sshPublic),
	}, nil
}

func platformSSHCredentialRef(endpointID endpoint.EndpointID, routeID endpoint.RouteID) string {
	sum := sha256.Sum256([]byte(string(endpointID) + "\x00" + string(routeID)))
	return fmt.Sprintf("%s%x", platformSSHCredentialPrefix, sum[:16])
}

func rollbackSSHCredential(broker platformBroker, record *bindingpb.SSHCredentialRecord, cause error) error {
	if record == nil || !record.GetNewlyCreated() {
		return cause
	}
	// caller 可能已取消；补偿删除必须使用独立有界上下文，不能把未发布的 key 留在平台 secure store。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_SshCredentialDelete{
		SshCredentialDelete: &bindingpb.SSHCredentialDeleteRequest{CredentialRef: record.GetCredentialRef()},
	}})
	if err != nil {
		return fmt.Errorf("%v; roll back SSH credential: %w", cause, err)
	}
	if err := platformResponseError(response); err != nil {
		return fmt.Errorf("%v; roll back SSH credential: %w", cause, err)
	}
	return cause
}

type platformBroker interface {
	Exchange(context.Context, *bindingpb.PlatformRequest) (*bindingpb.PlatformResponse, error)
}

type platformSSHCredential struct{ broker platformBroker }

func (source platformSSHCredential) AvailableContext(ctx context.Context, reference string) bool {
	_, _, err := source.lookup(ctx, reference, false)
	return err == nil
}

func (source platformSSHCredential) ResolveSSHCredential(ctx context.Context, reference string, descriptor *endpoint.CredentialDescriptor) (port.SSHCredential, error) {
	if descriptor == nil || descriptor.Kind != endpoint.CredentialSSHPrivateKey {
		return port.SSHCredential{}, fmt.Errorf("Android SSH Route requires an ssh-private-key descriptor")
	}
	_, publicKey, err := source.lookup(ctx, reference, false)
	if err != nil {
		return port.SSHCredential{}, err
	}
	signer, err := golangssh.NewSignerFromSigner(&platformSSHSigner{ctx: ctx, broker: source.broker, credentialRef: reference, publicKey: publicKey})
	if err != nil {
		return port.SSHCredential{}, fmt.Errorf("create platform SSH signer: %w", err)
	}
	return port.SSHCredential{AuthMethods: []golangssh.AuthMethod{golangssh.PublicKeys(signer)}}, nil
}

func (source platformSSHCredential) lookup(ctx context.Context, reference string, create bool) (*bindingpb.SSHCredentialRecord, crypto.PublicKey, error) {
	reference = strings.TrimSpace(reference)
	if !strings.HasPrefix(reference, platformSSHCredentialPrefix) {
		return nil, nil, fmt.Errorf("platform SSH credential ref is invalid")
	}
	response, err := source.broker.Exchange(ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_SshCredentialLookup{
		SshCredentialLookup: &bindingpb.SSHCredentialLookupRequest{CredentialRef: reference, CreateIfMissing: create},
	}})
	if err != nil {
		return nil, nil, err
	}
	if err := platformResponseError(response); err != nil {
		return nil, nil, err
	}
	record := response.GetSshCredential()
	if record == nil || record.GetCredentialRef() != reference || len(record.GetPublicKeyPkix()) == 0 {
		return nil, nil, fmt.Errorf("platform SSH credential response is invalid")
	}
	publicKey, err := x509.ParsePKIXPublicKey(record.GetPublicKeyPkix())
	if err != nil {
		return nil, nil, fmt.Errorf("parse platform SSH public key: %w", err)
	}
	ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
	if !ok || ecdsaKey.Curve.Params().Name != "P-256" {
		return nil, nil, fmt.Errorf("platform SSH key must use ECDSA P-256")
	}
	return record, ecdsaKey, nil
}

type platformSSHSigner struct {
	ctx           context.Context
	broker        platformBroker
	credentialRef string
	publicKey     crypto.PublicKey
}

func (signer *platformSSHSigner) Public() crypto.PublicKey { return signer.publicKey }

func (signer *platformSSHSigner) Sign(_ io.Reader, digest []byte, options crypto.SignerOpts) ([]byte, error) {
	if options == nil || options.HashFunc() != crypto.SHA256 || len(digest) != crypto.SHA256.Size() {
		return nil, fmt.Errorf("platform SSH signer only accepts SHA-256 digests")
	}
	response, err := signer.broker.Exchange(signer.ctx, &bindingpb.PlatformRequest{Request: &bindingpb.PlatformRequest_SshCredentialSign{
		SshCredentialSign: &bindingpb.SSHCredentialSignRequest{
			CredentialRef: signer.credentialRef, Digest: append([]byte(nil), digest...), Hash: crypto.SHA256.String(),
		},
	}})
	if err != nil {
		return nil, err
	}
	if err := platformResponseError(response); err != nil {
		return nil, err
	}
	signature := append([]byte(nil), response.GetSshCredentialSign().GetSignature()...)
	if len(signature) == 0 {
		return nil, fmt.Errorf("platform SSH signer returned an empty signature")
	}
	return signature, nil
}

var _ port.SSHCredentialSource = platformSSHCredential{}
var _ crypto.Signer = (*platformSSHSigner)(nil)
