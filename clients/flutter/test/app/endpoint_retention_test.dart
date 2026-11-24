import 'package:anytty_native/src/app/endpoint_retention.dart';
import 'package:anytty_native/src/native/anytty_runtime.dart'
    show EndpointDemandLease, EndpointDemandRetainer;
import 'package:flutter_test/flutter_test.dart';

final class _RecordingRetainer implements EndpointDemandRetainer {
  final Map<String, int> retainCalls = {};
  final Set<String> released = {};

  @override
  EndpointDemandLease retainEndpointDemand(String endpointId) {
    retainCalls[endpointId] = (retainCalls[endpointId] ?? 0) + 1;
    return EndpointDemandLease(() {
      released.add(endpointId);
    });
  }
}

void main() {
  group('EndpointRetentionController', () {
    test('retain 是幂等的：同一 endpoint 只向引擎申请一次租约', () {
      final retainer = _RecordingRetainer();
      final controller = EndpointRetentionController(retainer);

      controller.retain('device-a');
      controller.retain('device-a');
      controller.retain(' device-a ');

      expect(retainer.retainCalls['device-a'], 1);
      expect(controller.retainedEndpointIds, ['device-a']);
      expect(retainer.released, isEmpty);
    });

    test('切换机器时旧连接保留，release 后才归还租约', () {
      final retainer = _RecordingRetainer();
      final controller = EndpointRetentionController(retainer);

      controller.retain('device-a');
      controller.retain('device-b');

      expect(controller.retainedEndpointIds, ['device-a', 'device-b']);
      expect(retainer.released, isEmpty);

      // 用户显式断开 device-a：只有它的租约被归还。
      controller.release('device-a');
      expect(controller.retainedEndpointIds, ['device-b']);
      expect(retainer.released, {'device-a'});

      // 未保留的 endpointId 是空操作。
      controller.release('device-zzz');
      expect(controller.retainedEndpointIds, ['device-b']);
    });

    test('releaseAll 归还全部租约且之后不再接受新租约', () {
      final retainer = _RecordingRetainer();
      final controller = EndpointRetentionController(retainer);
      controller.retain('device-a');
      controller.retain('device-b');

      controller.releaseAll();

      expect(retainer.released, {'device-a', 'device-b'});
      expect(controller.retainedEndpointIds, isEmpty);

      controller.retain('device-c');
      expect(retainer.retainCalls.containsKey('device-c'), false);
    });

    test('空白 endpointId 被忽略', () {
      final retainer = _RecordingRetainer();
      final controller = EndpointRetentionController(retainer);

      controller.retain('');
      controller.retain('   ');

      expect(retainer.retainCalls, isEmpty);
      expect(controller.retainedEndpointIds, isEmpty);
    });
  });
}
