import 'browser_bookmark_store.dart';
import 'browser_history_store.dart';
import 'browser_session_store.dart';

Future<void> clearBrowserDeviceData(String endpointId) async {
  await Future.wait([
    const SharedPreferencesBrowserBookmarkStore().forScope(endpointId).clear(),
    const SharedPreferencesBrowserHistoryStore().forScope(endpointId).clear(),
    const SharedPreferencesBrowserSessionStore().remove(endpointId),
  ]);
}

extension on SharedPreferencesBrowserBookmarkStore {
  SharedPreferencesBrowserBookmarkStore forScope(String endpointId) =>
      SharedPreferencesBrowserBookmarkStore(limit: limit, scope: endpointId);
}

extension on SharedPreferencesBrowserHistoryStore {
  SharedPreferencesBrowserHistoryStore forScope(String endpointId) =>
      SharedPreferencesBrowserHistoryStore(limit: limit, scope: endpointId);
}
