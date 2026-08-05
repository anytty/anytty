package protocol

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	internalprotocol "github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/remoteauth"
)

const attachmentStreamReadyTimeout = 5 * time.Second

// ApplicationClient 组合单条 ready connection 的 Proto application session 与可选 runtime control plane。
// raw framing client 只存在于 route adapter 尚未发布 winner 的内部阶段；CLI/TUI facade 的 command、attachment 与 file stream 都经 generation-fenced ready capability。
type ApplicationClient struct {
	*internalprotocol.Client
	*clientruntime.ApplicationSession
	ready    clientruntime.ApplicationReadyPeerSession
	control  clientruntime.Runtime
	path     string
	evidence clientruntime.ReadyPeerSessionEvidence
}

// MarkReady 在 adapter 按 route 类型完成身份边界、authorization 与 Hello 后冻结 readiness evidence。
// 该方法只能在 session 发布给 SessionOwner 前调用一次；缺失或重复 evidence 都直接失败。
func (client *ApplicationClient) MarkReady(evidence clientruntime.ReadyPeerSessionEvidence) error {
	if client == nil {
		return errors.New("protocol application client is required")
	}
	if client.evidence.IdentityVerified || client.evidence.AuthorizationVerified || client.evidence.ProtocolVersion != 0 {
		return errors.New("protocol application client readiness is already frozen")
	}
	if err := evidence.Validate(endpoint.DaemonIdentity{}); err != nil {
		return err
	}
	client.evidence = evidence
	return nil
}

// VerifyDaemonIdentity 通过当前已经完成 Hello 的 authenticated application session读取 daemon public identity，
// 校验 public key fingerprint，并在 Endpoint 已有 pin 时要求 DeviceID/Fingerprint 精确匹配。
func VerifyDaemonIdentity(ctx context.Context, session *clientruntime.ApplicationSession, expected endpoint.DaemonIdentity) (endpoint.DaemonIdentity, error) {
	result, err := VerifyDaemonIdentityResult(ctx, session, expected)
	if err != nil {
		return endpoint.DaemonIdentity{}, err
	}
	identity := result.GetIdentity()
	return endpoint.DaemonIdentity{DeviceID: identity.GetDeviceId(), DeviceFingerprint: identity.GetDeviceFingerprint()}, nil
}

// VerifyDaemonIdentityResult 执行 fresh challenge 并返回已验签的完整 Proto identity result，供需要展示 public key 的 application consumer 使用。
// 它不授予 capability；调用方仍只能在 local/SSH 已授权 session 或 managed remote-auth 后调用。
func VerifyDaemonIdentityResult(ctx context.Context, session *clientruntime.ApplicationSession, expected endpoint.DaemonIdentity) (*apipb.ClientAccessIdentityResult, error) {
	if session == nil {
		return nil, errors.New("application session is required for daemon identity proof")
	}
	challenge := make([]byte, remoteauth.DeviceIdentityChallengeBytes)
	if _, err := rand.Read(challenge); err != nil {
		return nil, fmt.Errorf("generate daemon identity challenge: %w", err)
	}
	result, err := session.ClientAccessIdentity(ctx, &apipb.ClientAccessIdentityCommand{Challenge: challenge})
	if err != nil {
		return nil, fmt.Errorf("read daemon identity proof: %w", err)
	}
	identity := result.GetIdentity()
	if identity == nil || !bytes.Equal(result.GetChallenge(), challenge) {
		return nil, errors.New("daemon identity proof is incomplete")
	}
	verified := endpoint.DaemonIdentity{DeviceID: identity.GetDeviceId(), DeviceFingerprint: identity.GetDeviceFingerprint()}
	if err := verified.Validate(true); err != nil {
		return nil, fmt.Errorf("daemon identity proof is invalid: %w", err)
	}
	if err := remoteauth.VerifyDeviceIdentityProof(challenge, identity.GetDeviceId(), identity.GetDeviceFingerprint(), identity.GetDevicePublicKey(), result.GetProof()); err != nil {
		return nil, err
	}
	if !expected.Empty() && verified != expected {
		return nil, errors.New("daemon identity proof does not match endpoint pin")
	}
	return result, nil
}

// HistoryWindow 显式选择 Proto application API，消除 framing client 旧同名方法的嵌入歧义。
func (client *ApplicationClient) HistoryWindow(ctx context.Context, command *apipb.HistoryWindowCommand) (*apipb.HistoryWindowResult, error) {
	return client.ApplicationSession.HistoryWindow(ctx, command)
}

// HistoryCopy 从 frozen Proto history window 复制 authoritative text。
func (client *ApplicationClient) HistoryCopy(ctx context.Context, command *apipb.HistoryCopyCommand) (*apipb.HistoryCopyResult, error) {
	return client.ApplicationSession.HistoryCopy(ctx, command)
}

// HistoryRelease 释放 Proto history token。
func (client *ApplicationClient) HistoryRelease(ctx context.Context, command *apipb.HistoryReleaseCommand) error {
	return client.ApplicationSession.HistoryRelease(ctx, command)
}

// LiveScreenNext 等待并读取 Proto native screen projection。
func (client *ApplicationClient) LiveScreenNext(ctx context.Context, command *apipb.LiveScreenNextCommand) (*apipb.NativeScreenResult, error) {
	return client.ApplicationSession.LiveScreenNext(ctx, command)
}

// NewApplicationClient 把已完成 Hello/auth 的 framing client 绑定到 runtime winner stamp。
// 构造失败时不得返回部分可用客户端，也不得 fallback 到旧 terminal method。
func NewApplicationClient(client *internalprotocol.Client, stamp clientruntime.EndpointSessionStamp) (*ApplicationClient, error) {
	return NewApplicationClientWithObservedPath(client, stamp, "protocol")
}

// NewApplicationClientWithObservedPath 把已完成 Hello/auth 的 framing client 绑定到 attempt stamp 与观测路径。
// observedPath 仅用于诊断，不参与 route 选择或 session identity。
func NewApplicationClientWithObservedPath(client *internalprotocol.Client, stamp clientruntime.EndpointSessionStamp, observedPath string) (*ApplicationClient, error) {
	if client == nil {
		return nil, errors.New("protocol client is required")
	}
	application, err := clientruntime.NewApplicationSession(stamp, client)
	if err != nil {
		return nil, err
	}
	return &ApplicationClient{Client: client, ApplicationSession: application, path: observedPath}, nil
}

// NewReadyApplicationClient 把 runtime-owned ready session 投影为现有 typed Proto client facade。
// 它不暴露 raw framing client；attachment/file primitive 只能经 ready session 的 generation-fenced capability 调用。
func NewReadyApplicationClient(ready clientruntime.ApplicationReadyPeerSession) (*ApplicationClient, error) {
	return NewRuntimeApplicationClient(ready, nil)
}

// NewRuntimeApplicationClient 在 ready data plane 之外保留同一 ClientRuntime control plane，供 TUI lifecycle adapter 订阅 endpoint event。
// control 不能执行 Proto command 或暴露 transport；nil 仅用于不需要重连/事件的测试与内部 harness。
func NewRuntimeApplicationClient(ready clientruntime.ApplicationReadyPeerSession, control clientruntime.Runtime) (*ApplicationClient, error) {
	if ready == nil {
		return nil, errors.New("ready application session is required")
	}
	application, err := clientruntime.NewApplicationSession(ready.Stamp(), ready)
	if err != nil {
		return nil, err
	}
	return &ApplicationClient{ApplicationSession: application, ready: ready, control: control, path: ready.ObservedPath(), evidence: ready.Readiness()}, nil
}

// ConnectionRuntime 返回创建当前 ready session 的共享控制面；没有装配时返回 nil。
func (client *ApplicationClient) ConnectionRuntime() clientruntime.Runtime {
	if client == nil {
		return nil
	}
	return client.control
}

// ObservedPath 返回 route adapter 观测到的实际连接路径。
func (client *ApplicationClient) ObservedPath() string {
	if client == nil {
		return ""
	}
	return client.path
}

// ConnectionSnapshot samples the selected pair owned by this exact ready-session generation.
func (client *ApplicationClient) ConnectionSnapshot(at time.Time) (clientruntime.ConnectionSnapshot, bool) {
	if client == nil {
		return clientruntime.ConnectionSnapshot{}, false
	}
	provider, ok := client.ready.(clientruntime.ConnectionSnapshotProvider)
	if !ok {
		return clientruntime.ConnectionSnapshot{}, false
	}
	return provider.ConnectionSnapshot(at)
}

// Readiness 返回 adapter 在 session 发布前冻结的 identity、authorization 与 Hello 证据。
func (client *ApplicationClient) Readiness() clientruntime.ReadyPeerSessionEvidence {
	if client == nil {
		return clientruntime.ReadyPeerSessionEvidence{}
	}
	if client.ready != nil {
		return client.ready.Readiness()
	}
	return client.evidence
}

// ExecuteApplication 运输完整 generated Proto command；owned client 会先通过 SessionOwner generation fence。
func (client *ApplicationClient) ExecuteApplication(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if client.ready != nil {
		return client.ready.ExecuteApplication(ctx, command)
	}
	return client.Client.ExecuteApplication(ctx, command)
}

// ValidateApplicationSession 在 protocol/resource 副作用前确认请求 stamp 与当前 facade 一致，并委托 runtime owner 校验 generation。
func (client *ApplicationClient) ValidateApplicationSession(stamp clientruntime.EndpointSessionStamp) error {
	if client == nil || client.ApplicationSession == nil {
		return &clientruntime.Error{Code: clientruntime.ErrorUnavailable, Message: "protocol application client is unavailable"}
	}
	if client.ApplicationSession.Stamp() != stamp {
		return &clientruntime.Error{Code: clientruntime.ErrorStaleSession, Message: "application operation session stamp does not match protocol client"}
	}
	if validator, ok := client.ready.(clientruntime.ApplicationSessionValidator); ok {
		return validator.ValidateApplicationSession(stamp)
	}
	return nil
}

// ExecuteApplicationTerminal 为 resource-producing binding operation 保留有界 terminal response。
// 该选择来自上层 owner；protocol adapter 只转交通用能力，不解释 command oneof。
func (client *ApplicationClient) ExecuteApplicationTerminal(ctx context.Context, command *apipb.CommandEnvelope) (*apipb.ResultEnvelope, error) {
	if executor, ok := client.ready.(clientruntime.TerminalResponseApplicationExecutor); ok {
		return executor.ExecuteApplicationTerminal(ctx, command)
	}
	if client == nil || client.Client == nil {
		return nil, errors.New("ready session does not support terminal application responses")
	}
	return client.Client.ExecuteApplicationTerminal(ctx, command)
}

// ApplicationEvents 返回同一 ready connection 的 generated Proto event stream。
func (client *ApplicationClient) ApplicationEvents(ctx context.Context) (<-chan *apipb.EventEnvelope, error) {
	if client.ready != nil {
		return client.ready.ApplicationEvents(ctx)
	}
	return client.Client.ApplicationEvents(ctx)
}

// OpenResourceStream 通过 runtime generation fence 打开 session-bound attachment/file framing stream。
func (client *ApplicationClient) OpenResourceStream(resource *apipb.ResourceHandle) (clientruntime.ResourceStream, error) {
	if client == nil {
		return nil, errors.New("protocol application client is required")
	}
	if provider, ok := client.ready.(clientruntime.ResourceStreamSession); ok {
		return provider.OpenResourceStream(resource)
	}
	if client.Client != nil {
		channel, ok := client.Client.ApplicationResourceChannel(resource)
		if !ok {
			return nil, errors.New("application resource is not bound to this protocol session")
		}
		frames, stop := client.Client.Stream(channel)
		stream := &applicationResourceStream{
			client: client.Client, channel: channel, kind: resource.GetKind(), frames: frames, stop: stop,
		}
		if resource.GetKind() == apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
			if err := client.Client.SendAttachmentReady(channel); err != nil {
				_ = stream.Close()
				return nil, err
			}
			if err := stream.awaitReady(); err != nil {
				_ = stream.Close()
				return nil, err
			}
		}
		return stream, nil
	}
	return nil, errors.New("ready session does not support resource streams")
}

func (stream *applicationResourceStream) awaitReady() error {
	timer := time.NewTimer(attachmentStreamReadyTimeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		return errors.New("terminal attachment raw PTY stream ready timed out")
	case frame, ok := <-stream.frames:
		if !ok {
			return io.EOF
		}
		if frame.Type != wire.TypeStreamReady {
			return fmt.Errorf("terminal attachment raw PTY stream returned frame %d before ready", frame.Type)
		}
		return nil
	}
}

type applicationResourceStream struct {
	client   *internalprotocol.Client
	channel  uint16
	kind     apipb.ResourceKind
	frames   <-chan internalprotocol.StreamFrame
	stop     func()
	once     sync.Once
	closeErr error
	mu       sync.Mutex
	ended    bool
}

func (stream *applicationResourceStream) Receive(ctx context.Context) (uint8, []byte, error) {
	stream.mu.Lock()
	ended := stream.ended
	stream.mu.Unlock()
	if ended {
		_ = stream.Close()
		return 0, nil, io.EOF
	}
	select {
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	case frame, ok := <-stream.frames:
		if !ok {
			return 0, nil, io.EOF
		}
		if frame.Type == wire.TypeClosed || frame.Type == wire.TypeSyncLost {
			stream.mu.Lock()
			stream.ended = true
			stream.mu.Unlock()
		}
		return frame.Type, append([]byte(nil), frame.Payload...), nil
	}
}

func (stream *applicationResourceStream) Send(ctx context.Context, typ uint8, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if stream.kind != apipb.ResourceKind_RESOURCE_KIND_FILE_TRANSFER {
		return errors.New("terminal attachment stream is receive-only; input uses TerminalInputCommand")
	}
	return stream.client.SendFileFrame(stream.channel, typ, payload)
}

func (stream *applicationResourceStream) Close() error {
	stream.once.Do(func() {
		if stream.kind == apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
			stream.closeErr = stream.client.SendAttachmentStreamClose(stream.channel)
		}
		stream.stop()
	})
	return stream.closeErr
}

// ApplicationAttachmentChannel 返回当前 generation 内 attachment resource 对应的私有 channel。
func (client *ApplicationClient) ApplicationAttachmentChannel(resource *apipb.ResourceHandle) (uint16, bool) {
	if provider, ok := client.ready.(clientruntime.ApplicationAttachmentSession); ok {
		return provider.ApplicationAttachmentChannel(resource)
	}
	if client != nil && client.Client != nil {
		return client.Client.ApplicationAttachmentChannel(resource)
	}
	return 0, false
}

// ApplicationAttachment 返回当前 generation 内 channel 对应的 attachment resource。
func (client *ApplicationClient) ApplicationAttachment(channel uint16) (*apipb.ResourceHandle, bool) {
	if provider, ok := client.ready.(clientruntime.ApplicationAttachmentSession); ok {
		return provider.ApplicationAttachment(channel)
	}
	if client != nil && client.Client != nil {
		return client.Client.ApplicationAttachment(channel)
	}
	return nil, false
}

// Done 返回 ready connection 的终止信号。
func (client *ApplicationClient) Done() <-chan struct{} {
	if client.ready != nil {
		return client.ready.Done()
	}
	return client.Client.Done()
}

// Err 返回 ready connection 的终止原因。
func (client *ApplicationClient) Err() error {
	if client.ready != nil {
		return client.ready.Err()
	}
	return client.Client.Err()
}

// Close 释放 owner 中精确匹配的 generation；底层 protocol connection 由 owner session 一并关闭。
func (client *ApplicationClient) Close() error {
	if client.ready != nil {
		return client.ready.Close()
	}
	return client.Client.Close()
}

var _ clientruntime.ApplicationReadyPeerSession = (*ApplicationClient)(nil)
var _ clientruntime.ConnectionSnapshotProvider = (*ApplicationClient)(nil)
var _ clientruntime.ApplicationAttachmentSession = (*ApplicationClient)(nil)
var _ clientruntime.ResourceStreamSession = (*ApplicationClient)(nil)
