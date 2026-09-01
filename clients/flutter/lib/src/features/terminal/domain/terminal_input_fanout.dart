Future<void> dispatchTerminalInput<T>(
  Iterable<T> targets,
  Future<void> Function(T target) send,
) async {
  Object? firstError;
  StackTrace? firstStackTrace;
  var accepted = false;
  var attempted = false;
  await Future.wait<void>(
    targets.map((target) async {
      attempted = true;
      try {
        await send(target);
        accepted = true;
      } catch (error, stackTrace) {
        firstError ??= error;
        firstStackTrace ??= stackTrace;
      }
    }),
  );
  if (accepted) return;
  if (!attempted) throw StateError('No terminal input targets are ready');
  Error.throwWithStackTrace(firstError!, firstStackTrace!);
}
