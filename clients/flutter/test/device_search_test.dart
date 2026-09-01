import 'package:anytty_native/src/features/endpoints/domain/device_search.dart';
import 'package:anytty_native/src/generated/proto/remoteauthpb/remote_auth.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  final endpoints = [
    EndpointConfigV1(
      endpointId: 'device-mac-01',
      label: 'Studio Mac',
      platform: 'darwin',
    ),
    EndpointConfigV1(
      endpointId: 'build-linux-02',
      label: 'Build Server',
      platform: 'linux',
    ),
  ];

  test('searches devices by label, id, and platform', () {
    expect(searchEndpoints(endpoints, 'studio'), [endpoints.first]);
    expect(searchEndpoints(endpoints, 'LINUX'), [endpoints.last]);
    expect(searchEndpoints(endpoints, 'mac-01'), [endpoints.first]);
  });

  test('returns all devices for an empty query', () {
    expect(searchEndpoints(endpoints, '  '), endpoints);
  });

  test('matches an ordered partial query across skipped characters', () {
    expect(searchEndpoints(endpoints, 'SdoMc'), [endpoints.first]);
    expect(searchEndpoints(endpoints, 'bux02'), [endpoints.last]);
    expect(searchEndpoints(endpoints, 'McS'), isEmpty);
  });
}
