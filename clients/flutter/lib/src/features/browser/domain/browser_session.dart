import 'dart:convert';

import 'package:flutter/foundation.dart';

enum BrowserSessionPhase { parked, restoring, active, parking, blocked, failed }

@immutable
final class BrowserSessionSnapshot {
  const BrowserSessionSnapshot({
    required this.sessionId,
    required this.endpointId,
    required this.endpointLabel,
    required this.url,
    required this.title,
    required this.scrollX,
    required this.scrollY,
    required this.snapshotPath,
    required this.routeId,
    required this.routeGeneration,
    required this.parkedAt,
  });

  factory BrowserSessionSnapshot.empty({
    required String sessionId,
    required String endpointId,
    required String endpointLabel,
  }) => BrowserSessionSnapshot(
    sessionId: sessionId,
    endpointId: endpointId,
    endpointLabel: endpointLabel,
    url: '',
    title: '',
    scrollX: 0,
    scrollY: 0,
    snapshotPath: null,
    routeId: null,
    routeGeneration: 0,
    parkedAt: null,
  );

  final String sessionId;
  final String endpointId;
  final String endpointLabel;
  final String url;
  final String title;
  final int scrollX;
  final int scrollY;
  final String? snapshotPath;
  final String? routeId;
  final int routeGeneration;
  final DateTime? parkedAt;

  bool get hasSnapshot => snapshotPath?.isNotEmpty == true;

  Uri? get restorableUri {
    final parsed = Uri.tryParse(url.trim());
    if (parsed == null) return null;
    if (parsed.scheme == 'http' ||
        parsed.scheme == 'https' ||
        parsed.scheme == 'about') {
      return parsed;
    }
    return null;
  }

  BrowserSessionSnapshot copyWith({
    String? sessionId,
    String? endpointId,
    String? endpointLabel,
    String? url,
    String? title,
    int? scrollX,
    int? scrollY,
    Object? snapshotPath = _unchanged,
    Object? routeId = _unchanged,
    int? routeGeneration,
    Object? parkedAt = _unchanged,
  }) => BrowserSessionSnapshot(
    sessionId: sessionId ?? this.sessionId,
    endpointId: endpointId ?? this.endpointId,
    endpointLabel: endpointLabel ?? this.endpointLabel,
    url: url ?? this.url,
    title: title ?? this.title,
    scrollX: scrollX ?? this.scrollX,
    scrollY: scrollY ?? this.scrollY,
    snapshotPath: identical(snapshotPath, _unchanged)
        ? this.snapshotPath
        : snapshotPath as String?,
    routeId: identical(routeId, _unchanged) ? this.routeId : routeId as String?,
    routeGeneration: routeGeneration ?? this.routeGeneration,
    parkedAt: identical(parkedAt, _unchanged)
        ? this.parkedAt
        : parkedAt as DateTime?,
  );

  Map<String, Object?> toJson() => {
    'sessionId': sessionId,
    'endpointId': endpointId,
    'endpointLabel': endpointLabel,
    'url': url,
    'title': title,
    'scrollX': scrollX,
    'scrollY': scrollY,
    'snapshotPath': snapshotPath,
    'routeId': routeId,
    'routeGeneration': routeGeneration,
    'parkedAt': parkedAt?.toUtc().toIso8601String(),
  };

  String encode() => jsonEncode(toJson());

  factory BrowserSessionSnapshot.fromJson(Map<String, Object?> json) {
    String stringValue(String key) {
      final value = json[key];
      return value is String ? value.trim() : '';
    }

    String? nullableStringValue(String key) {
      final value = json[key];
      return value is String ? value.trim() : null;
    }

    final sessionId = stringValue('sessionId');
    final endpointId = stringValue('endpointId');
    if (sessionId.isEmpty || endpointId.isEmpty) {
      throw const FormatException('Browser session identity is required');
    }
    final parkedAtText = nullableStringValue('parkedAt');
    return BrowserSessionSnapshot(
      sessionId: sessionId,
      endpointId: endpointId,
      endpointLabel: stringValue('endpointLabel'),
      url: stringValue('url'),
      title: stringValue('title'),
      scrollX: _intValue(json['scrollX']),
      scrollY: _intValue(json['scrollY']),
      snapshotPath: nullableStringValue('snapshotPath'),
      routeId: nullableStringValue('routeId'),
      routeGeneration: _intValue(json['routeGeneration']),
      parkedAt: parkedAtText == null ? null : DateTime.tryParse(parkedAtText),
    );
  }

  factory BrowserSessionSnapshot.decode(String value) =>
      BrowserSessionSnapshot.fromJson(
        (jsonDecode(value) as Map).cast<String, Object?>(),
      );

  static int _intValue(Object? value) {
    if (value is int) return value;
    if (value is num) return value.toInt();
    return int.tryParse('$value') ?? 0;
  }

  @override
  bool operator ==(Object other) =>
      other is BrowserSessionSnapshot && encode() == other.encode();

  @override
  int get hashCode => encode().hashCode;
}

const _unchanged = Object();

@immutable
final class BrowserSessionState {
  const BrowserSessionState({
    required this.phase,
    required this.sessionId,
    required this.operation,
    required this.snapshot,
    required this.error,
  });

  const BrowserSessionState.initial()
    : phase = BrowserSessionPhase.parked,
      sessionId = null,
      operation = 0,
      snapshot = null,
      error = null;

  final BrowserSessionPhase phase;
  final String? sessionId;
  final int operation;
  final BrowserSessionSnapshot? snapshot;
  final String? error;

  BrowserSessionState copyWith({
    BrowserSessionPhase? phase,
    Object? sessionId = _unchanged,
    int? operation,
    Object? snapshot = _unchanged,
    Object? error = _unchanged,
  }) => BrowserSessionState(
    phase: phase ?? this.phase,
    sessionId: identical(sessionId, _unchanged)
        ? this.sessionId
        : sessionId as String?,
    operation: operation ?? this.operation,
    snapshot: identical(snapshot, _unchanged)
        ? this.snapshot
        : snapshot as BrowserSessionSnapshot?,
    error: identical(error, _unchanged) ? this.error : error as String?,
  );
}

final class BrowserSessionStateMachine {
  BrowserSessionState _state = const BrowserSessionState.initial();
  int _nextOperation = 0;

  BrowserSessionState get state => _state;

  int begin(String sessionId) {
    final operation = ++_nextOperation;
    _state = _state.copyWith(
      phase: _state.sessionId == null
          ? BrowserSessionPhase.restoring
          : BrowserSessionPhase.parking,
      sessionId: sessionId,
      operation: operation,
      error: null,
    );
    return operation;
  }

  bool isCurrent(int operation, String sessionId) =>
      _state.operation == operation && _state.sessionId == sessionId;

  void markRestoring(
    int operation,
    String sessionId,
    BrowserSessionSnapshot snapshot,
  ) {
    if (!isCurrent(operation, sessionId)) return;
    _state = _state.copyWith(
      phase: BrowserSessionPhase.restoring,
      snapshot: snapshot,
      error: null,
    );
  }

  void markActive(
    int operation,
    String sessionId,
    BrowserSessionSnapshot snapshot,
  ) {
    if (!isCurrent(operation, sessionId)) return;
    _state = _state.copyWith(
      phase: BrowserSessionPhase.active,
      snapshot: snapshot,
      error: null,
    );
  }

  void markParked(
    int operation,
    String sessionId,
    BrowserSessionSnapshot snapshot,
  ) {
    if (!isCurrent(operation, sessionId)) return;
    _state = _state.copyWith(
      phase: BrowserSessionPhase.parked,
      snapshot: snapshot,
      error: null,
    );
  }

  void markBlocked(int operation, String sessionId, String message) {
    if (!isCurrent(operation, sessionId)) return;
    _state = _state.copyWith(
      phase: BrowserSessionPhase.blocked,
      error: message,
    );
  }

  void markFailed(int operation, String sessionId, Object error) {
    if (!isCurrent(operation, sessionId)) return;
    _state = _state.copyWith(
      phase: BrowserSessionPhase.failed,
      error: '$error',
    );
  }
}
