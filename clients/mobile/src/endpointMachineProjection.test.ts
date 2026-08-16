import { create } from '@bufbuild/protobuf'
import { describe, expect, it } from 'vitest'
import {
  DirectWebRTCTCPRouteConfigSchema,
  EndpointConfigV1Schema,
  EndpointRouteConfigV1Schema,
  ManagedWebRTCRouteConfigSchema,
  SSHWebRTCTCPRouteConfigSchema,
} from '../../ui/src/generated/remoteauthpb/remote_auth_pb'
import { endpointMachineAccessClass } from './endpointMachineProjection'

describe('endpointMachineAccessClass', () => {
  it('projects a managed-only endpoint as Cloud', () => {
    const endpoint = create(EndpointConfigV1Schema, {
      routes: [create(EndpointRouteConfigV1Schema, {
        route: { case: 'managedWebrtc', value: create(ManagedWebRTCRouteConfigSchema) },
      })],
    })

    expect(endpointMachineAccessClass(endpoint)).toBe('cloud')
  })

  it('projects Direct and SSH routes into the direct-access compatibility bucket', () => {
    const endpoint = create(EndpointConfigV1Schema, {
      routes: [
        create(EndpointRouteConfigV1Schema, {
          route: { case: 'directWebrtcTcp', value: create(DirectWebRTCTCPRouteConfigSchema) },
        }),
        create(EndpointRouteConfigV1Schema, {
          route: { case: 'sshWebrtcTcp', value: create(SSHWebRTCTCPRouteConfigSchema) },
        }),
      ],
    })

    expect(endpointMachineAccessClass(endpoint)).toBe('local')
  })

  it('projects mixed routes as local plus Cloud', () => {
    const endpoint = create(EndpointConfigV1Schema, {
      routes: [
        create(EndpointRouteConfigV1Schema, {
          route: { case: 'directWebrtcTcp', value: create(DirectWebRTCTCPRouteConfigSchema) },
        }),
        create(EndpointRouteConfigV1Schema, {
          route: { case: 'managedWebrtc', value: create(ManagedWebRTCRouteConfigSchema) },
        }),
      ],
    })

    expect(endpointMachineAccessClass(endpoint)).toBe('local_cloud')
  })
})
