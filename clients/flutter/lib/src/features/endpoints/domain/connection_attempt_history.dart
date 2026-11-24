import '../../../generated/proto/apipb/common.pbenum.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';

/// 一轮连接尝试里，单条路由的最终状态。
final class ConnectionRouteSnapshot {
  const ConnectionRouteSnapshot({
    required this.kind,
    required this.stage,
    required this.failed,
    this.errorCode,
    this.errorMessage,
  });

  final ConnectionRouteKind kind;
  final String stage;
  final bool failed;
  final String? errorCode;
  final String? errorMessage;

  factory ConnectionRouteSnapshot.fromEvent(EndpointConnectionEvent event) {
    final error = event.hasError() ? event.error : null;
    return ConnectionRouteSnapshot(
      kind: event.attemptedRouteKind,
      stage: event.connectionStage,
      failed: event.connectionStage == 'attempt_failed',
      errorCode: error == null ? null : _errorCodeName(error.code),
      errorMessage: error == null || error.message.trim().isEmpty
          ? null
          : error.message.trim(),
    );
  }
}

String? _errorCodeName(ApiErrorCode code) {
  final name = code.name;
  if (name.isEmpty) return null;
  return name.replaceFirst('API_ERROR_CODE_', '').toLowerCase();
}

/// 一轮已经结束的连接尝试（被新一轮取代时归档）。
final class ConnectionAttemptSnapshot {
  const ConnectionAttemptSnapshot({
    required this.index,
    required this.startedAt,
    required this.endedAt,
    required this.routes,
  });

  final int index;
  final DateTime startedAt;
  final DateTime endedAt;
  final List<ConnectionRouteSnapshot> routes;

  Duration get duration {
    final value = endedAt.difference(startedAt);
    return value.isNegative ? Duration.zero : value;
  }

  bool get anyFailed => routes.any((route) => route.failed);

  /// 第一条带错误码的路由失败原因，用于折叠态摘要。
  ConnectionRouteSnapshot? get firstFailure =>
      routes.where((route) => route.failed).firstOrNull;

  /// 失败原因里最有用的一行：优先带错误码/消息的。
  String? get failureDetail {
    for (final route in routes) {
      final detail = route.errorMessage ?? route.errorCode;
      if (route.failed && detail != null) return detail;
    }
    for (final route in routes) {
      final detail = route.errorMessage ?? route.errorCode;
      if (detail != null) return detail;
    }
    return null;
  }
}

/// 判断这条事件是否只是 AUTO 竞速里被取消的慢路由。
///
/// 竞速收敛不是用户可见的失败，不应该打断或污染历史。
bool isSupersededConnectionAttempt(EndpointConnectionEvent event) {
  return event.connectionStage == 'attempt_failed' &&
      event.hasError() &&
      event.error.code == ApiErrorCode.API_ERROR_CODE_CANCELLED;
}

/// 有界的连接尝试历史。
///
/// 引擎在连不上时会反复重试，每轮都从 PLANNING 重新开始。历史保留最近
/// [limit] 轮已完成尝试的每条路由结论（阶段、错误码、错误消息），让用户
/// 在重试之后仍然能看到“上一轮到底为什么失败”。
final class ConnectionAttemptHistory {
  ConnectionAttemptHistory({this.limit = 8, DateTime Function()? clock})
    : assert(limit > 0),
      _clock = clock ?? DateTime.now;

  /// 最多保留多少轮历史（不含当前轮）。
  final int limit;
  final DateTime Function() _clock;

  final List<ConnectionAttemptSnapshot> _archives = [];
  final Map<ConnectionRouteKind, EndpointConnectionEvent> _current = {};
  int _index = 0;
  DateTime? _startedAt;
  DateTime? _lastEventAt;

  /// 已归档的尝试，最新的一轮在前。
  List<ConnectionAttemptSnapshot> get archives =>
      List.unmodifiable(_archives.reversed);

  /// 当前轮次的路由事件，按到达顺序。
  Map<ConnectionRouteKind, EndpointConnectionEvent> get current =>
      Map.unmodifiable(_current);

  /// 当前是第几轮（从 1 开始）。
  int get attemptIndex => _index;

  DateTime? get currentStartedAt => _startedAt;

  /// 记录一条连接事件。返回 true 表示这条事件开启了一个新的轮次。
  bool record(EndpointConnectionEvent event) {
    if (event.phase ==
        EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING) {
      _beginAttempt();
      return true;
    }
    // AUTO 竞速中被取消的慢路由不是失败，不进当前轮也不进历史。
    if (isSupersededConnectionAttempt(event)) return false;
    if (event.attemptedRouteKind ==
        ConnectionRouteKind.CONNECTION_ROUTE_KIND_UNSPECIFIED) {
      return false;
    }
    final now = _clock();
    _startedAt ??= now;
    _lastEventAt = now;
    _current[event.attemptedRouteKind] = event.deepCopy();
    return false;
  }

  /// 切换 endpoint 或重新进入页面时清空。
  void reset() {
    _archives.clear();
    _current.clear();
    _index = 0;
    _startedAt = null;
    _lastEventAt = null;
  }

  void _beginAttempt() {
    final now = _clock();
    if (_current.isNotEmpty) {
      _archives.add(
        ConnectionAttemptSnapshot(
          index: _index,
          startedAt: _startedAt ?? now,
          // 用最后一条事件的时间，而不是重试开始的时间。
          endedAt: _lastEventAt ?? now,
          routes: [
            for (final event in _current.values)
              ConnectionRouteSnapshot.fromEvent(event),
          ],
        ),
      );
      if (_archives.length > limit) {
        _archives.removeRange(0, _archives.length - limit);
      }
      _current.clear();
    }
    _index += 1;
    _startedAt = now;
    _lastEventAt = null;
  }
}
