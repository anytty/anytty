import type { EndpointConfigV1 } from '../../ui/src/generated/remoteauthpb/remote_auth_pb'
import type { MachineAccessClass } from '@anytty/ui'

/** Projects configured route capabilities without treating a saved endpoint as local by default. */
export function endpointMachineAccessClass(
  endpoint: Pick<EndpointConfigV1, 'routes'>,
): MachineAccessClass {
  let hasDirectAccessRoute = false
  let hasCloudRoute = false
  for (const route of endpoint.routes) {
    if (route.route.case === 'managedWebrtc') hasCloudRoute = true
    else if (route.route.case !== undefined) hasDirectAccessRoute = true
  }
  if (hasDirectAccessRoute && hasCloudRoute) return 'local_cloud'
  if (hasCloudRoute) return 'cloud'
  return 'local'
}
