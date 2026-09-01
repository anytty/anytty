import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_localizations/flutter_localizations.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/endpoints/domain/route_management.dart';
import '../features/endpoints/presentation/connection_screen.dart';
import '../features/endpoints/presentation/device_list_screen.dart';
import '../features/endpoints/presentation/route_management_screen.dart';
import '../features/settings/presentation/settings_screen.dart';
import '../features/settings/presentation/terminal_petal_menu_settings_screen.dart';
import '../features/settings/presentation/theme_color_settings_screen.dart';
import '../features/terminal/presentation/terminal_workspace_screen.dart';
import '../native/background_platform.dart';
import 'anytty_theme.dart';
import 'app_appearance.dart';
import 'app_language.dart';
import 'anytty_localizations.dart';
import 'providers.dart';
import 'app_color_preferences.dart';

final anyttyRouter = GoRouter(
  routes: [
    GoRoute(
      path: '/',
      pageBuilder: (context, state) =>
          const NoTransitionPage(child: DeviceListScreen()),
    ),
    GoRoute(
      path: '/settings',
      pageBuilder: (context, state) =>
          _utilityPage(state, const SettingsScreen()),
      routes: [
        GoRoute(
          path: 'petal-menu',
          pageBuilder: (context, state) =>
              _utilityPage(state, const TerminalPetalMenuSettingsScreen()),
        ),
        GoRoute(
          path: 'theme-colors',
          pageBuilder: (context, state) =>
              _utilityPage(state, const ThemeColorSettingsScreen()),
        ),
      ],
    ),
    GoRoute(
      path: '/connection/:endpointId',
      pageBuilder: (context, state) => _utilityPage(
        state,
        ConnectionScreen(
          endpointId: state.pathParameters['endpointId']!,
          label: state.uri.queryParameters['label'],
        ),
      ),
    ),
    GoRoute(
      path: '/routes/:endpointId',
      pageBuilder: (context, state) => _utilityPage(
        state,
        RouteManagementScreen(
          endpointId: state.pathParameters['endpointId']!,
          label: state.uri.queryParameters['label'],
        ),
      ),
      routes: [
        GoRoute(
          path: 'edit',
          pageBuilder: (context, state) => _utilityPage(
            state,
            RouteEditorScreen(
              endpointId: state.pathParameters['endpointId']!,
              label: state.uri.queryParameters['label'],
              routeId: state.uri.queryParameters['routeId'],
              newKind: _routeKindFromName(state.uri.queryParameters['kind']),
            ),
          ),
        ),
      ],
    ),
    GoRoute(
      path: '/terminal/:endpointId',
      pageBuilder: (context, state) => _terminalPage(
        state,
        TerminalWorkspaceScreen(
          endpointId: state.pathParameters['endpointId']!,
          label: state.uri.queryParameters['label'],
        ),
      ),
      routes: [
        GoRoute(
          path: ':terminalId',
          pageBuilder: (context, state) => _terminalPage(
            state,
            TerminalWorkspaceScreen(
              endpointId: state.pathParameters['endpointId']!,
              terminalId: state.pathParameters['terminalId']!,
              label: state.uri.queryParameters['label'],
            ),
          ),
        ),
      ],
    ),
  ],
);

EndpointRouteKind? _routeKindFromName(String? value) {
  for (final kind in EndpointRouteKind.values) {
    if (kind.name == value) return kind;
  }
  return null;
}

NoTransitionPage<void> _terminalPage(GoRouterState state, Widget child) =>
    NoTransitionPage<void>(key: state.pageKey, child: child);

CustomTransitionPage<void> _utilityPage(GoRouterState state, Widget child) =>
    CustomTransitionPage<void>(
      key: state.pageKey,
      child: child,
      transitionDuration: AnyttyMotion.quick,
      reverseTransitionDuration: AnyttyMotion.quick,
      transitionsBuilder: (context, animation, secondaryAnimation, child) {
        final curved = CurvedAnimation(
          parent: animation,
          curve: AnyttyMotion.emphasized,
        );
        return FadeTransition(
          opacity: curved,
          child: SlideTransition(
            position: Tween<Offset>(
              begin: const Offset(0.015, 0),
              end: Offset.zero,
            ).animate(curved),
            child: child,
          ),
        );
      },
    );

final class AnyttyApp extends ConsumerStatefulWidget {
  const AnyttyApp({super.key});

  @override
  ConsumerState<AnyttyApp> createState() => _AnyttyAppState();
}

final class _AnyttyAppState extends ConsumerState<AnyttyApp> {
  StreamSubscription<String>? _routeSubscription;

  @override
  void initState() {
    super.initState();
    _routeSubscription = MethodChannelBackgroundPlatform.instance.routes.listen(
      anyttyRouter.go,
    );
  }

  @override
  void dispose() {
    _routeSubscription?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final appearance =
        ref.watch(appAppearanceProvider).valueOrNull ?? AppAppearance.dark;
    final language =
        ref.watch(appLanguageProvider).valueOrNull ?? AppLanguage.system;
    final colors =
        ref.watch(appColorPreferencesProvider).valueOrNull ??
        AppColorPreferences.defaults;
    return MaterialApp.router(
      title: 'AnyTTY',
      debugShowCheckedModeBanner: false,
      theme: anyttyTheme(Brightness.light, colors),
      darkTheme: anyttyTheme(Brightness.dark, colors),
      themeMode: appearance.themeMode,
      locale: language.locale,
      supportedLocales: AnyttyLocalizations.supportedLocales,
      localizationsDelegates: const [
        AnyttyLocalizations.delegate,
        GlobalMaterialLocalizations.delegate,
        GlobalWidgetsLocalizations.delegate,
        GlobalCupertinoLocalizations.delegate,
      ],
      themeAnimationDuration: Duration.zero,
      themeAnimationCurve: AnyttyMotion.emphasized,
      builder: (context, child) => AnnotatedRegion<SystemUiOverlayStyle>(
        value: anyttySystemUiOverlayStyle(
          Theme.of(context).brightness,
          palette: AnyttyPalette.of(context),
        ),
        child: child ?? const SizedBox.shrink(),
      ),
      routerConfig: anyttyRouter,
    );
  }
}
