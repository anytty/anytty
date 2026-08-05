package binding

import (
	"context"
	"fmt"
	"log"
	"time"

	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/proto/bindingpb"
	"google.golang.org/protobuf/proto"
)

// EngineCommand 解析 bindingpb.EngineCommand，并把命令路由到 engine-owned 异步 operation。
// C/JNI/WASM 只暴露这个通用 Proto 入口；业务命令类型不得继续扩张跨语言符号面。
func (engine *Engine) EngineCommand(payload []byte) (uint64, error) {
	if err := validatePayload(payload); err != nil {
		return 0, err
	}
	command := &bindingpb.EngineCommand{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, command); err != nil {
		return 0, fmt.Errorf("decode engine command: %w", err)
	}
	switch value := command.GetCommand().(type) {
	case *bindingpb.EngineCommand_ImportPairing:
		encoded, err := proto.Marshal(value.ImportPairing)
		if err != nil {
			return 0, err
		}
		return engine.ImportPairing(encoded)
	case *bindingpb.EngineCommand_DeleteCredential:
		encoded, err := proto.Marshal(value.DeleteCredential)
		if err != nil {
			return 0, err
		}
		return engine.DeleteCredential(encoded)
	case *bindingpb.EngineCommand_EndpointRegistryGet:
		return engine.startEndpointRegistryGet(value.EndpointRegistryGet)
	case *bindingpb.EngineCommand_EndpointUpsert:
		return engine.startEndpointUpsert(value.EndpointUpsert)
	case *bindingpb.EngineCommand_EndpointDelete:
		return engine.startEndpointDelete(value.EndpointDelete)
	case *bindingpb.EngineCommand_EndpointShareReceive:
		return engine.startEndpointShareReceive(value.EndpointShareReceive)
	case *bindingpb.EngineCommand_EndpointShareCommit:
		return engine.startEndpointShareCommit(value.EndpointShareCommit)
	case *bindingpb.EngineCommand_SshCredentialProvision:
		return engine.startSSHCredentialProvision(value.SshCredentialProvision)
	case *bindingpb.EngineCommand_ConnectionPolicyGet:
		return engine.startConnectionPolicyGet(value.ConnectionPolicyGet)
	case *bindingpb.EngineCommand_ConnectionPolicyApply:
		return engine.startConnectionPolicyApply(value.ConnectionPolicyApply)
	case *bindingpb.EngineCommand_ConnectionSnapshotGet:
		return engine.startConnectionSnapshotGet(value.ConnectionSnapshotGet)
	case *bindingpb.EngineCommand_SessionInvalidate:
		return engine.startSessionInvalidate(value.SessionInvalidate)
	default:
		return 0, fmt.Errorf("engine command is required")
	}
}

func (engine *Engine) sessionInvalidationHost() (SessionInvalidationHost, error) {
	host, ok := engine.host.(SessionInvalidationHost)
	if !ok {
		return nil, fmt.Errorf("binding host does not support session invalidation")
	}
	return host, nil
}

func (engine *Engine) startSessionInvalidate(request *bindingpb.SessionInvalidateRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetSessionHandle() == 0 {
		return 0, fmt.Errorf("session invalidate request is incomplete")
	}
	host, err := engine.sessionInvalidationHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, session, err := engine.startSessionOperation(request.GetSessionHandle())
	if err != nil {
		return 0, err
	}
	go engine.runSessionInvalidate(handle, operationContext, host, session, proto.Clone(request).(*bindingpb.SessionInvalidateRequest))
	return handle, nil
}

func (engine *Engine) connectionPolicyHost() (ConnectionPolicyHost, error) {
	host, ok := engine.host.(ConnectionPolicyHost)
	if !ok {
		return nil, fmt.Errorf("binding host does not support connection policy operations")
	}
	return host, nil
}

func (engine *Engine) startConnectionPolicyGet(request *bindingpb.ConnectionPolicyGetRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetEndpointId() == "" {
		return 0, fmt.Errorf("connection policy get request is incomplete")
	}
	host, err := engine.connectionPolicyHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runConnectionPolicyGet(handle, operationContext, host, proto.Clone(request).(*bindingpb.ConnectionPolicyGetRequest))
	return handle, nil
}

func (engine *Engine) startConnectionPolicyApply(request *bindingpb.ConnectionPolicyApplyRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetEndpointId() == "" || request.GetPolicy() == nil {
		return 0, fmt.Errorf("connection policy apply request is incomplete")
	}
	host, err := engine.connectionPolicyHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runConnectionPolicyApply(handle, operationContext, host, proto.Clone(request).(*bindingpb.ConnectionPolicyApplyRequest))
	return handle, nil
}

func (engine *Engine) startConnectionSnapshotGet(request *bindingpb.ConnectionSnapshotGetRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetSessionHandle() == 0 {
		return 0, fmt.Errorf("connection snapshot get request is incomplete")
	}
	handle, operationContext, session, err := engine.startSessionOperation(request.GetSessionHandle())
	if err != nil {
		return 0, err
	}
	go engine.runConnectionSnapshotGet(handle, operationContext, session, proto.Clone(request).(*bindingpb.ConnectionSnapshotGetRequest))
	return handle, nil
}

func (engine *Engine) sshCredentialHost() (SSHCredentialHost, error) {
	host, ok := engine.host.(SSHCredentialHost)
	if !ok {
		return nil, fmt.Errorf("binding host does not support SSH credential provisioning")
	}
	return host, nil
}

func (engine *Engine) startSSHCredentialProvision(request *bindingpb.SSHCredentialProvisionRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetEndpointId() == "" || request.GetRouteId() == "" {
		return 0, fmt.Errorf("SSH credential provision request is incomplete")
	}
	host, err := engine.sshCredentialHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runSSHCredentialProvision(handle, operationContext, host, proto.Clone(request).(*bindingpb.SSHCredentialProvisionRequest))
	return handle, nil
}

func (engine *Engine) shareHost() (EndpointShareHost, error) {
	host, ok := engine.host.(EndpointShareHost)
	if !ok {
		return nil, fmt.Errorf("binding host does not support endpoint share operations")
	}
	return host, nil
}

func (engine *Engine) startEndpointShareReceive(request *bindingpb.EndpointShareReceiveRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetPortableOffer() == "" {
		return 0, fmt.Errorf("endpoint share receive request is incomplete")
	}
	host, err := engine.shareHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runEndpointShareReceive(handle, operationContext, host, proto.Clone(request).(*bindingpb.EndpointShareReceiveRequest))
	return handle, nil
}

func (engine *Engine) startEndpointShareCommit(request *bindingpb.EndpointShareCommitRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetImportToken() == "" {
		return 0, fmt.Errorf("endpoint share commit request is incomplete")
	}
	host, err := engine.shareHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runEndpointShareCommit(handle, operationContext, host, proto.Clone(request).(*bindingpb.EndpointShareCommitRequest))
	return handle, nil
}

func (engine *Engine) registryHost() (EndpointRegistryHost, error) {
	host, ok := engine.host.(EndpointRegistryHost)
	if !ok {
		return nil, fmt.Errorf("binding host does not support endpoint registry operations")
	}
	return host, nil
}

func (engine *Engine) startEndpointRegistryGet(request *bindingpb.EndpointRegistryGetRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" {
		return 0, fmt.Errorf("endpoint registry get request is incomplete")
	}
	host, err := engine.registryHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runEndpointRegistryGet(handle, operationContext, host, proto.Clone(request).(*bindingpb.EndpointRegistryGetRequest))
	return handle, nil
}

func (engine *Engine) startEndpointUpsert(request *bindingpb.EndpointUpsertRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetEndpoint() == nil {
		return 0, fmt.Errorf("endpoint upsert request is incomplete")
	}
	host, err := engine.registryHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runEndpointUpsert(handle, operationContext, host, proto.Clone(request).(*bindingpb.EndpointUpsertRequest))
	return handle, nil
}

func (engine *Engine) startEndpointDelete(request *bindingpb.EndpointDeleteRequest) (uint64, error) {
	if request == nil || request.GetRequestId() == "" || request.GetEndpointId() == "" {
		return 0, fmt.Errorf("endpoint delete request is incomplete")
	}
	host, err := engine.registryHost()
	if err != nil {
		return 0, err
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runEndpointDelete(handle, operationContext, host, proto.Clone(request).(*bindingpb.EndpointDeleteRequest))
	return handle, nil
}

// ImportPairing 解析 bindingpb.ImportPairingRequest 并异步请求可选 PairingHost。
// 完成结果只通过 NextEvent 发布；binding 不解析二维码、ticket 或 credential 内容。
func (engine *Engine) ImportPairing(payload []byte) (uint64, error) {
	if err := validatePayload(payload); err != nil {
		return 0, err
	}
	request := &bindingpb.ImportPairingRequest{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, request); err != nil {
		return 0, fmt.Errorf("decode import pairing request: %w", err)
	}
	if request.GetRequestId() == "" || request.GetPortablePayload() == "" {
		return 0, fmt.Errorf("import pairing request is incomplete")
	}
	host, ok := engine.host.(PairingHost)
	if !ok {
		return 0, fmt.Errorf("binding host does not support pairing import")
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runImportPairing(handle, operationContext, host, proto.Clone(request).(*bindingpb.ImportPairingRequest))
	return handle, nil
}

// DeleteCredential 解析 bindingpb.DeleteCredentialRequest 并异步删除平台 credential。
// 它不撤销 daemon grant；结果只通过 NextEvent 发布并需要显式 Release operation handle。
func (engine *Engine) DeleteCredential(payload []byte) (uint64, error) {
	if err := validatePayload(payload); err != nil {
		return 0, err
	}
	request := &bindingpb.DeleteCredentialRequest{}
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(payload, request); err != nil {
		return 0, fmt.Errorf("decode delete credential request: %w", err)
	}
	if request.GetRequestId() == "" || request.GetCredentialRef() == "" {
		return 0, fmt.Errorf("delete credential request is incomplete")
	}
	host, ok := engine.host.(CredentialHost)
	if !ok {
		return 0, fmt.Errorf("binding host does not support credential deletion")
	}
	handle, operationContext, err := engine.startOperation()
	if err != nil {
		return 0, err
	}
	go engine.runDeleteCredential(handle, operationContext, host, proto.Clone(request).(*bindingpb.DeleteCredentialRequest))
	return handle, nil
}

func (engine *Engine) runImportPairing(handle uint64, ctx context.Context, host PairingHost, request *bindingpb.ImportPairingRequest) {
	result, err := host.ImportPairing(ctx, request)
	if ctx.Err() != nil {
		result = nil
		err = ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.ImportPairingResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.ImportPairingResult)
	}
	result.RequestId = request.GetRequestId()
	result.OperationHandle = handle
	if err != nil {
		// 只记录错误链，不记录一次性 pairing payload、grant 或平台 credential。
		log.Printf("anytty binding pairing failed: %v", err)
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_ImportPairing{ImportPairing: result}})
}

func (engine *Engine) runDeleteCredential(handle uint64, ctx context.Context, host CredentialHost, request *bindingpb.DeleteCredentialRequest) {
	err := host.DeleteCredential(ctx, request)
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	engine.markOperationDone(handle)
	result := &bindingpb.DeleteCredentialResult{RequestId: request.GetRequestId(), OperationHandle: handle}
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_DeleteCredential{DeleteCredential: result}})
}

func (engine *Engine) runEndpointRegistryGet(handle uint64, ctx context.Context, host EndpointRegistryHost, request *bindingpb.EndpointRegistryGetRequest) {
	result, err := host.GetEndpointRegistry(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.EndpointRegistryGetResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.EndpointRegistryGetResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_EndpointRegistryGet{EndpointRegistryGet: result}})
}

func (engine *Engine) runEndpointUpsert(handle uint64, ctx context.Context, host EndpointRegistryHost, request *bindingpb.EndpointUpsertRequest) {
	result, err := host.UpsertEndpoint(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.EndpointUpsertResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.EndpointUpsertResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_EndpointUpsert{EndpointUpsert: result}})
}

func (engine *Engine) runEndpointDelete(handle uint64, ctx context.Context, host EndpointRegistryHost, request *bindingpb.EndpointDeleteRequest) {
	result, err := host.DeleteEndpoint(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.EndpointDeleteResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.EndpointDeleteResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_EndpointDelete{EndpointDelete: result}})
}

func (engine *Engine) runEndpointShareReceive(handle uint64, ctx context.Context, host EndpointShareHost, request *bindingpb.EndpointShareReceiveRequest) {
	result, err := host.ReceiveEndpointShare(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.EndpointShareReceiveResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.EndpointShareReceiveResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_EndpointShareReceive{EndpointShareReceive: result}})
}

func (engine *Engine) runEndpointShareCommit(handle uint64, ctx context.Context, host EndpointShareHost, request *bindingpb.EndpointShareCommitRequest) {
	result, err := host.CommitEndpointShare(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.EndpointShareCommitResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.EndpointShareCommitResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_EndpointShareCommit{EndpointShareCommit: result}})
}

func (engine *Engine) runSSHCredentialProvision(handle uint64, ctx context.Context, host SSHCredentialHost, request *bindingpb.SSHCredentialProvisionRequest) {
	result, err := host.ProvisionSSHCredential(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.SSHCredentialProvisionResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.SSHCredentialProvisionResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_SshCredentialProvision{SshCredentialProvision: result}})
}

func (engine *Engine) runConnectionPolicyGet(handle uint64, ctx context.Context, host ConnectionPolicyHost, request *bindingpb.ConnectionPolicyGetRequest) {
	result, err := host.GetConnectionPolicy(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.ConnectionPolicyGetResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.ConnectionPolicyGetResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_ConnectionPolicyGet{ConnectionPolicyGet: result}})
}

func (engine *Engine) runConnectionPolicyApply(handle uint64, ctx context.Context, host ConnectionPolicyHost, request *bindingpb.ConnectionPolicyApplyRequest) {
	result, err := host.ApplyConnectionPolicy(ctx, request)
	if ctx.Err() != nil {
		result, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	if result == nil {
		result = &bindingpb.ConnectionPolicyApplyResult{}
	} else {
		result = proto.Clone(result).(*bindingpb.ConnectionPolicyApplyResult)
	}
	result.RequestId, result.OperationHandle = request.GetRequestId(), handle
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_ConnectionPolicyApply{ConnectionPolicyApply: result}})
}

func (engine *Engine) runConnectionSnapshotGet(handle uint64, ctx context.Context, session clientruntime.ApplicationReadyPeerSession, request *bindingpb.ConnectionSnapshotGetRequest) {
	var snapshot *bindingpb.ConnectionSnapshot
	var err error
	if provider, ok := session.(clientruntime.ConnectionSnapshotProvider); !ok {
		err = fmt.Errorf("application session does not expose a connection snapshot")
	} else if value, valid := provider.ConnectionSnapshot(time.Now().UTC()); !valid {
		err = fmt.Errorf("application session connection snapshot is unavailable")
	} else {
		snapshot = connectionSnapshotToProto(value)
	}
	if ctx.Err() != nil {
		snapshot, err = nil, ctx.Err()
	}
	engine.markOperationDone(handle)
	result := &bindingpb.ConnectionSnapshotGetResult{
		RequestId: request.GetRequestId(), OperationHandle: handle, SessionHandle: request.GetSessionHandle(), Connection: snapshot,
	}
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_ConnectionSnapshotGet{ConnectionSnapshotGet: result}})
}

func (engine *Engine) runSessionInvalidate(handle uint64, ctx context.Context, host SessionInvalidationHost, session clientruntime.ApplicationReadyPeerSession, request *bindingpb.SessionInvalidateRequest) {
	err := host.InvalidateSession(ctx, session.Stamp())
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	engine.markOperationDone(handle)
	result := &bindingpb.SessionInvalidateResult{
		RequestId: request.GetRequestId(), OperationHandle: handle, SessionHandle: request.GetSessionHandle(),
	}
	if err != nil {
		result.Error = apiError(err)
	}
	engine.emit(&bindingpb.EventEnvelope{Event: &bindingpb.EventEnvelope_SessionInvalidate{SessionInvalidate: result}})
}
