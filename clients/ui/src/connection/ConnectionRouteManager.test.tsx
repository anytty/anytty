import { create } from '@bufbuild/protobuf'
import { cleanup, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import '../i18n'
import { SSHCredentialProvisionResultSchema } from '../generated/bindingpb/client_binding_pb'
import {
  DirectWebRTCTCPRouteConfigSchema,
  EndpointConfigV1Schema,
  EndpointRouteConfigV1Schema,
  EndpointSource,
  SSHWebRTCTCPRouteConfigSchema,
} from '../generated/remoteauthpb/remote_auth_pb'
import { ConnectionRouteManager, moveRoute, orderedRoutes, removeRoute } from './ConnectionRouteManager'

afterEach(() => cleanup())

describe('ConnectionRouteManager', () => {
  it('reorders routes with stable priorities and keeps at least one route', () => {
    const endpoint = testEndpoint()

    const moved = moveRoute(endpoint, 'ssh', -1)
    expect(orderedRoutes(moved).map((route) => [route.routeId, route.priority])).toEqual([
      ['ssh', 10],
      ['direct', 20],
    ])
    expect(orderedRoutes(moved).every((route) => route.policySource === EndpointSource.USER)).toBe(true)

    const one = removeRoute(moved, 'direct')
    expect(one.routes.map((route) => route.routeId)).toEqual(['ssh'])
    expect(removeRoute(one, 'ssh').routes.map((route) => route.routeId)).toEqual(['ssh'])
  })

  it('tests the selected route through the supplied Go-owned connector', async () => {
    const user = userEvent.setup()
    const endpoint = testEndpoint()
    const test = vi.fn(async () => {})

    render(<ConnectionRouteManager
      endpointId="studio"
      adapter={{
        load: vi.fn(async () => endpoint),
        save: vi.fn(async (value) => value),
        test,
        provisionSSH: vi.fn(async () => { throw new Error('not used') }),
      }}
    />)

    await screen.findByText('Office LAN')
    await user.click(screen.getAllByRole('button', { name: 'Test route' })[0]!)

    await waitFor(() => expect(test).toHaveBeenCalledWith('direct'))
    expect(await screen.findByText('Connection test passed')).toBeTruthy()
  })

  it('shows Cloud as unavailable without exposing an old route action', async () => {
    render(<ConnectionRouteManager
      endpointId="studio"
      adapter={{
        load: vi.fn(async () => testEndpoint()),
        save: vi.fn(async (value) => value),
        test: vi.fn(async () => {}),
        provisionSSH: vi.fn(async () => { throw new Error('not used') }),
      }}
    />)

    expect(await screen.findByText('Cloud unavailable')).toBeTruthy()
    expect(screen.queryByRole('button', { name: 'Add Cloud' })).toBeNull()
  })

  it('localizes a stable unavailable error instead of exposing transport text', async () => {
    const user = userEvent.setup()
    const failure = Object.assign(new Error('dial tcp 192.0.2.10: connection refused'), { code: 'unavailable' })

    render(<ConnectionRouteManager
      endpointId="studio"
      adapter={{
        load: vi.fn(async () => testEndpoint()),
        save: vi.fn(async (value) => value),
        test: vi.fn(async () => { throw failure }),
        provisionSSH: vi.fn(async () => { throw new Error('not used') }),
      }}
    />)

    await screen.findByText('Office LAN')
    await user.click(screen.getAllByRole('button', { name: 'Test route' })[0]!)

    expect((await screen.findByRole('alert')).textContent).toBe('This route is currently unreachable. Check both networks and try again.')
    expect(screen.queryByText(/connection refused/)).toBeNull()
  })

  it('does not expose transport text when an adapter omits a stable error code', async () => {
    const user = userEvent.setup()

    render(<ConnectionRouteManager
      endpointId="studio"
      adapter={{
        load: vi.fn(async () => testEndpoint()),
        save: vi.fn(async (value) => value),
        test: vi.fn(async () => { throw new Error('dial tcp 192.0.2.10: connection refused') }),
        provisionSSH: vi.fn(async () => { throw new Error('not used') }),
      }}
    />)

    await screen.findByText('Office LAN')
    await user.click(screen.getAllByRole('button', { name: 'Test route' })[0]!)

    expect((await screen.findByRole('alert')).textContent).toBe('AnyTTY could not test this route. Try again.')
    expect(screen.queryByText(/192\.0\.2\.10|connection refused/)).toBeNull()
  })

  it('provisions an SSH credential through the platform and shows the authorized key', async () => {
    const user = userEvent.setup()
    const endpoint = testEndpoint()
    endpoint.routes.push(create(EndpointRouteConfigV1Schema, {
      schemaVersion: 1,
      routeId: 'ssh-office',
      displayName: 'Backup SSH',
      priority: 30,
      enabled: true,
      route: {
        case: 'sshWebrtcTcp',
        value: create(SSHWebRTCTCPRouteConfigSchema, {
          host: 'ssh.example.test',
          port: 22,
          user: 'anytty',
          hostKeyFingerprints: ['SHA256:host'],
        }),
      },
    }))
    const provisionSSH = vi.fn(async () => create(SSHCredentialProvisionResultSchema, {
      endpoint,
      authorizedKey: 'ssh-ed25519 AAAATEST anytty',
      keyFingerprint: 'SHA256:client',
    }))

    render(<ConnectionRouteManager
      endpointId="studio"
      adapter={{
        load: vi.fn(async () => endpoint),
        save: vi.fn(async (value) => value),
        test: vi.fn(async () => {}),
        provisionSSH,
      }}
    />)

    await screen.findByText('Backup SSH')
    await user.click(screen.getAllByRole('button', { name: 'Prepare SSH key' })[1]!)

    await waitFor(() => expect(provisionSSH).toHaveBeenCalledWith('ssh-office'))
    expect((await screen.findByLabelText('SSH public key') as HTMLTextAreaElement).value).toBe('ssh-ed25519 AAAATEST anytty')
    expect(screen.getByText('SHA256:client')).toBeTruthy()
  })
})

function testEndpoint() {
  return create(EndpointConfigV1Schema, {
    schemaVersion: 1,
    endpointId: 'studio',
    identity: { deviceId: 'daemon-studio', deviceFingerprint: 'ed25519-sha256:test' },
    routes: [
      create(EndpointRouteConfigV1Schema, {
        schemaVersion: 1,
        routeId: 'direct',
        displayName: 'Office LAN',
        priority: 10,
        enabled: true,
        route: {
          case: 'directWebrtcTcp',
          value: create(DirectWebRTCTCPRouteConfigSchema, {
            signalingAddresses: ['192.0.2.10:41120'],
            iceTcpAddresses: ['192.0.2.10:41121'],
          }),
        },
      }),
      create(EndpointRouteConfigV1Schema, {
        schemaVersion: 1,
        routeId: 'ssh',
        displayName: 'Office SSH',
        priority: 20,
        enabled: true,
        route: {
          case: 'sshWebrtcTcp',
          value: create(SSHWebRTCTCPRouteConfigSchema, {
            host: 'ssh.example.test',
            port: 22,
            user: 'anytty',
            hostKeyFingerprints: ['SHA256:host'],
          }),
        },
      }),
    ],
  })
}
