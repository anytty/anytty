import '../../../generated/proto/apipb/terminal.pb.dart';
import '../../../shared/domain/fuzzy_search.dart';

enum TerminalStatusFilter { running, exited, all }

bool terminalUsesHistoryOnly(TerminalState state) {
  return state == TerminalState.TERMINAL_STATE_EXITED ||
      state == TerminalState.TERMINAL_STATE_REMOVED;
}

final class PublicTerminalTag {
  const PublicTerminalTag({required this.id, required this.label});

  final String id;
  final String label;
}

final class TerminalTagOption {
  const TerminalTagOption({
    required this.id,
    required this.label,
    required this.count,
  });

  final String id;
  final String label;
  final int count;
}

final _positionalTagKey = RegExp(r'^tag\d+$', caseSensitive: false);

List<PublicTerminalTag> publicTerminalTags(TerminalInfo terminal) {
  final labels = <String>{};
  for (final entry in terminal.tags.entries) {
    final key = entry.key.trim();
    final value = entry.value.trim();
    if (key.isEmpty || key == 'cwd' || key.startsWith('anytty.')) continue;
    final label = _positionalTagKey.hasMatch(key) && value.isNotEmpty
        ? value
        : value.isEmpty
        ? key
        : '$key=$value';
    if (label.isNotEmpty) labels.add(label);
  }
  final sorted = labels.toList()..sort();
  return sorted
      .map((label) => PublicTerminalTag(id: label, label: label))
      .toList(growable: false);
}

List<TerminalTagOption> terminalTagOptions(List<TerminalInfo> terminals) {
  final counts = <String, int>{};
  for (final terminal in terminals) {
    for (final tag in publicTerminalTags(terminal)) {
      counts.update(tag.id, (count) => count + 1, ifAbsent: () => 1);
    }
  }
  final labels = counts.keys.toList()..sort();
  return labels
      .map(
        (label) =>
            TerminalTagOption(id: label, label: label, count: counts[label]!),
      )
      .toList(growable: false);
}

List<TerminalInfo> filterTerminals({
  required List<TerminalInfo> terminals,
  required TerminalStatusFilter status,
  Set<String> tagIds = const {},
  String query = '',
}) {
  final needle = query.trim();
  return terminals
      .where((terminal) {
        final matchesStatus = switch (status) {
          TerminalStatusFilter.running =>
            terminal.state == TerminalState.TERMINAL_STATE_CREATED ||
                terminal.state == TerminalState.TERMINAL_STATE_RUNNING,
          TerminalStatusFilter.exited =>
            terminal.state == TerminalState.TERMINAL_STATE_EXITED ||
                terminal.state == TerminalState.TERMINAL_STATE_REMOVED,
          TerminalStatusFilter.all => true,
        };
        if (!matchesStatus) return false;
        if (needle.isNotEmpty && !_terminalMatchesQuery(terminal, needle)) {
          return false;
        }
        if (tagIds.isEmpty) return true;
        final terminalTags = publicTerminalTags(terminal)
            .map((tag) => tag.id)
            .toSet();
        return tagIds.every(terminalTags.contains);
      })
      .toList(growable: false);
}

bool _terminalMatchesQuery(TerminalInfo terminal, String needle) {
  final values = [
    terminal.name,
    terminal.ref.terminalId,
    terminal.ref.endpointId,
    terminal.foregroundProcess,
    terminal.cwd,
    terminal.liveCwd,
    terminal.command.join(' '),
    ...publicTerminalTags(terminal).map((tag) => tag.label),
  ];
  return fuzzyMatchesAny(values, needle);
}

List<TerminalInfo> sortPinnedTerminals(
  List<TerminalInfo> terminals,
  List<String> pinnedIds,
) {
  final rank = <String, int>{
    for (var index = 0; index < pinnedIds.length; index++)
      pinnedIds[index]: index,
  };
  final decorated = terminals.indexed
      .map(
        (entry) => (
          terminal: entry.$2,
          index: entry.$1,
          rank: rank[entry.$2.ref.terminalId],
        ),
      )
      .toList();
  decorated.sort((left, right) {
    if (left.rank != null && right.rank != null) {
      return left.rank!.compareTo(right.rank!);
    }
    if (left.rank != null) return -1;
    if (right.rank != null) return 1;
    return left.index.compareTo(right.index);
  });
  return decorated.map((entry) => entry.terminal).toList(growable: false);
}

List<String> toggleTerminalPin(List<String> pinnedIds, String terminalId) {
  if (pinnedIds.contains(terminalId)) {
    return pinnedIds.where((id) => id != terminalId).toList(growable: false);
  }
  return [...pinnedIds, terminalId];
}

List<String> movePinnedTerminal(
  List<String> pinnedIds,
  String terminalId,
  int delta,
) {
  final current = pinnedIds.indexOf(terminalId);
  final target = current + delta;
  if (current < 0 || target < 0 || target >= pinnedIds.length) {
    return List.unmodifiable(pinnedIds);
  }
  final next = [...pinnedIds];
  final value = next.removeAt(current);
  next.insert(target, value);
  return List.unmodifiable(next);
}
