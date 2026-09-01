import 'package:anytty_native/src/app/anytty_theme.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/features/endpoints/presentation/route_management_screen.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows compact route rows and route-specific actions', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      _app(
        const RouteManagementScreen(endpointId: 'studio', label: 'Studio Mac'),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Routes'), findsOneWidget);
    expect(find.text('Office LAN'), findsOneWidget);
    expect(find.text('Office SSH'), findsOneWidget);
    expect(find.text('AnyTTY Cloud'), findsOneWidget);
    expect(find.byTooltip('Add route'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.tap(find.byTooltip('More actions for Office LAN'));
    await tester.pumpAndSettle();

    expect(find.text('Test route'), findsOneWidget);
    expect(find.text('Edit route'), findsOneWidget);
    expect(find.text('Remove route'), findsOneWidget);
    expect(find.text('Prepare SSH key'), findsNothing);
    expect(tester.takeException(), isNull);
  });

  testWidgets('keeps the split Direct editor usable at 320 pixels', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      _app(
        const RouteEditorScreen(
          endpointId: 'studio',
          label: 'Studio Mac',
          routeId: 'direct',
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Edit route'), findsOneWidget);
    expect(find.text('Route ID'), findsOneWidget);
    expect(find.text('Setup addresses'), findsOneWidget);
    expect(find.text('ICE TCP addresses'), findsOneWidget);
    expect(find.text('Separate setup and ICE ports'), findsOneWidget);

    await tester.scrollUntilVisible(
      find.text('Save route'),
      240,
      scrollable: find.byType(Scrollable).first,
    );
    await tester.pumpAndSettle();

    expect(find.text('Save route'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('keeps the final route visually active and explains the guard', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 844);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith(
            (ref) async => EndpointRegistryV1(
              schemaVersion: 1,
              defaultEndpointId: 'studio',
              endpoints: [_endpoint()..routes.removeRange(1, 3)],
            ),
          ),
        ],
        child: MaterialApp(
          theme: anyttyTheme(Brightness.light),
          home: const RouteManagementScreen(endpointId: 'studio'),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final toggle = tester.widget<Switch>(find.byType(Switch));
    expect(toggle.value, isTrue);
    expect(toggle.onChanged, isNotNull);
    expect(find.byTooltip('Move Office LAN up'), findsNothing);
    expect(find.byTooltip('Move Office LAN down'), findsNothing);

    await tester.tap(find.byType(Switch));
    await tester.pump();

    expect(
      find.text('Keep at least one route enabled for this device'),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });
}

Widget _app(Widget home) => ProviderScope(
  overrides: [
    endpointRegistryProvider.overrideWith(
      (ref) async => EndpointRegistryV1(
        schemaVersion: 1,
        defaultEndpointId: 'studio',
        endpoints: [_endpoint()],
      ),
    ),
  ],
  child: MaterialApp(theme: anyttyTheme(Brightness.light), home: home),
);

EndpointConfigV1 _endpoint() => EndpointConfigV1(
  schemaVersion: 1,
  endpointId: 'studio',
  label: 'Studio Mac',
  routes: [
    EndpointRouteConfigV1(
      schemaVersion: 1,
      routeId: 'direct',
      displayName: 'Office LAN',
      priority: 10,
      enabled: true,
      directWebrtcTcp: DirectWebRTCTCPRouteConfig(
        signalingAddresses: ['192.0.2.10:41120'],
        iceTcpAddresses: ['192.0.2.10:41121'],
      ),
    ),
    EndpointRouteConfigV1(
      schemaVersion: 1,
      routeId: 'ssh',
      displayName: 'Office SSH',
      priority: 20,
      enabled: true,
      sshWebrtcTcp: SSHWebRTCTCPRouteConfig(
        host: 'ssh.example.test',
        port: 22,
        user: 'anytty',
        hostKeyFingerprints: ['SHA256:host'],
      ),
    ),
    EndpointRouteConfigV1(
      schemaVersion: 1,
      routeId: 'cloud',
      displayName: 'AnyTTY Cloud',
      priority: 30,
      enabled: true,
      managedWebrtc: ManagedWebRTCRouteConfig(targetDeviceId: 'daemon-studio'),
    ),
  ],
);
