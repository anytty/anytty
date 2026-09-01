import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';

enum EndpointRouteKind { direct, ssh, cloud, local, unknown }

EndpointRouteKind endpointRouteKind(EndpointRouteConfigV1 route) {
  switch (route.whichRoute()) {
    case EndpointRouteConfigV1_Route.directWebrtcTcp:
      return EndpointRouteKind.direct;
    case EndpointRouteConfigV1_Route.sshWebrtcTcp:
      return EndpointRouteKind.ssh;
    case EndpointRouteConfigV1_Route.managedWebrtc:
      return EndpointRouteKind.cloud;
    case EndpointRouteConfigV1_Route.localUnix:
      return EndpointRouteKind.local;
    case EndpointRouteConfigV1_Route.notSet:
      return EndpointRouteKind.unknown;
  }
}

List<EndpointRouteConfigV1> orderedEndpointRoutes(EndpointConfigV1 endpoint) {
  final routes = endpoint.routes.map((route) => route.deepCopy()).toList();
  routes.sort((left, right) {
    final leftPriority = left.hasPriority() ? left.priority : 0x7fffffff;
    final rightPriority = right.hasPriority() ? right.priority : 0x7fffffff;
    final priority = leftPriority.compareTo(rightPriority);
    return priority != 0 ? priority : left.routeId.compareTo(right.routeId);
  });
  return routes;
}

EndpointConfigV1 moveEndpointRoute(
  EndpointConfigV1 endpoint,
  String routeId,
  int direction,
) {
  final next = endpoint.deepCopy();
  final routes = orderedEndpointRoutes(next);
  final current = routes.indexWhere((route) => route.routeId == routeId);
  final target = current + direction;
  if (current < 0 || target < 0 || target >= routes.length) return next;
  final moving = routes[current];
  routes[current] = routes[target];
  routes[target] = moving;
  _replaceOrderedRoutes(next, routes);
  return next;
}

EndpointConfigV1 replaceEndpointRoute(
  EndpointConfigV1 endpoint,
  EndpointRouteConfigV1 route,
) {
  final next = endpoint.deepCopy();
  final index = next.routes.indexWhere(
    (candidate) => candidate.routeId == route.routeId,
  );
  if (index >= 0) {
    next.routes[index] = route.deepCopy();
    return next;
  }
  final routes = orderedEndpointRoutes(next)..add(route.deepCopy());
  _replaceOrderedRoutes(next, routes);
  return next;
}

EndpointConfigV1 removeEndpointRoute(
  EndpointConfigV1 endpoint,
  String routeId,
) {
  final next = endpoint.deepCopy();
  if (next.routes.length <= 1) return next;
  next.routes.removeWhere((route) => route.routeId == routeId);
  _replaceOrderedRoutes(next, orderedEndpointRoutes(next));
  return next;
}

EndpointRouteConfigV1 newEndpointRoute(
  EndpointConfigV1 endpoint,
  EndpointRouteKind kind,
) {
  if (kind != EndpointRouteKind.direct && kind != EndpointRouteKind.ssh) {
    throw ArgumentError.value(kind, 'kind', 'Only Direct and SSH can be added');
  }
  final routeId = uniqueEndpointRouteId(endpoint, kind.name);
  final route = EndpointRouteConfigV1(
    schemaVersion: 1,
    routeId: routeId,
    displayName: kind == EndpointRouteKind.direct ? 'Direct' : 'SSH',
    enabled: true,
    manualOnly: false,
    priority: (endpoint.routes.length + 1) * 10,
    source: EndpointSource.ENDPOINT_SOURCE_USER,
    policySource: EndpointSource.ENDPOINT_SOURCE_USER,
  );
  if (kind == EndpointRouteKind.direct) {
    route.directWebrtcTcp = DirectWebRTCTCPRouteConfig();
  } else {
    route.sshWebrtcTcp = SSHWebRTCTCPRouteConfig(
      port: 22,
      credentialDescriptor: EndpointCredentialDescriptor(
        descriptorId: '$routeId-private-key',
        kind: EndpointCredentialKind.ENDPOINT_CREDENTIAL_KIND_SSH_PRIVATE_KEY,
        exportable: false,
      ),
    );
  }
  return route;
}

String uniqueEndpointRouteId(EndpointConfigV1 endpoint, String base) {
  final used = endpoint.routes.map((route) => route.routeId).toSet();
  if (!used.contains(base)) return base;
  var suffix = 2;
  while (used.contains('$base-$suffix')) {
    suffix++;
  }
  return '$base-$suffix';
}

String? validateEndpointRoute(
  EndpointConfigV1 endpoint,
  EndpointRouteConfigV1 route, {
  required bool isNew,
}) {
  final id = route.routeId.trim();
  if (!RegExp(r'^[A-Za-z0-9._-]+$').hasMatch(id)) {
    return 'Route ID may contain only letters, numbers, dot, dash, and underscore';
  }
  if (isNew && endpoint.routes.any((candidate) => candidate.routeId == id)) {
    return 'A route with this ID already exists';
  }
  if (route.displayName.trim().length > 128) {
    return 'Display name must be 128 characters or fewer';
  }
  switch (endpointRouteKind(route)) {
    case EndpointRouteKind.direct:
      final direct = route.directWebrtcTcp;
      if (_invalidValues(direct.signalingAddresses) ||
          _invalidValues(direct.iceTcpAddresses)) {
        return 'Direct routes require valid setup and ICE TCP addresses';
      }
      return null;
    case EndpointRouteKind.ssh:
      final ssh = route.sshWebrtcTcp;
      if (ssh.host.trim().isEmpty ||
          ssh.user.trim().isEmpty ||
          ssh.port < 1 ||
          ssh.port > 65535 ||
          _invalidValues(ssh.hostKeyFingerprints) ||
          ssh.remoteSignalingAddress.trim().isEmpty ||
          ssh.remoteIceTcpAddress.trim().isEmpty) {
        return 'SSH routes require host, user, port, a host key fingerprint, and remote AnyTTY addresses';
      }
      return null;
    case EndpointRouteKind.cloud:
    case EndpointRouteKind.local:
      return 'This route is managed by its source and cannot be edited here';
    case EndpointRouteKind.unknown:
      return 'Route type is not supported';
  }
}

List<String> splitRouteValues(String value) {
  final result = <String>[];
  final seen = <String>{};
  for (final item in value.split(RegExp(r'[\n,]'))) {
    final normalized = item.trim();
    if (normalized.isEmpty || !seen.add(normalized)) continue;
    result.add(normalized);
  }
  return result;
}

bool sameRouteValues(List<String> left, List<String> right) =>
    left.length == right.length &&
    List.generate(
      left.length,
      (index) => left[index] == right[index],
    ).every((matches) => matches);

void _replaceOrderedRoutes(
  EndpointConfigV1 endpoint,
  List<EndpointRouteConfigV1> routes,
) {
  endpoint.routes.clear();
  for (var index = 0; index < routes.length; index++) {
    final route = routes[index].deepCopy()
      ..priority = (index + 1) * 10
      ..policySource = EndpointSource.ENDPOINT_SOURCE_USER;
    endpoint.routes.add(route);
  }
}

bool _invalidValues(List<String> values) =>
    values.isEmpty ||
    values.any(
      (value) =>
          value.isEmpty ||
          value != value.trim() ||
          value.runes.any((rune) => rune < 0x20 || rune == 0x7f),
    ) ||
    values.toSet().length != values.length;
