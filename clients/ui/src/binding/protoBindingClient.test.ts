import { create, fromBinary, toBinary } from '@bufbuild/protobuf'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { AcknowledgeResultSchema, CommandEnvelopeSchema, ResultEnvelopeSchema } from '../generated/apipb/application_pb'
import { ApiErrorCode, ApiErrorSchema, EndpointSessionStampSchema, ResourceHandleSchema, ResourceKind } from '../generated/apipb/common_pb'
import { EventSubscribeCommandSchema, EventSubscriptionResultSchema } from '../generated/apipb/events_pb'
import { FileTransferCancelResultSchema, FileTransferHandleSchema, FileTransferOpenResultSchema, FileUploadOpenCommandSchema, FileUploadResumeHandleSchema } from '../generated/apipb/file_pb'
import { HistoryWindowCommandSchema, HistoryWindowMode, HistoryWindowResultSchema } from '../generated/apipb/history_pb'
import { TerminalListCommandSchema, TerminalRefSchema } from '../generated/apipb/terminal_pb'
import { ConnectionPolicyApplyResultSchema, ConnectionPolicyAvailabilityReason, ConnectionPolicyGetResultSchema, ConnectionPolicyRouteAvailabilitySchema, ConnectionPolicySchema, ConnectionPolicyStateSchema, ConnectionRouteKind, ConnectionSnapshotGetResultSchema, ConnectionSnapshotSchema, EndpointRegistryGetResultSchema, EndpointShareCommitResultSchema, EndpointSharePreviewSchema, EndpointShareReceiveResultSchema, EngineCommandSchema, type EventEnvelope, EventEnvelopeSchema, ExecuteResultSchema, OpenSessionRequestSchema, OpenSessionResultSchema, ResourceStreamClosedEventSchema, ResourceStreamFrameSchema, ResourceStreamFrameType, SessionClosedEventSchema, SessionInvalidateResultSchema, SSHCredentialProvisionResultSchema } from '../generated/bindingpb/client_binding_pb'
import { EndpointConfigV1Schema, EndpointRegistryV1Schema, EndpointRouteConfigV1Schema, EndpointRoutePreference, ManagedWebRTCRelayMode, ManagedWebRTCRelayTransport, ManagedWebRTCRouteConfigSchema } from '../generated/remoteauthpb/remote_auth_pb'
import { BindingOperation, ProtoBindingClient, ProtoBindingConnector, type BindingOperationCode, type ProtoBindingBackend } from './protoBindingClient'

class CancellationBackend implements ProtoBindingBackend {
  readonly released: bigint[] = []
  private next = 0n
  private onEvent: ((payload: Uint8Array) => void) | null = null

  start(onEvent: (payload: Uint8Array) => void): void { this.onEvent = onEvent }

  emit(payload: Uint8Array): void { this.onEvent?.(payload) }

  async request(operation: BindingOperationCode, _payload: Uint8Array, handle?: bigint): Promise<bigint> {
    if (operation === BindingOperation.EXECUTE) return ++this.next
    if (operation === BindingOperation.CANCEL && handle) {
      queueMicrotask(() => this.onEvent?.(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
        event: { case: 'execute', value: create(ExecuteResultSchema, { operationHandle: handle, sessionHandle: 1n }) },
      }))))
    }
    if (operation === BindingOperation.RELEASE && handle) this.released.push(handle)
    return 0n
  }

  async close(): Promise<void> {}
}

describe('ProtoBindingClient cancellation ownership', () => {
  it('releases every cancelled operation after its terminal event', async () => {
    const backend = new CancellationBackend()
    const client = new ProtoBindingClient(backend)
    const command = create(CommandEnvelopeSchema, { command: { case: 'terminalList', value: create(TerminalListCommandSchema) } })

    for (let index = 0; index < 4100; index += 1) {
      const controller = new AbortController()
      const pending = client.execute(1n, command, controller.signal)
      controller.abort(new DOMException('cancel test', 'AbortError'))
      await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    }
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.released).toHaveLength(4100)
    await client.close()
  })
})

describe('ProtoBindingSession invalidation', () => {
  it('invalidates the exact binding session through an engine command', async () => {
    const backend = new CancellationBackend()
    let nextOperation = 0n
    let invalidatedSession = 0n
    backend.request = async (operation, payload, handle) => {
      if (operation === BindingOperation.OPEN_SESSION) {
        const operationHandle = ++nextOperation
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'openSession', value: create(OpenSessionResultSchema, {
            operationHandle,
            sessionHandle: 70n,
            session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'direct', generation: 1n }),
          }) },
        }))))
        return operationHandle
      }
      if (operation === BindingOperation.ENGINE_COMMAND) {
        const command = fromBinary(EngineCommandSchema, payload)
        expect(command.command.case).toBe('sessionInvalidate')
        invalidatedSession = command.command.case === 'sessionInvalidate' ? command.command.value.sessionHandle : 0n
        const operationHandle = ++nextOperation
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'sessionInvalidate', value: create(SessionInvalidateResultSchema, {
            operationHandle,
            sessionHandle: invalidatedSession,
          }) },
        }))))
        return operationHandle
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)

    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    await expect(session.invalidate?.()).resolves.toBeUndefined()
    expect(invalidatedSession).toBe(70n)
    await client.close()
  })

  it('retires a session and notifies the workspace when application commands report it unavailable', async () => {
    const backend = new CancellationBackend()
    let nextOperation = 0n
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_SESSION) {
        const operationHandle = ++nextOperation
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'openSession', value: create(OpenSessionResultSchema, {
            operationHandle,
            sessionHandle: 70n,
            session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'direct', generation: 1n }),
          }) },
        }))))
        return operationHandle
      }
      if (operation === BindingOperation.EXECUTE) {
        const operationHandle = ++nextOperation
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'execute', value: create(ExecuteResultSchema, {
            operationHandle,
            sessionHandle: handle,
            error: create(ApiErrorSchema, {
              code: ApiErrorCode.UNAVAILABLE,
              message: 'client session is unavailable',
              retryable: true,
            }),
          }) },
        }))))
        return operationHandle
      }
      return 0n
    }
    const invalidated = vi.fn()
    document.addEventListener('anytty:session-invalidated', invalidated)
    const client = new ProtoBindingClient(backend)

    try {
      const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
      const closed = vi.fn()
      session.subscribeClosed(closed)
      await expect(session.execute(create(CommandEnvelopeSchema))).rejects.toMatchObject({ code: 'unavailable' })
      expect(session.isAlive()).toBe(false)
      expect(invalidated).toHaveBeenCalledTimes(1)
      expect(closed).toHaveBeenCalledTimes(1)
      expect(closed).toHaveBeenCalledWith(expect.objectContaining({ code: 'unavailable', retryable: true }))

      backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
        event: { case: 'sessionClosed', value: create(SessionClosedEventSchema, {
          sessionHandle: 70n,
          error: create(ApiErrorSchema, { code: ApiErrorCode.UNAVAILABLE, message: 'already closed', retryable: true }),
        }) },
      })))
      expect(closed).toHaveBeenCalledTimes(1)
    } finally {
      document.removeEventListener('anytty:session-invalidated', invalidated)
      await client.close()
    }
  })

  it('publishes a structured asynchronous close reason exactly once', async () => {
    const backend = new CancellationBackend()
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_SESSION) {
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'openSession', value: create(OpenSessionResultSchema, {
            operationHandle: 1n,
            sessionHandle: 70n,
            session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
          }) },
        }))))
        return 1n
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    const closed = vi.fn()
    session.subscribeClosed(closed)
    const event = toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'sessionClosed', value: create(SessionClosedEventSchema, {
        sessionHandle: 70n,
        error: create(ApiErrorSchema, { code: ApiErrorCode.DAEMON_BLOCKED, message: 'daemon blocked', retryable: true }),
      }) },
    }))

    backend.emit(event)
    backend.emit(event)

    expect(session.isAlive()).toBe(false)
    expect(closed).toHaveBeenCalledTimes(1)
    expect(closed).toHaveBeenCalledWith(expect.objectContaining({
      message: 'daemon blocked',
      code: 'daemon_blocked',
      retryable: true,
    }))
    await client.close()
  })
})

describe('ProtoBindingClient engine command boundary', () => {
  it('owns operation release failures when the binding generation closes', async () => {
    const backend = new CancellationBackend()
    const close = vi.spyOn(backend, 'close')
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.ENGINE_COMMAND) {
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'endpointRegistryGet', value: create(EndpointRegistryGetResultSchema, {
            operationHandle: 1n,
            registry: create(EndpointRegistryV1Schema, { schemaVersion: 1 }),
          }) },
        }))))
        return 1n
      }
      if (operation === BindingOperation.RELEASE && handle === 1n) throw new Error('binding generation closed during release')
      return 0n
    }
    const client = new ProtoBindingClient(backend)

    await expect(client.getEndpointRegistry()).resolves.toMatchObject({ schemaVersion: 1 })
    await vi.waitFor(() => expect(close).toHaveBeenCalledTimes(1))
  })

  it('queries the Go-owned endpoint registry through one generic command operation', async () => {
    const backend = new CancellationBackend()
    backend.request = async (operation, payload, handle) => {
      if (operation === BindingOperation.ENGINE_COMMAND) {
        const command = fromBinary(EngineCommandSchema, payload)
        expect(command.command.case).toBe('endpointRegistryGet')
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'endpointRegistryGet', value: create(EndpointRegistryGetResultSchema, {
            operationHandle: 1n,
            registry: create(EndpointRegistryV1Schema, { schemaVersion: 1, defaultEndpointId: 'studio' }),
          }) },
        }))))
        return 1n
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    await expect(client.getEndpointRegistry()).resolves.toMatchObject({ defaultEndpointId: 'studio' })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.released).toEqual([1n])
    await client.close()
  })

  it('keeps endpoint share preview and commit as two generic Proto operations', async () => {
  const backend = new CancellationBackend()
  let next = 0n
  backend.request = async (operation, payload, handle) => {
    if (operation === BindingOperation.ENGINE_COMMAND) {
    const command = fromBinary(EngineCommandSchema, payload)
    const operationHandle = ++next
    if (command.command.case === 'endpointShareReceive') {
      queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'endpointShareReceive', value: create(EndpointShareReceiveResultSchema, {
        operationHandle,
        preview: create(EndpointSharePreviewSchema, { importToken: 'preview-token', endpointId: 'studio' }),
      }) },
      }))))
    } else if (command.command.case === 'endpointShareCommit') {
      expect(command.command.value.importToken).toBe('preview-token')
      queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'endpointShareCommit', value: create(EndpointShareCommitResultSchema, {
        operationHandle,
        endpoint: create(EndpointConfigV1Schema, { schemaVersion: 1, endpointId: 'studio' }),
        registry: create(EndpointRegistryV1Schema, { schemaVersion: 1, defaultEndpointId: 'studio' }),
        authorizationRequired: true,
      }) },
      }))))
    } else {
      throw new Error(`unexpected command ${command.command.case}`)
    }
    return operationHandle
    }
    if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
    return 0n
  }
  const client = new ProtoBindingClient(backend)
  const received = await client.receiveEndpointShare('anytty://share?payload=test')
  expect(received.preview?.importToken).toBe('preview-token')
  const committed = await client.commitEndpointShare(received.preview?.importToken ?? '')
  expect(committed.authorizationRequired).toBe(true)
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(backend.released).toEqual([1n, 2n])
  await client.close()
  })

  it('provisions an SSH signer through the generic Proto engine command', async () => {
    const backend = new CancellationBackend()
    backend.request = async (operation, payload, handle) => {
      if (operation === BindingOperation.ENGINE_COMMAND) {
        const command = fromBinary(EngineCommandSchema, payload)
        expect(command.command.case).toBe('sshCredentialProvision')
        if (command.command.case !== 'sshCredentialProvision') throw new Error('unexpected command')
        expect(command.command.value).toMatchObject({ endpointId: 'studio', routeId: 'ssh' })
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'sshCredentialProvision', value: create(SSHCredentialProvisionResultSchema, {
            operationHandle: 1n,
            endpoint: create(EndpointConfigV1Schema, { schemaVersion: 1, endpointId: 'studio' }),
            registry: create(EndpointRegistryV1Schema, { schemaVersion: 1, defaultEndpointId: 'studio' }),
            credentialRef: 'ssh-platform-test',
            authorizedKey: 'ecdsa-sha2-nistp256 AAAA',
            keyFingerprint: 'SHA256:test',
          }) },
        }))))
        return 1n
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    await expect(client.provisionSSHCredential('studio', 'ssh')).resolves.toMatchObject({ credentialRef: 'ssh-platform-test' })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.released).toEqual([1n])
    await client.close()
  })

  it('routes policy and live snapshot operations through generated engine commands', async () => {
    const backend = new CancellationBackend()
    let operationHandle = 0n
    const state = create(ConnectionPolicyStateSchema, { policy: create(ConnectionPolicySchema, {
      routePreference: EndpointRoutePreference.AUTO,
    }) })
    backend.request = async (operation, payload, handle) => {
      if (operation === BindingOperation.ENGINE_COMMAND) {
        operationHandle++
        const current = operationHandle
        const command = fromBinary(EngineCommandSchema, payload)
        const event = command.command.case === 'connectionPolicyGet'
          ? { case: 'connectionPolicyGet' as const, value: create(ConnectionPolicyGetResultSchema, { operationHandle: current, state }) }
          : command.command.case === 'connectionPolicyApply'
            ? { case: 'connectionPolicyApply' as const, value: create(ConnectionPolicyApplyResultSchema, { operationHandle: current, state }) }
            : command.command.case === 'connectionSnapshotGet'
              ? { case: 'connectionSnapshotGet' as const, value: create(ConnectionSnapshotGetResultSchema, {
                  operationHandle: current,
                  sessionHandle: command.command.value.sessionHandle,
                  connection: create(ConnectionSnapshotSchema, { routeId: 'direct', bytesReceived: 42n, connected: true }),
                }) }
              : undefined
        if (!event) throw new Error(`unexpected command ${command.command.case}`)
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event }))))
        return current
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)

    await expect(client.getConnectionPolicy('studio')).resolves.toEqual(state)
    await expect(client.applyConnectionPolicy('studio', state.policy!)).resolves.toEqual(state)
    await expect(client.getConnectionSnapshot(77n)).resolves.toMatchObject({ routeId: 'direct', bytesReceived: 42n })
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.released).toEqual([1n, 2n, 3n])
    await client.close()
  })
})

describe('ProtoBindingConnector route policy', () => {
  it('reads availability and persists a Cloud route policy through Proto registry commands', async () => {
    const state = create(ConnectionPolicyStateSchema, {
      policy: create(ConnectionPolicySchema, {
        routePreference: EndpointRoutePreference.AUTO,
        cloudRelayMode: ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO,
        relayTransport: ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO,
      }),
      routes: [
        create(ConnectionPolicyRouteAvailabilitySchema, { routeKind: ConnectionRouteKind.DIRECT, available: true, reason: ConnectionPolicyAvailabilityReason.AVAILABLE }),
        create(ConnectionPolicyRouteAvailabilitySchema, { routeKind: ConnectionRouteKind.SSH, available: false, reason: ConnectionPolicyAvailabilityReason.CREDENTIAL_UNAVAILABLE }),
        create(ConnectionPolicyRouteAvailabilitySchema, { routeKind: ConnectionRouteKind.CLOUD, available: true, reason: ConnectionPolicyAvailabilityReason.AVAILABLE }),
      ],
    })
    const client = {
      getConnectionPolicy: vi.fn(async () => state),
      applyConnectionPolicy: vi.fn(async () => state),
    } as unknown as ProtoBindingClient
    const connector = new ProtoBindingConnector(() => client, { endpointId: 'studio' })

    await expect(connector.getConnectionPolicy()).resolves.toEqual({
      policy: { route: 'auto', cloud: 'auto', relayTransport: 'auto' },
      available: { direct: true, ssh: false, cloud: true },
      unavailableReasons: { ssh: 'credential_unavailable' },
    })
    await connector.applyConnectionPolicy({ route: 'cloud', cloud: 'relay', relayTransport: 'tcp' })
    expect(client.applyConnectionPolicy).toHaveBeenCalledWith('studio', expect.objectContaining({
      routePreference: EndpointRoutePreference.MANAGED_CLOUD,
      cloudRelayMode: ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_RELAY_ONLY,
      relayTransport: ManagedWebRTCRelayTransport.MANAGED_WEBRTC_RELAY_TRANSPORT_TCP,
    }), undefined)
  })

  it('opens a Direct session without platform-side policy mutation', async () => {
  const client = {
    getConnectionPolicy: vi.fn(),
    applyConnectionPolicy: vi.fn(),
    openSession: vi.fn(async () => ({ close: vi.fn() })),
  } as unknown as ProtoBindingClient
  const connector = new ProtoBindingConnector(() => client, { endpointId: 'studio', routeId: 'direct' })

  await connector.connect({ machineId: 'studio' })

  expect(client.getConnectionPolicy).not.toHaveBeenCalled()
  expect(client.applyConnectionPolicy).not.toHaveBeenCalled()
  expect(client.openSession).toHaveBeenCalledTimes(1)
  })
})

describe('ProtoBindingClient failed open ownership', () => {
  it('preserves the stable Proto error code for localized user feedback', async () => {
    const backend = new CancellationBackend()
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_SESSION) {
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'openSession', value: create(OpenSessionResultSchema, {
            operationHandle: 1n,
            error: create(ApiErrorSchema, { code: ApiErrorCode.UNAVAILABLE, message: 'sanitized connection failure', retryable: true }),
          }) },
        }))))
        return 1n
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    await expect(client.openSession(create(OpenSessionRequestSchema, { endpointId: 'studio' }))).rejects.toMatchObject({
      code: 'unavailable',
      retryable: true,
    })
    await client.close()
  })

  it.each([
    { protoCode: ApiErrorCode.DAEMON_BLOCKED, code: 'daemon_blocked', retryable: true },
    { protoCode: ApiErrorCode.DAEMON_DELETED, code: 'daemon_deleted', retryable: false },
    { protoCode: ApiErrorCode.RELAY_NOT_IN_PLAN, code: 'relay_not_in_plan', retryable: false },
    { protoCode: ApiErrorCode.RELAY_QUOTA_EXHAUSTED, code: 'relay_quota_exhausted', retryable: false },
    { protoCode: ApiErrorCode.RELAY_CONCURRENCY_EXHAUSTED, code: 'relay_concurrency_exhausted', retryable: false },
    { protoCode: ApiErrorCode.SUBSCRIPTION_INACTIVE, code: 'subscription_inactive', retryable: false },
    { protoCode: ApiErrorCode.RELAY_REGION_UNAVAILABLE, code: 'relay_region_unavailable', retryable: false },
  ])('preserves stable connection error $code', async ({ protoCode, code, retryable }) => {
    const backend = new CancellationBackend()
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_SESSION) {
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'openSession', value: create(OpenSessionResultSchema, {
            operationHandle: 1n,
            error: create(ApiErrorSchema, { code: protoCode, message: code, retryable }),
          }) },
        }))))
        return 1n
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    await expect(client.openSession(create(OpenSessionRequestSchema, { endpointId: 'studio' }))).rejects.toMatchObject({ code, retryable })
    await client.close()
  })

  it('releases failed open operations beyond the engine handle capacity', async () => {
    const backend = new CancellationBackend()
    let next = 0n
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_SESSION) {
        const operationHandle = ++next
        queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
          event: { case: 'openSession', value: create(OpenSessionResultSchema, {
            operationHandle,
            error: create(ApiErrorSchema, { code: ApiErrorCode.UNAVAILABLE, message: 'connect failed' }),
          }) },
        }))))
        return operationHandle
      }
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    const request = create(OpenSessionRequestSchema, { endpointId: 'studio' })
    for (let index = 0; index < 4100; index += 1) {
      await expect(client.openSession(request)).rejects.toThrow('connect failed')
    }
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.released).toHaveLength(4100)
    await client.close()
  })

  it('lets SessionClosedEvent own release for an early successful open that was cancelled', async () => {
  const backend = new CancellationBackend()
  let backendClosed = false
  backend.close = async () => { backendClosed = true }
  backend.request = async (operation, _payload, handle) => {
    if (operation === BindingOperation.OPEN_SESSION) {
    backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'openSession', value: create(OpenSessionResultSchema, {
      operationHandle: 1n,
      sessionHandle: 50n,
      session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
    }) } })))
    return 1n
    }
    if (operation === BindingOperation.CLOSE_SESSION && handle === 50n) {
    queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'sessionClosed', value: create(SessionClosedEventSchema, { sessionHandle: 50n }) } }))))
    }
    if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
    return 0n
  }
  const client = new ProtoBindingClient(backend)
  const controller = new AbortController()
  controller.abort(new DOMException('cancel open', 'AbortError'))
  await expect(client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }), controller.signal)).rejects.toMatchObject({ name: 'AbortError' })
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(backend.released.filter((handle) => handle === 50n)).toHaveLength(1)
  expect(backend.released.filter((handle) => handle === 1n)).toHaveLength(1)
  expect(backendClosed).toBe(false)
  await client.close()
  })
})

describe('ProtoBindingClient resource open cancellation', () => {
  it('returns promptly and closes a late stream handle', async () => {
    const backend = new CancellationBackend()
    let releaseOpen!: (handle: bigint) => void
    const open = new Promise<bigint>((resolve) => { releaseOpen = resolve })
    const closed: bigint[] = []
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_RESOURCE_STREAM) return await open
      if (operation === BindingOperation.CLOSE_RESOURCE_STREAM && handle) closed.push(handle)
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    const controller = new AbortController()
    const pending = client.openResourceStream(1n, create(ResourceHandleSchema, {
      kind: ResourceKind.FILE_TRANSFER,
      opaqueToken: Uint8Array.of(1),
    }), { signal: controller.signal })
    controller.abort(new DOMException('cancel stream open', 'AbortError'))
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'resourceStreamFrame', value: create(ResourceStreamFrameSchema, { streamHandle: 9n, type: ResourceStreamFrameType.FILE_DATA }) },
    })))
    releaseOpen(9n)
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(closed).toEqual([9n])
    expect(backend.released).toEqual([9n])
    backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'resourceStreamFrame', value: create(ResourceStreamFrameSchema, { streamHandle: 9n, type: ResourceStreamFrameType.FILE_DATA }) },
    })))
    expect((client as unknown as { earlyStreamEvents: Map<bigint, unknown> }).earlyStreamEvents.size).toBe(0)
    backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'resourceStreamClosed', value: create(ResourceStreamClosedEventSchema, { streamHandle: 9n }) },
    })))
    expect((client as unknown as { abandonedStreamHandles: Set<bigint> }).abandonedStreamHandles.size).toBe(0)
    await client.close()
  })

  it('reclaims a late stream handle when closed arrived before the handle', async () => {
  const backend = new CancellationBackend()
  let releaseOpen!: (handle: bigint) => void
  const open = new Promise<bigint>((resolve) => { releaseOpen = resolve })
  backend.request = async (operation, _payload, handle) => {
    if (operation === BindingOperation.OPEN_RESOURCE_STREAM) return await open
    if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
    return 0n
  }
  const client = new ProtoBindingClient(backend)
  const controller = new AbortController()
  const pending = client.openResourceStream(1n, create(ResourceHandleSchema, { kind: ResourceKind.FILE_TRANSFER, opaqueToken: Uint8Array.of(1) }), { signal: controller.signal })
  controller.abort(new DOMException('cancel stream open', 'AbortError'))
  await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
  backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
    event: { case: 'resourceStreamClosed', value: create(ResourceStreamClosedEventSchema, { streamHandle: 10n }) },
  })))
  releaseOpen(10n)
  await new Promise((resolve) => setTimeout(resolve, 0))
  const state = client as unknown as { earlyStreamEvents: Map<bigint, unknown>; abandonedStreamHandles: Set<bigint> }
  expect(state.earlyStreamEvents.size).toBe(0)
  expect(state.abandonedStreamHandles.size).toBe(0)
  expect(backend.released).toEqual([10n])
  await client.close()
  })
})

describe('ProtoBindingClient terminal event release ownership', () => {
  it('replays a normal stream close that arrives before the open handle ACK', async () => {
    const backend = new CancellationBackend()
    let releaseOpen!: (handle: bigint) => void
    const open = new Promise<bigint>((resolve) => { releaseOpen = resolve })
    backend.request = async (operation, _payload, handle) => {
      if (operation === BindingOperation.OPEN_RESOURCE_STREAM) return await open
      if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
      return 0n
    }
    const client = new ProtoBindingClient(backend)
    const pending = client.openResourceStream(1n, create(ResourceHandleSchema, { kind: ResourceKind.FILE_TRANSFER, opaqueToken: Uint8Array.of(1) }))
    backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'resourceStreamClosed', value: create(ResourceStreamClosedEventSchema, { streamHandle: 60n }) },
    })))
    releaseOpen(60n)
    const stream = await pending
    await new Promise((resolve) => setTimeout(resolve, 0))
    await expect(stream.send(ResourceStreamFrameType.FILE_DATA, new Uint8Array())).rejects.toThrow('closed')
    expect((client as unknown as { streams: Map<bigint, unknown> }).streams.has(60n)).toBe(false)
    expect(backend.released.filter((handle) => handle === 60n)).toHaveLength(1)
    backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, {
      event: { case: 'resourceStreamClosed', value: create(ResourceStreamClosedEventSchema, { streamHandle: 60n }) },
    })))
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.released.filter((handle) => handle === 60n)).toHaveLength(1)
    await client.close()
  })

  it.each(['before_ack', 'after_ack'] as const)('releases normal session and stream handles exactly once when events arrive %s', async (order) => {
  const backend = new CancellationBackend()
  let nextOperation = 0n
  let backendClosed = false
  backend.close = async () => { backendClosed = true }
  const emit = (payload: Uint8Array) => order === 'before_ack' ? backend.emit(payload) : queueMicrotask(() => backend.emit(payload))
  backend.request = async (operation, _payload, handle) => {
    if (operation === BindingOperation.OPEN_SESSION) {
    const operationHandle = ++nextOperation
    queueMicrotask(() => backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'openSession', value: create(OpenSessionResultSchema, {
      operationHandle,
      sessionHandle: 70n,
      session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
    }) } }))))
    return operationHandle
    }
    if (operation === BindingOperation.OPEN_RESOURCE_STREAM) return 80n
    if (operation === BindingOperation.CLOSE_RESOURCE_STREAM && handle === 80n) {
    emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'resourceStreamClosed', value: create(ResourceStreamClosedEventSchema, { streamHandle: 80n }) } })))
    }
    if (operation === BindingOperation.CLOSE_SESSION && handle === 70n) {
    emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'sessionClosed', value: create(SessionClosedEventSchema, { sessionHandle: 70n }) } })))
    }
    if (operation === BindingOperation.RELEASE && handle) backend.released.push(handle)
    return 0n
  }
  const client = new ProtoBindingClient(backend)
  const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
  const sessionClosed = vi.fn()
  session.subscribeClosed(sessionClosed)
  const stream = await session.openResourceStream(create(ResourceHandleSchema, { kind: ResourceKind.FILE_TRANSFER, opaqueToken: Uint8Array.of(1) }))
  await stream.close()
  await session.close()
  await new Promise((resolve) => setTimeout(resolve, 0))
  backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'resourceStreamClosed', value: create(ResourceStreamClosedEventSchema, { streamHandle: 80n }) } })))
  backend.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'sessionClosed', value: create(SessionClosedEventSchema, { sessionHandle: 70n }) } })))
  await new Promise((resolve) => setTimeout(resolve, 0))
  expect(backend.released.filter((handle) => handle === 80n)).toHaveLength(1)
  expect(backend.released.filter((handle) => handle === 70n)).toHaveLength(1)
  expect(sessionClosed).not.toHaveBeenCalled()
  expect(backendClosed).toBe(false)
  await client.close()
  })
})

describe('ProtoBindingClient late execute cleanup', () => {
  afterEach(() => vi.useRealTimers())

  it('destroys a late FileUploadOpen resource through its resume credential', async () => {
  const backend = new LateFileOpenBackend()
  const client = new ProtoBindingClient(backend)
  const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
  const controller = new AbortController()
  const pending = session.execute(create(CommandEnvelopeSchema, { command: { case: 'fileUploadOpen', value: create(FileUploadOpenCommandSchema, { path: '/tmp/demo', size: 8n, overwrite: true }) } }), { signal: controller.signal })
  controller.abort(new DOMException('cancel upload open', 'AbortError'))
  await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
  backend.emitLateOpen()
  await backend.cleanupObserved
  expect(backend.cleanupCase).toBe('fileTransferCancel')
  await session.close()
  await client.close()
  })

  it.each([
    ['history', 'historyRelease'],
    ['event', 'releaseResource'],
  ] as const)('destroys a late %s resource', async (kind, cleanupCase) => {
    const backend = new LateSessionResourceBackend(kind)
    const client = new ProtoBindingClient(backend)
    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    const controller = new AbortController()
    const command = kind === 'history'
      ? create(CommandEnvelopeSchema, { command: { case: 'historyWindow', value: create(HistoryWindowCommandSchema) } })
      : create(CommandEnvelopeSchema, { command: { case: 'eventSubscribe', value: create(EventSubscribeCommandSchema) } })
    const pending = session.execute(command, { signal: controller.signal })
    controller.abort(new DOMException(`cancel ${kind}`, 'AbortError'))
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    backend.emitLateResult()
    await backend.cleanupObserved
    expect(backend.cleanupCase).toBe(cleanupCase)
    await client.close()
  })

  it('keeps an existing frozen token when a cancelled older page arrives late', async () => {
    const backend = new LateSessionResourceBackend('history')
    const client = new ProtoBindingClient(backend)
    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    const controller = new AbortController()
    const command = create(CommandEnvelopeSchema, { command: { case: 'historyWindow', value: create(HistoryWindowCommandSchema, {
      mode: HistoryWindowMode.OLDER,
      token: 'history-token',
    }) } })
    const pending = session.execute(command, { signal: controller.signal })
    controller.abort(new DOMException('cancel older history page', 'AbortError'))
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    backend.emitLateResult()
    await new Promise((resolve) => setTimeout(resolve, 0))
    expect(backend.cleanupCase).toBe('')
    await client.close()
  })

  it('cleans a FileUploadOpen result emitted before the operation handle returns', async () => {
    const backend = new LateFileOpenBackend('success', true)
    const client = new ProtoBindingClient(backend)
    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    const controller = new AbortController()
    const pending = session.execute(create(CommandEnvelopeSchema, { command: { case: 'fileUploadOpen', value: create(FileUploadOpenCommandSchema, { path: '/tmp/demo', size: 8n, overwrite: true }) } }), { signal: controller.signal })
    controller.abort(new DOMException('cancel upload open', 'AbortError'))
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    await backend.cleanupObserved
    expect(backend.cleanupCase).toBe('fileTransferCancel')
    await client.close()
  })

  it.each(['false', 'reject'] as const)('fails closed when late cleanup is %s', async (mode) => {
    const backend = new LateFileOpenBackend(mode)
    const client = new ProtoBindingClient(backend)
    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    const controller = new AbortController()
    const pending = session.execute(create(CommandEnvelopeSchema, { command: { case: 'fileUploadOpen', value: create(FileUploadOpenCommandSchema, { path: '/tmp/demo', size: 8n, overwrite: true }) } }), { signal: controller.signal })
    controller.abort(new DOMException('cancel upload open', 'AbortError'))
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    backend.emitLateOpen()
    await waitFor(() => backend.closed)
    expect(session.isAlive()).toBe(false)
  })

  it('fails closed when late cleanup times out', async () => {
    vi.useFakeTimers()
    const backend = new LateFileOpenBackend('hang')
    const client = new ProtoBindingClient(backend)
    const session = await client.openSession(create(OpenSessionRequestSchema, { requestId: 'open', endpointId: 'studio' }))
    const controller = new AbortController()
    const pending = session.execute(create(CommandEnvelopeSchema, { command: { case: 'fileUploadOpen', value: create(FileUploadOpenCommandSchema, { path: '/tmp/demo', size: 8n, overwrite: true }) } }), { signal: controller.signal })
    controller.abort(new DOMException('cancel upload open', 'AbortError'))
    await expect(pending).rejects.toMatchObject({ name: 'AbortError' })
    backend.emitLateOpen()
    await vi.advanceTimersByTimeAsync(5_001)
    expect(backend.closed).toBe(true)
    expect(session.isAlive()).toBe(false)
  })
})

class LateFileOpenBackend implements ProtoBindingBackend {
  cleanupCase = ''
  closed = false
  private onEvent: ((payload: Uint8Array) => void) | null = null
  private next = 0n
  private lateOperation = 0n
  private cleanupResolve!: () => void
  readonly cleanupObserved = new Promise<void>((resolve) => { this.cleanupResolve = resolve })

  constructor(private readonly cleanupMode: 'success' | 'false' | 'reject' | 'hang' = 'success', private readonly emitOpenBeforeReturn = false) {}

  start(onEvent: (payload: Uint8Array) => void): void { this.onEvent = onEvent }
  emit(payload: Uint8Array): void { this.onEvent?.(payload) }

  async request(operation: BindingOperationCode, payload: Uint8Array, handle?: bigint): Promise<bigint> {
  if (operation === BindingOperation.OPEN_SESSION) {
    const operationHandle = ++this.next
    queueMicrotask(() => this.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'openSession', value: create(OpenSessionResultSchema, {
    operationHandle, sessionHandle: 20n, session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
    }) } }))))
    return operationHandle
  }
  if (operation === BindingOperation.EXECUTE) {
    const operationHandle = ++this.next
    const command = fromBinary(CommandEnvelopeSchema, payload)
    if (command.command.case === 'fileUploadOpen') {
    this.lateOperation = operationHandle
    if (this.emitOpenBeforeReturn) this.emitLateOpen()
    return operationHandle
    }
    this.cleanupCase = command.command.case
    this.cleanupResolve()
    if (this.cleanupMode === 'reject') throw new Error('cleanup transport failed')
    if (this.cleanupMode === 'hang') return operationHandle
    queueMicrotask(() => this.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'execute', value: create(ExecuteResultSchema, {
    operationHandle, sessionHandle: handle ?? 0n, result: create(ResultEnvelopeSchema, {
      result: { case: 'fileTransferCancel', value: create(FileTransferCancelResultSchema, { cancelled: this.cleanupMode !== 'false' }) },
    }),
    }) } }))))
    return operationHandle
  }
  return 0n
  }

  emitLateOpen(): void {
  this.emit(toBinary(EventEnvelopeSchema, create(EventEnvelopeSchema, { event: { case: 'execute', value: create(ExecuteResultSchema, {
    operationHandle: this.lateOperation,
    sessionHandle: 20n,
    result: create(ResultEnvelopeSchema, { result: { case: 'fileTransferOpen', value: create(FileTransferOpenResultSchema, { transfer: create(FileTransferHandleSchema, {
    resource: create(ResourceHandleSchema, { kind: ResourceKind.FILE_TRANSFER, opaqueToken: Uint8Array.of(1) }),
    resume: create(FileUploadResumeHandleSchema, { opaqueToken: Uint8Array.of(7) }),
    }) }) } }),
  }) } })))
  }

  async close(): Promise<void> { this.closed = true }
}

class LateSessionResourceBackend implements ProtoBindingBackend {
  cleanupCase = ''
  private onEvent: ((payload: Uint8Array) => void) | null = null
  private next = 0n
  private lateOperation = 0n
  private cleanupResolve!: () => void
  readonly cleanupObserved = new Promise<void>((resolve) => { this.cleanupResolve = resolve })

  constructor(private readonly kind: 'history' | 'event') {}

  start(onEvent: (payload: Uint8Array) => void): void { this.onEvent = onEvent }

  async request(operation: BindingOperationCode, payload: Uint8Array, handle?: bigint): Promise<bigint> {
    if (operation === BindingOperation.OPEN_SESSION) {
      const operationHandle = ++this.next
      queueMicrotask(() => this.emit(create(EventEnvelopeSchema, { event: { case: 'openSession', value: create(OpenSessionResultSchema, {
        operationHandle, sessionHandle: 30n, session: create(EndpointSessionStampSchema, { endpointId: 'studio', routeId: 'cloud', generation: 1n }),
      }) } })))
      return operationHandle
    }
    if (operation === BindingOperation.EXECUTE) {
      const operationHandle = ++this.next
      const command = fromBinary(CommandEnvelopeSchema, payload)
      if (command.command.case === 'historyWindow' || command.command.case === 'eventSubscribe') {
        this.lateOperation = operationHandle
        return operationHandle
      }
      this.cleanupCase = command.command.case
      this.cleanupResolve()
      queueMicrotask(() => this.emit(create(EventEnvelopeSchema, { event: { case: 'execute', value: create(ExecuteResultSchema, {
        operationHandle,
        sessionHandle: handle ?? 0n,
        result: create(ResultEnvelopeSchema, { result: { case: 'acknowledge', value: create(AcknowledgeResultSchema) } }),
      }) } })))
      return operationHandle
    }
    return 0n
  }

  emitLateResult(): void {
    const result = this.kind === 'history'
      ? create(ResultEnvelopeSchema, { result: { case: 'historyWindow', value: create(HistoryWindowResultSchema, {
          terminal: create(TerminalRefSchema, { endpointId: 'studio', terminalId: 'term-1' }),
          token: 'history-token',
          historyGeneration: 7n,
        }) } })
      : create(ResultEnvelopeSchema, { result: { case: 'eventSubscription', value: create(EventSubscriptionResultSchema, {
          subscription: create(ResourceHandleSchema, { kind: ResourceKind.SUBSCRIPTION, opaqueToken: Uint8Array.of(9) }),
        }) } })
    this.emit(create(EventEnvelopeSchema, { event: { case: 'execute', value: create(ExecuteResultSchema, {
      operationHandle: this.lateOperation,
      sessionHandle: 30n,
      result,
    }) } }))
  }

  async close(): Promise<void> {}

  private emit(event: EventEnvelope): void {
    this.onEvent?.(toBinary(EventEnvelopeSchema, event))
  }
}

async function waitFor(predicate: () => boolean): Promise<void> {
  const deadline = Date.now() + 1_000
  while (!predicate()) {
    if (Date.now() > deadline) throw new Error('condition was not reached')
    await new Promise((resolve) => setTimeout(resolve, 1))
  }
}
