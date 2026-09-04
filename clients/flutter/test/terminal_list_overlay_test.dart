import 'dart:async';

import 'package:anytty_native/src/app/anytty_app.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows the live connection phase while terminals are loading', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(393, 852);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    anyttyRouter.go('/');

    final endpoint = EndpointConfigV1(
      endpointId: 'loading-test',
      label: 'Loading test',
      enabled: true,
    );
    final inventory = Completer<List<TerminalInfo>>();
    addTearDown(() {
      if (!inventory.isCompleted) inventory.complete(const []);
    });

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith(
            (ref) async =>
                EndpointRegistryV1(schemaVersion: 1, endpoints: [endpoint]),
          ),
          terminalListProvider.overrideWith(
            (ref, endpointId) => inventory.future,
          ),
          endpointConnectionProgressProvider.overrideWith(
            (ref, endpointId) => Stream.value(
              EndpointConnectionEvent(
                endpointId: endpointId,
                phase: EndpointConnectionPhase
                    .ENDPOINT_CONNECTION_PHASE_CONNECTING,
              ),
            ),
          ),
          connectionPolicyProvider.overrideWith(
            (ref, endpointId) async => ConnectionPolicyState(
              policy: ConnectionPolicy(
                routePreference:
                    EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO,
              ),
            ),
          ),
          connectionDiagnosticsProvider.overrideWith((ref, endpointId) async {
            return (
              session: EndpointSessionStamp(endpointId: endpointId),
              snapshot: ConnectionSnapshot(connected: false),
            );
          }),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Loading test'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(find.text('Connecting to device'), findsOneWidget);
    expect(find.text('ICE negotiation'), findsNothing);
    expect(find.text('Load terminal list'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets(
    'explains a stalled direct-only connection and links to its settings',
    (tester) async {
      tester.view.physicalSize = const Size(393, 852);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      anyttyRouter.go('/');

      final endpoint = EndpointConfigV1(
        endpointId: 'direct-only-test',
        label: 'Direct only test',
        enabled: true,
      );
      final inventory = Completer<List<TerminalInfo>>();
      final connectionProgress = StreamController<EndpointConnectionEvent>();
      addTearDown(() {
        if (!inventory.isCompleted) inventory.complete(const []);
        unawaited(connectionProgress.close());
      });
      final directOnlyPolicy = ConnectionPolicyState(
        policy: ConnectionPolicy(
          routePreference:
              EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_DIRECT,
        ),
        routes: [
          ConnectionPolicyRouteAvailability(
            routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
            available: true,
          ),
          ConnectionPolicyRouteAvailability(
            routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
            available: true,
          ),
        ],
      );

      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            endpointRegistryProvider.overrideWith(
              (ref) async =>
                  EndpointRegistryV1(schemaVersion: 1, endpoints: [endpoint]),
            ),
            terminalListProvider.overrideWith(
              (ref, endpointId) => inventory.future,
            ),
            endpointConnectionProgressProvider.overrideWith(
              (ref, endpointId) => connectionProgress.stream,
            ),
            connectionPolicyProvider.overrideWith(
              (ref, endpointId) async => directOnlyPolicy,
            ),
            connectionDiagnosticsProvider.overrideWith((ref, endpointId) async {
              return (
                session: EndpointSessionStamp(endpointId: endpointId),
                snapshot: ConnectionSnapshot(connected: false),
              );
            }),
          ],
          child: const AnyttyApp(),
        ),
      );
      await tester.pumpAndSettle();
      await tester.tap(find.text('Direct only test'));
      await tester.pump();
      connectionProgress.add(
        EndpointConnectionEvent(
          endpointId: endpoint.endpointId,
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_SIGNALING,
          attemptedRouteKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
          connectionStage: 'signaling',
        ),
      );
      await tester.pump(const Duration(milliseconds: 100));

      expect(find.text('Currently using Direct mode'), findsOneWidget);
      expect(find.text('Direct · exchanging signaling data'), findsOneWidget);

      connectionProgress.add(
        EndpointConnectionEvent(
          endpointId: endpoint.endpointId,
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          attemptedRouteKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
          connectionStage: 'attempt_failed',
          error: ApiError(message: 'direct route unavailable'),
        ),
      );
      await tester.pump(const Duration(milliseconds: 300));

      expect(find.text('Direct connection unavailable'), findsOneWidget);
      expect(find.text('Use Automatic'), findsOneWidget);
      expect(find.text('Connection settings'), findsOneWidget);

      await tester.tap(find.text('Connection settings'));
      await tester.pumpAndSettle();

      expect(find.text('Connection'), findsOneWidget);
      expect(find.text('ROUTE PREFERENCE'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );

  testWidgets('keeps the terminal list visible beneath its overlays', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(393, 852);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    anyttyRouter.go('/');

    final endpoint = EndpointConfigV1(
      endpointId: 'overlay-test',
      label: 'Overlay test',
      platform: 'darwin',
      enabled: true,
    );
    final terminals = [
      TerminalInfo(
        ref: TerminalRef(endpointId: endpoint.endpointId, terminalId: 'build'),
        name: 'Build terminal',
        state: TerminalState.TERMINAL_STATE_RUNNING,
        cwd: '/workspace/build',
        lastOutputAtUnixNano: Int64(
          (DateTime.now().toUtc().microsecondsSinceEpoch - 12 * 1000000) * 1000,
        ),
        tags: const {'tag1': 'release'}.entries,
        resources: TerminalResourceUsage(
          pid: 4242,
          cpuPercentX100: 420,
          memoryBytes: Int64(224 * 1024 * 1024),
          sampledAtUnixNano: Int64(3),
        ),
      ),
      TerminalInfo(
        ref: TerminalRef(
          endpointId: endpoint.endpointId,
          terminalId: 'observer',
        ),
        name: 'Background observer',
        state: TerminalState.TERMINAL_STATE_RUNNING,
        cwd: '/workspace/observer',
      ),
    ];

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith(
            (ref) async => EndpointRegistryV1(
              schemaVersion: 1,
              defaultEndpointId: endpoint.endpointId,
              endpoints: [endpoint],
            ),
          ),
          terminalListProvider.overrideWith(
            (ref, endpointId) async => terminals,
          ),
          connectionDiagnosticsProvider.overrideWith((ref, endpointId) async {
            return (
              session: EndpointSessionStamp(
                endpointId: endpointId,
                routeId: 'direct-home',
                generation: Int64(7),
              ),
              snapshot: ConnectionSnapshot(
                connected: true,
                routeId: 'cloud-relay',
                routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
                observedPath: ConnectionObservedPath
                    .CONNECTION_OBSERVED_PATH_SINGLE_RELAY,
                roundTripNanos: Int64(12800000),
                localCandidateType:
                    ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_RELAY,
                remoteCandidateType:
                    ConnectionCandidateType.CONNECTION_CANDIDATE_TYPE_RELAY,
                localProtocol: ConnectionTransport.CONNECTION_TRANSPORT_UDP,
                remoteProtocol: ConnectionTransport.CONNECTION_TRANSPORT_UDP,
                relayTransport: ConnectionTransport.CONNECTION_TRANSPORT_TCP,
                localIp: '203.0.113.8',
                localPort: 40520,
                remoteIp: '203.0.113.9',
                remotePort: 40521,
                bytesSent: Int64(64000),
                bytesReceived: Int64(180000),
                networkClass: 'wifi',
              ),
            );
          }),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Overlay test'));
    await tester.pumpAndSettle();

    expect(find.text('12s'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('terminal-resource-strip-cpu')),
      findsNothing,
    );
    expect(
      find.byKey(const ValueKey('terminal-resource-sparkline-cpu-build')),
      findsOneWidget,
    );
    expect(find.byTooltip('Files'), findsOneWidget);
    final filesButton = find.widgetWithIcon(IconButton, Icons.folder_outlined);
    expect(tester.widget<IconButton>(filesButton).onPressed, isNotNull);
    await tester.enterText(
      find.byKey(const ValueKey('terminal-list-search-field')),
      'observer',
    );
    await tester.pump();
    expect(find.text('Build terminal'), findsNothing);
    expect(find.text('Background observer'), findsOneWidget);
    await tester.enterText(
      find.byKey(const ValueKey('terminal-list-search-field')),
      '',
    );
    await tester.pump();
    expect(find.text('Relay'), findsOneWidget);
    await tester.tap(find.byKey(const ValueKey('terminal-network-status')));
    await tester.pumpAndSettle();
    expect(find.text('Network status'), findsOneWidget);
    expect(find.text('Connection type'), findsOneWidget);
    expect(find.text('TCP'), findsOneWidget);
    expect(find.text('ICE transport'), findsOneWidget);
    expect(find.text('Local candidate'), findsOneWidget);
    await tester.drag(find.byType(ListView).last, const Offset(0, -260));
    await tester.pumpAndSettle();
    expect(find.text('Traffic'), findsOneWidget);
    await tester.tap(find.byTooltip('Close'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Terminal actions').first);
    await tester.pumpAndSettle();

    expect(find.text('Background observer'), findsOneWidget);
    expect(find.text('Rename'), findsOneWidget);
    expect(tester.getTopLeft(find.text('Rename')).dx, greaterThan(120));
    expect(tester.takeException(), isNull);

    await tester.tapAt(const Offset(12, 760));
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('resources-build')));
    await tester.pumpAndSettle();

    expect(find.text('Background observer'), findsOneWidget);
    expect(find.text('Resource snapshots · 1 sample'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.tap(find.byTooltip('Close resource details'));
    await tester.pumpAndSettle();
    await tester.tap(find.bySemanticsLabel(RegExp('Filter by tags')));
    await tester.pumpAndSettle();

    expect(find.text('Background observer'), findsOneWidget);
    expect(find.text('Terminal tags'), findsOneWidget);
    expect(find.text('release'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
