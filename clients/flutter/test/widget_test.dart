import 'dart:ui' as ui;

import 'package:anytty_native/src/app/anytty_app.dart';
import 'package:anytty_native/src/app/providers.dart';
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/semantics.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  testWidgets('shows the native endpoint registry', (tester) async {
    anyttyRouter.go('/');
    final registry = EndpointRegistryV1(
      schemaVersion: 1,
      defaultEndpointId: 'studio',
      endpoints: [
        EndpointConfigV1(
          endpointId: 'studio',
          label: 'Studio Mac',
          platform: 'darwin',
          enabled: true,
        ),
      ],
    );

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith((ref) async => registry),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Devices'), findsOneWidget);
    expect(find.text('1 saved'), findsNothing);
    expect(find.byKey(const ValueKey('device-header-status')), findsOneWidget);
    expect(find.text('Studio Mac'), findsOneWidget);
    expect(find.textContaining('darwin'), findsOneWidget);
    expect(find.byTooltip('Search devices'), findsOneWidget);
    expect(find.byKey(const ValueKey('device-search-field')), findsNothing);
    expect(find.byTooltip('Download center'), findsOneWidget);

    await tester.tap(find.byTooltip('Search devices'));
    await tester.pump();
    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pump();
    expect(find.byKey(const ValueKey('device-search-field')), findsNothing);
    await tester.tap(find.byTooltip('Search devices'));
    await tester.pump();
    await tester.enterText(
      find.byKey(const ValueKey('device-search-field')),
      'missing',
    );
    await tester.pump();
    expect(find.text('Studio Mac'), findsNothing);
    expect(find.text('No matching devices'), findsOneWidget);
    await tester.enterText(
      find.byKey(const ValueKey('device-search-field')),
      '',
    );
    await tester.pump();
    expect(find.text('Studio Mac'), findsOneWidget);
    await tester.tap(find.byTooltip('Close search'));
    await tester.pump();
    expect(find.byKey(const ValueKey('device-search-field')), findsNothing);

    await tester.tap(find.byTooltip('More actions for Studio Mac'));
    await tester.pumpAndSettle();

    expect(find.text('Device actions'), findsOneWidget);
    expect(find.text('Display name'), findsOneWidget);
    expect(find.text('Default device'), findsOneWidget);
    expect(find.text('Connection'), findsOneWidget);
    expect(find.text('Disconnect'), findsOneWidget);
    expect(find.text('Remove device'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.tap(find.byTooltip('Close device actions'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();

    expect(find.text('Settings'), findsOneWidget);
    expect(find.text('APPEARANCE'), findsOneWidget);
    expect(find.bySemanticsLabel(RegExp('Terminal preview')), findsOneWidget);
    expect(find.text('Light'), findsOneWidget);
    expect(find.text('Dark'), findsOneWidget);
    expect(find.text('System'), findsWidgets);
    expect(
      tester.getSize(find.byTooltip('Decrease font size')).height,
      greaterThanOrEqualTo(44),
    );
    final light = tester.getSemantics(
      find.bySemanticsLabel('Light interface theme'),
    );
    final dark = tester.getSemantics(
      find.bySemanticsLabel('Dark interface theme'),
    );
    final system = tester.getSemantics(
      find.bySemanticsLabel('System interface theme'),
    );
    expect(light.flagsCollection.isButton, isTrue);
    expect(dark.flagsCollection.isButton, isTrue);
    expect(system.flagsCollection.isButton, isTrue);
    expect(light.flagsCollection.isSelected, ui.Tristate.isFalse);
    expect(dark.flagsCollection.isSelected, ui.Tristate.isTrue);
    expect(system.flagsCollection.isSelected, ui.Tristate.isFalse);
    expect(
      tester.getSize(find.bySemanticsLabel('Light interface theme')).height,
      greaterThanOrEqualTo(44),
    );
    expect(
      tester.getSize(find.bySemanticsLabel('System interface theme')).height,
      greaterThanOrEqualTo(44),
    );

    final themeColorsLink = find.byKey(
      const ValueKey('theme-color-settings-link'),
    );
    await tester.ensureVisible(themeColorsLink);
    await tester.pumpAndSettle();
    await tester.tap(themeColorsLink);
    for (
      var frame = 0;
      frame < 10 &&
          find.byKey(const ValueKey('theme-color-plane')).evaluate().isEmpty;
      frame += 1
    ) {
      await tester.pump(const Duration(milliseconds: 100));
    }
    await tester.pump(const Duration(milliseconds: 200));
    expect(find.text('Theme colors'), findsOneWidget);
    expect(find.text('Color direction'), findsOneWidget);
    expect(find.byKey(const ValueKey('theme-color-plane')), findsOneWidget);
    await tester.tap(find.byTooltip('Back to settings'));
    await tester.pump(const Duration(milliseconds: 400));

    final petalMenuLink = find.byKey(
      const ValueKey('petal-menu-settings-link'),
    );
    await tester.ensureVisible(petalMenuLink);
    await tester.pumpAndSettle();
    expect(tester.getSemantics(petalMenuLink).flagsCollection.isButton, isTrue);
    await tester.tap(petalMenuLink);
    await tester.pumpAndSettle();
    expect(find.text('Petal menu'), findsOneWidget);
    expect(find.text('Current menu'), findsOneWidget);
    expect(find.text('Haptic feedback'), findsOneWidget);
    await tester.tap(find.byTooltip('Back to settings'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(
      find.byKey(const ValueKey('terminal-theme-picker')),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('terminal-theme-picker')));
    await tester.pumpAndSettle();
    expect(find.text('Choose terminal theme'), findsOneWidget);
    expect(find.text('DARK THEMES'), findsOneWidget);
    final selectedTheme = tester.getSemantics(
      find.bySemanticsLabel('AnyTTY Dark theme palette preview'),
    );
    expect(selectedTheme.label, 'AnyTTY Dark theme palette preview');
    expect(selectedTheme.flagsCollection.isButton, isTrue);
    expect(selectedTheme.flagsCollection.isSelected, ui.Tristate.isTrue);
    expect(
      find.byKey(const ValueKey('terminal-theme-dracula')),
      findsOneWidget,
    );
    await tester.scrollUntilVisible(
      find.text('LIGHT THEMES'),
      240,
      scrollable: find.byType(Scrollable).last,
    );
    expect(find.text('LIGHT THEMES'), findsOneWidget);
    await tester.tap(find.byTooltip('Close theme picker'));
    await tester.pumpAndSettle();

    await tester.ensureVisible(
      find.byKey(const ValueKey('terminal-font-picker')),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byKey(const ValueKey('terminal-font-picker')));
    await tester.pumpAndSettle();
    expect(find.text('Choose font'), findsOneWidget);
    expect(find.text('JetBrains Mono'), findsWidgets);
    expect(find.text('Fira Code'), findsOneWidget);
    final selectedFont = tester.getSemantics(
      find.bySemanticsLabel('JetBrains Mono font preview'),
    );
    expect(selectedFont.label, 'JetBrains Mono font preview');
    expect(selectedFont.flagsCollection.isButton, isTrue);
    expect(selectedFont.flagsCollection.isSelected, ui.Tristate.isTrue);
    await tester.scrollUntilVisible(
      find.text('System Mono'),
      160,
      scrollable: find.byType(Scrollable).last,
    );
    expect(find.text('System Mono'), findsOneWidget);
    await tester.tap(find.byTooltip('Close font picker'));
    await tester.pumpAndSettle();

    await tester.scrollUntilVisible(
      find.text('BACKGROUND'),
      240,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.pumpAndSettle();

    expect(find.text('BACKGROUND'), findsOneWidget);
    expect(find.text('Background connections'), findsOneWidget);
    expect(find.text('Terminal notifications'), findsOneWidget);
    final keyboardMode = tester.getSemantics(
      find.bySemanticsLabel(RegExp(r'^Keyboard mode')),
    );
    expect(keyboardMode.label, contains('Keyboard mode'));
    expect(keyboardMode.label, contains('Auto'));
    expect(keyboardMode.flagsCollection.isButton, isTrue);
    final inertia = tester.getSemantics(
      find.bySemanticsLabel(RegExp(r'^Scroll inertia')),
    );
    expect(inertia.label, contains('Scroll inertia'));
    SemanticsNode? inertiaControl;
    inertia.visitChildren((child) {
      if (child.flagsCollection.isSlider) inertiaControl = child;
      return true;
    });
    expect(inertiaControl, isNotNull);
    expect(inertiaControl!.value, contains('60'));
    final backgroundConnections = tester.getSemantics(
      find.bySemanticsLabel(RegExp(r'^Background connections')),
    );
    final terminalNotifications = tester.getSemantics(
      find.bySemanticsLabel(RegExp(r'^Terminal notifications')),
    );
    expect(backgroundConnections.flagsCollection.isToggled, ui.Tristate.isTrue);
    expect(
      terminalNotifications.flagsCollection.isToggled,
      ui.Tristate.isFalse,
    );
    expect(tester.takeException(), isNull);

    await tester.tap(find.byTooltip('Back to devices'));
    await tester.pumpAndSettle();
  });

  testWidgets('keeps the first-use layout readable at 320 logical pixels', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    anyttyRouter.go('/');

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith(
            (ref) async => EndpointRegistryV1(schemaVersion: 1),
          ),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(
      find.textContaining(RegExp('No paired devices|暂无已配对设备')),
      findsOneWidget,
    );
    expect(find.textContaining(RegExp('Scan service|扫描服务')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('keeps settings usable at 200 percent text on a narrow phone', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(320, 640);
    tester.view.devicePixelRatio = 1;
    tester.platformDispatcher.textScaleFactorTestValue = 2;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
    anyttyRouter.go('/');

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith(
            (ref) async => EndpointRegistryV1(schemaVersion: 1),
          ),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();

    expect(find.text('Settings'), findsOneWidget);
    expect(find.bySemanticsLabel('System interface theme'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.scrollUntilVisible(
      find.text('BACKGROUND'),
      240,
      scrollable: find.byType(Scrollable).last,
    );
    await tester.pumpAndSettle();
    expect(find.text('Background connections'), findsOneWidget);
    expect(find.text('Terminal notifications'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.tap(find.byTooltip('Back to devices'));
    await tester.pumpAndSettle();
  });

  testWidgets('stops the settings preview when animations are disabled', (
    tester,
  ) async {
    tester.platformDispatcher.accessibilityFeaturesTestValue =
        const FakeAccessibilityFeatures(disableAnimations: true);
    addTearDown(tester.platformDispatcher.clearAccessibilityFeaturesTestValue);
    anyttyRouter.go('/');

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          endpointRegistryProvider.overrideWith(
            (ref) async => EndpointRegistryV1(schemaVersion: 1),
          ),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.byTooltip('Settings'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Increase font size'));
    await tester.pump(const Duration(milliseconds: 350));

    expect(tester.hasRunningAnimations, isFalse);
    expect(tester.takeException(), isNull);

    await tester.tap(find.byTooltip('Back to devices'));
    await tester.pumpAndSettle();
  });

  testWidgets('shows authenticated Cloud presence separately from enablement', (
    tester,
  ) async {
    final endpoint = EndpointConfigV1(
      endpointId: 'cloud-mac',
      label: 'Cloud Mac',
      platform: 'darwin',
      enabled: true,
      routes: [
        EndpointRouteConfigV1(
          routeId: 'cloud',
          enabled: true,
          credentialRef: 'credential:cloud-mac',
          managedWebrtc: ManagedWebRTCRouteConfig(targetDeviceId: 'device-1'),
        ),
      ],
    );
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
          endpointCloudPresenceProvider.overrideWith(
            (ref, endpointId) async => EndpointCloudPresenceGetResult(
              endpointId: endpointId,
              online: true,
              edgeRegion: 'cn-east',
            ),
          ),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Online'), findsOneWidget);
    expect(find.bySemanticsLabel(RegExp('Online')), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('projects missing remote credentials as authorization required', (
    tester,
  ) async {
    final endpoint = EndpointConfigV1(
      endpointId: 'shared-mac',
      label: 'Shared Mac',
      platform: 'darwin',
      enabled: true,
      routes: [
        EndpointRouteConfigV1(
          routeId: 'direct',
          enabled: true,
          directWebrtcTcp: DirectWebRTCTCPRouteConfig(
            signalingAddresses: ['127.0.0.1:41120'],
            iceTcpAddresses: ['127.0.0.1:41121'],
          ),
        ),
      ],
    );
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
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Authorization required'), findsOneWidget);
    expect(
      find.bySemanticsLabel(RegExp('Authorization required')),
      findsOneWidget,
    );

    await tester.tap(find.byTooltip('More actions for Shared Mac'));
    await tester.pumpAndSettle();
    expect(find.text('Authorize device'), findsOneWidget);
    expect(find.text('Fresh pairing required'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testWidgets('uses two terminal columns on a phone landscape viewport', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(844, 390);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    anyttyRouter.go('/');

    final endpoint = EndpointConfigV1(
      endpointId: 'landscape-mac',
      label: 'Landscape Mac',
      platform: 'darwin',
      enabled: true,
      routes: [
        EndpointRouteConfigV1(
          routeId: 'direct',
          enabled: true,
          credentialRef: 'credential:landscape',
          directWebrtcTcp: DirectWebRTCTCPRouteConfig(
            signalingAddresses: ['127.0.0.1:41120'],
            iceTcpAddresses: ['127.0.0.1:41121'],
          ),
        ),
      ],
    );
    final terminals = [
      for (var index = 1; index <= 4; index++)
        TerminalInfo(
          ref: TerminalRef(
            endpointId: endpoint.endpointId,
            terminalId: 'terminal-$index',
          ),
          name: 'Terminal $index',
          state: TerminalState.TERMINAL_STATE_RUNNING,
          cwd: '/workspace/$index',
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
              session: EndpointSessionStamp(endpointId: endpointId),
              snapshot: ConnectionSnapshot(
                connected: true,
                routeId: 'direct',
                routeKind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_DIRECT,
                observedPath:
                    ConnectionObservedPath.CONNECTION_OBSERVED_PATH_DIRECT,
              ),
            );
          }),
        ],
        child: const AnyttyApp(),
      ),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Landscape Mac'));
    await tester.pumpAndSettle();

    final first = tester.getTopLeft(find.text('Terminal 1'));
    final second = tester.getTopLeft(find.text('Terminal 2'));
    final third = tester.getTopLeft(find.text('Terminal 3'));
    final runningFilter = find.bySemanticsLabel('Running, 4 terminals');
    final runningSemantics = tester.getSemantics(runningFilter);
    expect(find.bySemanticsLabel(RegExp('Filter by tags')), findsNothing);
    expect((first.dy - second.dy).abs(), lessThan(1));
    expect(second.dx, greaterThan(first.dx + 200));
    expect(third.dy, greaterThan(first.dy + 60));
    expect(tester.getSize(runningFilter).height, greaterThanOrEqualTo(48));
    expect(runningSemantics.flagsCollection.isButton, isTrue);
    expect(runningSemantics.flagsCollection.isSelected, ui.Tristate.isTrue);
    expect(tester.takeException(), isNull);
  });
}
