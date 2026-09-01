import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/features/endpoints/presentation/connection_screen.dart';
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:fixnum/fixnum.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows live connection data and efficient policy controls', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          connectionPolicyProvider.overrideWith((ref, endpointId) async {
            return ConnectionPolicyState(
              policy: ConnectionPolicy(
                routePreference:
                    EndpointRoutePreference.ENDPOINT_ROUTE_PREFERENCE_AUTO,
                cloudRelayMode:
                    ManagedWebRTCRelayMode.MANAGED_WEBRTC_RELAY_MODE_AUTO,
                relayTransport: ManagedWebRTCRelayTransport
                    .MANAGED_WEBRTC_RELAY_TRANSPORT_AUTO,
              ),
              routes: [
                ConnectionPolicyRouteAvailability(
                  routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
                  available: true,
                  reason: ConnectionPolicyAvailabilityReason
                      .CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE,
                ),
                ConnectionPolicyRouteAvailability(
                  routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_SSH,
                  available: false,
                  reason: ConnectionPolicyAvailabilityReason
                      .CONNECTION_POLICY_AVAILABILITY_REASON_CREDENTIAL_UNAVAILABLE,
                ),
                ConnectionPolicyRouteAvailability(
                  routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
                  available: true,
                  reason: ConnectionPolicyAvailabilityReason
                      .CONNECTION_POLICY_AVAILABILITY_REASON_AVAILABLE,
                ),
              ],
            );
          }),
          connectionDiagnosticsProvider.overrideWith((ref, endpointId) async {
            return (
              session: EndpointSessionStamp(
                endpointId: endpointId,
                routeId: 'direct-home',
                generation: Int64(7),
              ),
              snapshot: ConnectionSnapshot(
                connected: true,
                routeId: 'direct-home',
                routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
                observedPath:
                    ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT,
                selectionReason: 'preferred route won',
                roundTripNanos: Int64(13400000),
                sampledAtUnixNano: Int64(1760000000000000000),
                localCandidateType: ConnectionCandidateType
                    .CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE,
                remoteCandidateType: ConnectionCandidateType
                    .CONNECTION_CANDIDATE_TYPE_SERVER_REFLEXIVE,
                localProtocol: ConnectionTransport.CONNECTION_TRANSPORT_UDP,
                remoteProtocol: ConnectionTransport.CONNECTION_TRANSPORT_UDP,
                localIp: '203.0.113.8',
                localPort: 40520,
                remoteIp: '203.0.113.8',
                remotePort: 40521,
                bytesSent: Int64(64000),
                bytesReceived: Int64(180000),
                packetsSent: Int64(420),
                lossEvents: Int64(2),
                candidatePairId: 'pair-7',
                networkClass: 'wifi',
              ),
            );
          }),
        ],
        child: MaterialApp(
          theme: anyttyTheme(Brightness.light),
          home: const ConnectionScreen(
            endpointId: 'studio',
            label: 'Studio Mac',
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Connection'), findsOneWidget);
    expect(find.text('Direct'), findsNWidgets(2));
    expect(find.text('13 ms'), findsOneWidget);
    expect(find.text('Required credential is unavailable'), findsOneWidget);
    expect(find.text('Apply and reconnect'), findsOneWidget);
    expect(
      tester
          .getSize(find.widgetWithText(FilledButton, 'Apply and reconnect'))
          .height,
      greaterThanOrEqualTo(48),
    );
    expect(tester.takeException(), isNull);

    await tester.scrollUntilVisible(
      find.byKey(const ValueKey('connection-diagnostics')),
      280,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.tap(find.byKey(const ValueKey('connection-diagnostics')));
    await tester.pumpAndSettle();

    expect(find.text('Candidate pair'), findsOneWidget);
    expect(find.text('pair-7'), findsOneWidget);
    expect(find.text('Generation'), findsOneWidget);
    expect(find.text('7'), findsOneWidget);
    expect(
      tester.getSize(find.byTooltip('Copy Candidate pair')).height,
      greaterThanOrEqualTo(48),
    );
    expect(find.text('Copy redacted report'), findsOneWidget);
    expect(
      tester
          .getSize(find.byKey(const ValueKey('copy-redacted-diagnostics')))
          .height,
      greaterThanOrEqualTo(48),
    );
    expect(tester.takeException(), isNull);
  });
}
