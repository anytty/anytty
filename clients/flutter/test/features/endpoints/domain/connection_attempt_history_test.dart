import 'package:anytty_native/src/features/endpoints/domain/connection_attempt_history.dart';
import 'package:anytty_native/src/generated/proto/apipb/common.pb.dart';
import 'package:anytty_native/src/generated/proto/bindingpb/client_binding.pb.dart';
import 'package:flutter_test/flutter_test.dart';

EndpointConnectionEvent _event({
  required EndpointConnectionPhase phase,
  required String stage,
  ConnectionRouteKind kind = ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD,
  ApiErrorCode? code,
  String message = '',
}) {
  return EndpointConnectionEvent(
    phase: phase,
    connectionStage: stage,
    attemptedRouteKind: kind,
    error: code == null ? null : (ApiError(message: message)..code = code),
  );
}

void main() {
  group('ConnectionAttemptHistory', () {
    late DateTime now;
    late ConnectionAttemptHistory history;

    setUp(() {
      now = DateTime(2026, 11, 24, 10, 30, 0);
      history = ConnectionAttemptHistory(clock: () => now);
    });

    test('新一轮开始时归档上一轮，而不是丢掉', () {
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      now = now.add(const Duration(milliseconds: 400));
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          stage: 'attempt_starting',
        ),
      );
      now = now.add(const Duration(seconds: 3));
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          stage: 'attempt_failed',
          code: ApiErrorCode.API_ERROR_CODE_UNAUTHORIZED,
          message: 'daemon rejected the binding',
        ),
      );

      // 重试：新的 planning
      now = now.add(const Duration(seconds: 1));
      final startedNew = history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );

      expect(startedNew, isTrue);
      expect(history.attemptIndex, 2);
      expect(history.archives, hasLength(1));
      final archived = history.archives.single;
      expect(archived.index, 1);
      expect(archived.anyFailed, isTrue);
      expect(archived.firstFailure?.errorCode, 'unauthorized');
      expect(archived.firstFailure?.errorMessage, 'daemon rejected the binding');
      expect(archived.duration, const Duration(milliseconds: 3400));
    });

    test('历史有上限，只保留最近 N 轮且最新在前', () {
      final bounded = ConnectionAttemptHistory(limit: 2, clock: () => now);
      for (var attempt = 0; attempt < 5; attempt++) {
        bounded.record(
          _event(
            phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
            stage: 'planning',
          ),
        );
        bounded.record(
          _event(
            phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
            stage: 'attempt_failed',
            code: ApiErrorCode.API_ERROR_CODE_UNAVAILABLE,
          ),
        );
        now = now.add(const Duration(seconds: 1));
      }
      expect(bounded.archives, hasLength(2));
      expect(bounded.archives.first.index, greaterThan(bounded.archives.last.index));
    });

    test('AUTO 竞速里被取消的路由不计入失败历史', () {
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          stage: 'attempt_failed',
          code: ApiErrorCode.API_ERROR_CODE_CANCELLED,
        ),
      );
      expect(history.current, isEmpty);

      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      expect(history.archives, isEmpty, reason: '被取消的竞速路由不应产生历史');
    });

    test('未指定路由的事件被忽略', () {
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          stage: 'attempt_starting',
          kind: ConnectionRouteKind.CONNECTION_ROUTE_KIND_UNSPECIFIED,
        ),
      );
      expect(history.current, isEmpty);
    });

    test('同一轮里同一路由取最新事件', () {
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          stage: 'attempt_starting',
        ),
      );
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_READY,
          stage: 'ready',
        ),
      );
      expect(history.current, hasLength(1));
      expect(
        history.current[ConnectionRouteKind.CONNECTION_ROUTE_KIND_CLOUD]
            ?.connectionStage,
        'ready',
      );
    });

    test('reset 清空全部状态', () {
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_CONNECTING,
          stage: 'attempt_failed',
          code: ApiErrorCode.API_ERROR_CODE_UNAVAILABLE,
        ),
      );
      history.record(
        _event(
          phase: EndpointConnectionPhase.ENDPOINT_CONNECTION_PHASE_PLANNING,
          stage: 'planning',
        ),
      );
      expect(history.archives, isNotEmpty);

      history.reset();

      expect(history.archives, isEmpty);
      expect(history.current, isEmpty);
      expect(history.attemptIndex, 0);
      expect(history.currentStartedAt, isNull);
    });
  });
}
