import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../features/endpoints/data/connection_repository.dart';
import '../features/endpoints/data/endpoint_repository.dart';
import '../features/files/presentation/file_transfer_controller.dart';
import '../features/terminal/data/endpoint_session_client.dart';
import '../features/terminal/data/terminal_keyboard_mode_store.dart';
import '../features/terminal/data/terminal_petal_menu_preferences_store.dart';
import '../features/terminal/data/terminal_quick_action_store.dart';
import '../features/terminal/data/terminal_settings_store.dart';
import '../features/terminal/domain/terminal_petal_menu_preferences.dart';
import '../features/terminal/domain/terminal_quick_action.dart';
import '../features/terminal/domain/terminal_settings.dart';
import '../generated/proto/apipb/common.pb.dart';
import '../generated/proto/apipb/terminal.pb.dart';
import '../generated/proto/bindingpb/client_binding.pb.dart';
import '../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../native/anytty_runtime.dart';
import '../native/app_lifecycle_controller.dart';
import '../native/background_platform.dart';
import '../native/background_runtime_controller.dart';
import '../native/binding_operation.dart';
import '../native/endpoint_registry_platform.dart';
import 'app_appearance.dart';
import 'app_appearance_store.dart';
import 'app_color_preferences.dart';
import 'app_color_preferences_store.dart';
import 'app_language.dart';
import 'app_language_store.dart';
import 'background_preferences.dart';
import 'background_preferences_store.dart';
import 'startup_recovery.dart';

final startupDiagnosticsProvider = Provider<StartupDiagnosticsRecorder>(
  (ref) => StartupDiagnosticsRecorder(),
);

final startupLocalResetProvider = Provider<StartupLocalReset>(
  (ref) => const RegistryStartupLocalReset(),
);

final anyttyRuntimeProvider = FutureProvider<AnyttyRuntime>((ref) async {
  final diagnostics = ref.read(startupDiagnosticsProvider);
  diagnostics.beginRuntimeAttempt();
  AnyttyBackgroundRuntimeController? background;
  AnyttyAppLifecycleController? lifecycle;
  AnyttyRuntime? runtime;
  var preferences = BackgroundPreferences.defaults;
  ref.listen<AsyncValue<BackgroundPreferences>>(backgroundPreferencesProvider, (
    _,
    next,
  ) {
    final value = next.valueOrNull;
    if (value == null) return;
    preferences = value;
    background?.setPreferences(value);
  });
  try {
    preferences = await ref.read(backgroundPreferencesProvider.future);
    runtime = await AnyttyRuntime.start(platform: FlutterClientPlatform());
    background = await AnyttyBackgroundRuntimeController.start(
      runtime: runtime,
      platform: MethodChannelBackgroundPlatform.instance,
      preferences: preferences,
    );
    lifecycle = await AnyttyAppLifecycleController.start(
      runtime,
      onForegroundChanged: background.setForeground,
    );
    diagnostics.recordReady(StartupStage.runtime);
    final readyRuntime = runtime;
    final readyBackground = background;
    final readyLifecycle = lifecycle;
    ref.onDispose(() {
      unawaited(() async {
        await readyLifecycle.close();
        await readyBackground.close();
        await readyRuntime.close();
      }());
    });
    return readyRuntime;
  } catch (error, stackTrace) {
    diagnostics.recordFailure(StartupStage.runtime, error);
    try {
      await lifecycle?.close();
    } catch (_) {}
    try {
      await background?.close();
    } catch (_) {}
    try {
      await runtime?.close();
    } catch (_) {}
    Error.throwWithStackTrace(error, stackTrace);
  }
});

final endpointRegistryProvider = FutureProvider<EndpointRegistryV1>((
  ref,
) async {
  final runtime = await ref.watch(anyttyRuntimeProvider.future);
  final diagnostics = ref.read(startupDiagnosticsProvider);
  try {
    final registry = await EndpointRepository(runtime).getRegistry();
    diagnostics.recordReady(StartupStage.registry);
    return registry;
  } catch (error) {
    diagnostics.recordFailure(StartupStage.registry, error);
    rethrow;
  }
});

final endpointCloudPresenceProvider = FutureProvider.autoDispose
    .family<EndpointCloudPresenceGetResult, String>((ref, endpointId) async {
      ref.watch(foregroundResumeRevisionProvider);
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      return EndpointRepository(runtime).getCloudPresence(endpointId);
    });

final connectionPolicyProvider = FutureProvider.autoDispose
    .family<ConnectionPolicyState, String>((ref, endpointId) async {
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      return ConnectionRepository(runtime).getPolicy(endpointId);
    });

typedef EndpointConnectionDiagnostics = ({
  ConnectionSnapshot snapshot,
  EndpointSessionStamp session,
});

final connectionDiagnosticsProvider = FutureProvider.autoDispose
    .family<EndpointConnectionDiagnostics, String>((ref, endpointId) async {
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      final session = await ref.watch(
        endpointSessionProvider(endpointId).future,
      );
      final snapshot = await ConnectionRepository(runtime)
          .getSnapshot(session.sessionHandle);
      return (snapshot: snapshot, session: session.stamp.deepCopy());
    });

final foregroundResumeRevisionProvider = StreamProvider<int>((ref) async* {
  final runtime = await ref.watch(anyttyRuntimeProvider.future);
  yield* runtime.foregroundResumes;
});

final endpointDemandProvider = FutureProvider.autoDispose
    .family<EndpointDemandLease, String>((ref, endpointId) async {
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      final lease = runtime.retainEndpointDemand(endpointId);
      final keepAlive = ref.keepAlive();
      Timer? releaseDelay;
      ref.onCancel(() {
        releaseDelay = Timer(const Duration(seconds: 2), keepAlive.close);
      });
      ref.onResume(() {
        releaseDelay?.cancel();
        releaseDelay = null;
      });
      ref.onDispose(() {
        releaseDelay?.cancel();
        lease.release();
      });
      return lease;
    });

final endpointConnectionProgressProvider = StreamProvider.autoDispose
    .family<EndpointConnectionEvent, String>((ref, endpointId) async* {
      final normalized = endpointId.trim();
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      await ref.watch(endpointDemandProvider(normalized).future);
      yield runtime.endpointConnectionEvent(normalized) ??
          EndpointConnectionEvent(
            endpointId: normalized,
            phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          );
      yield* runtime.events
          .where(
            (event) =>
                event.whichEvent() == EventEnvelope_Event.endpointConnection &&
                event.endpointConnection.endpointId == normalized,
          )
          .map((event) => event.endpointConnection.deepCopy());
    });

final endpointSessionProvider = FutureProvider.autoDispose
    .family<EndpointSessionClient, String>((ref, endpointId) async {
      ref.watch(foregroundResumeRevisionProvider);
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      await ref.watch(endpointDemandProvider(endpointId).future);
      final cancellation = Completer<void>();
      EndpointSessionClient? activeSession;
      var disposed = false;
      ref.onDispose(() {
        disposed = true;
        if (!cancellation.isCompleted) cancellation.complete();
        activeSession?.close();
      });
      final session = await openEndpointSessionWithRetry(
        runtime,
        endpointId,
        cancelWhen: cancellation.future,
      );
      activeSession = session;
      if (disposed) {
        session.close();
        throw const BindingOperationCancelledException();
      }
      unawaited(
        session.closed.then((_) {
          if (!disposed) ref.invalidateSelf();
        }),
      );
      return session;
    });

final fileTransferControllerProvider = Provider<FileTransferController>((ref) {
  final controller = FileTransferController(
    session: (endpointId) =>
        ref.read(endpointSessionProvider(endpointId).future),
  );
  ref.onDispose(controller.dispose);
  return controller;
});

final terminalListProvider = FutureProvider.autoDispose
    .family<List<TerminalInfo>, String>((ref, endpointId) async {
      final session = await ref.watch(
        endpointSessionProvider(endpointId).future,
      );
      return session.listTerminals();
    });

final terminalSettingsProvider =
    AsyncNotifierProvider<TerminalSettingsController, TerminalSettings>(
      TerminalSettingsController.new,
    );

final terminalQuickActionsProvider =
    AsyncNotifierProvider<
      TerminalQuickActionsController,
      List<TerminalQuickAction>
    >(TerminalQuickActionsController.new);

final terminalPetalMenuPreferencesProvider =
    AsyncNotifierProvider<
      TerminalPetalMenuPreferencesController,
      TerminalPetalMenuPreferences
    >(TerminalPetalMenuPreferencesController.new);

final appAppearanceProvider =
    AsyncNotifierProvider<AppAppearanceController, AppAppearance>(
      AppAppearanceController.new,
    );

final appColorPreferencesProvider =
    AsyncNotifierProvider<AppColorPreferencesController, AppColorPreferences>(
      AppColorPreferencesController.new,
    );

final appLanguageProvider =
    AsyncNotifierProvider<AppLanguageController, AppLanguage>(
      AppLanguageController.new,
    );

final backgroundPreferencesProvider =
    AsyncNotifierProvider<
      BackgroundPreferencesController,
      BackgroundPreferences
    >(BackgroundPreferencesController.new);

final class AppAppearanceController extends AsyncNotifier<AppAppearance> {
  final AppAppearanceStore _store = const AppAppearanceStore();
  Future<void> _saveTail = Future.value();
  AppAppearance _persisted = AppAppearance.dark;

  @override
  Future<AppAppearance> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  Future<void> save(AppAppearance next) {
    state = AsyncData(next);
    final completer = Completer<void>();

    Future<void> run() async {
      try {
        _persisted = await _store.save(next);
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.complete();
      } catch (error, stackTrace) {
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

final class AppColorPreferencesController
    extends AsyncNotifier<AppColorPreferences> {
  final AppColorPreferencesStore _store = const AppColorPreferencesStore();
  Future<void> _saveTail = Future.value();
  AppColorPreferences _persisted = AppColorPreferences.defaults;

  @override
  Future<AppColorPreferences> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  void preview(AppColorPreferences next) {
    state = AsyncData(next);
  }

  Future<void> save(AppColorPreferences next) {
    state = AsyncData(next);
    final completer = Completer<void>();

    Future<void> run() async {
      try {
        _persisted = await _store.save(next);
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.complete();
      } catch (error, stackTrace) {
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

final class AppLanguageController extends AsyncNotifier<AppLanguage> {
  final AppLanguageStore _store = const AppLanguageStore();
  Future<void> _saveTail = Future.value();
  AppLanguage _persisted = AppLanguage.system;

  @override
  Future<AppLanguage> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  Future<void> save(AppLanguage next) {
    state = AsyncData(next);
    final completer = Completer<void>();

    Future<void> run() async {
      try {
        _persisted = await _store.save(next);
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.complete();
      } catch (error, stackTrace) {
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

final class BackgroundPreferencesController
    extends AsyncNotifier<BackgroundPreferences> {
  final BackgroundPreferencesStore _store = const BackgroundPreferencesStore();
  Future<void> _saveTail = Future.value();
  BackgroundPreferences _persisted = BackgroundPreferences.defaults;

  @override
  Future<BackgroundPreferences> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  Future<void> save(BackgroundPreferences next) {
    state = AsyncData(next);
    final completer = Completer<void>();

    Future<void> run() async {
      try {
        _persisted = await _store.save(next);
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.complete();
      } catch (error, stackTrace) {
        if (state.valueOrNull == next) state = AsyncData(_persisted);
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

final class TerminalSettingsController extends AsyncNotifier<TerminalSettings> {
  final TerminalSettingsStore _store = const TerminalSettingsStore();
  Future<void> _saveTail = Future.value();
  TerminalSettings _persisted = defaultTerminalSettings;

  @override
  Future<TerminalSettings> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  Future<void> save(TerminalSettings next) {
    final normalized = next.normalized();
    state = AsyncData(normalized);

    final completer = Completer<void>();
    Future<void> run() async {
      try {
        _persisted = await _store.save(normalized);
        if (state.valueOrNull == normalized) state = AsyncData(_persisted);
        completer.complete();
      } catch (error, stackTrace) {
        if (state.valueOrNull == normalized) state = AsyncData(_persisted);
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

final class TerminalQuickActionsController
    extends AsyncNotifier<List<TerminalQuickAction>> {
  final TerminalQuickActionStore _store = const TerminalQuickActionStore();
  Future<void> _saveTail = Future.value();
  List<TerminalQuickAction> _persisted = defaultTerminalQuickActions;

  @override
  Future<List<TerminalQuickAction>> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  Future<void> save(List<TerminalQuickAction> next) {
    final optimistic = List<TerminalQuickAction>.unmodifiable(next);
    state = AsyncData(optimistic);
    final completer = Completer<void>();

    Future<void> run() async {
      try {
        _persisted = await _store.save(optimistic);
        if (identical(state.valueOrNull, optimistic)) {
          state = AsyncData(_persisted);
        }
        completer.complete();
      } catch (error, stackTrace) {
        if (identical(state.valueOrNull, optimistic)) {
          state = AsyncData(_persisted);
        }
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

final class TerminalPetalMenuPreferencesController
    extends AsyncNotifier<TerminalPetalMenuPreferences> {
  final TerminalPetalMenuPreferencesStore _store =
      const TerminalPetalMenuPreferencesStore();
  Future<void> _saveTail = Future.value();
  TerminalPetalMenuPreferences _persisted =
      TerminalPetalMenuPreferences.defaults;

  @override
  Future<TerminalPetalMenuPreferences> build() async {
    _persisted = await _store.load();
    return _persisted;
  }

  Future<void> save(TerminalPetalMenuPreferences next) {
    final normalized = next.normalized();
    state = AsyncData(normalized);
    final completer = Completer<void>();

    Future<void> run() async {
      try {
        _persisted = await _store.save(normalized);
        if (state.valueOrNull == normalized) state = AsyncData(_persisted);
        completer.complete();
      } catch (error, stackTrace) {
        if (state.valueOrNull == normalized) state = AsyncData(_persisted);
        completer.completeError(error, stackTrace);
      }
    }

    _saveTail = _saveTail.then((_) => run(), onError: (_) => run());
    return completer.future;
  }
}

typedef TerminalConnectionKey = ({String endpointId, String terminalId});

final terminalKeyboardModeProvider = FutureProvider.autoDispose
    .family<TerminalKeyboardMode?, TerminalConnectionKey>((ref, key) {
      return const TerminalKeyboardModeStore().load(
        endpointId: key.endpointId,
        terminalId: key.terminalId,
      );
    });

final terminalConnectionProvider = FutureProvider.autoDispose
    .family<TerminalConnection, TerminalConnectionKey>((ref, key) async {
      final session = await ref.watch(
        endpointSessionProvider(key.endpointId).future,
      );
      final connection = await TerminalConnection.open(
        session,
        TerminalRef(endpointId: key.endpointId, terminalId: key.terminalId),
      );
      ref.onDispose(() => unawaited(connection.close()));
      return connection;
    });
