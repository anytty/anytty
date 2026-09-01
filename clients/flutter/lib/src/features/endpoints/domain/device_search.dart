import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../../shared/domain/fuzzy_search.dart';

List<EndpointConfigV1> searchEndpoints(
  List<EndpointConfigV1> endpoints,
  String query,
) {
  final needle = query.trim();
  if (needle.isEmpty) return List.unmodifiable(endpoints);
  return endpoints
      .where((endpoint) {
        final values = [endpoint.label, endpoint.endpointId, endpoint.platform];
        return fuzzyMatchesAny(values, needle);
      })
      .toList(growable: false);
}
