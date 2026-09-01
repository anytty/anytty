enum TerminalSplitDirection { columns, rows }

enum TerminalSplitPlacement { before, after }

sealed class TerminalSplitNode {
  const TerminalSplitNode();
}

final class TerminalPaneNode extends TerminalSplitNode {
  const TerminalPaneNode(this.paneKey);

  final String paneKey;
}

final class TerminalSplitBranch extends TerminalSplitNode {
  const TerminalSplitBranch({
    required this.id,
    required this.direction,
    required this.ratio,
    required this.first,
    required this.second,
  });

  final String id;
  final TerminalSplitDirection direction;
  final double ratio;
  final TerminalSplitNode first;
  final TerminalSplitNode second;

  TerminalSplitBranch copyWith({
    double? ratio,
    TerminalSplitNode? first,
    TerminalSplitNode? second,
  }) {
    return TerminalSplitBranch(
      id: id,
      direction: direction,
      ratio: ratio ?? this.ratio,
      first: first ?? this.first,
      second: second ?? this.second,
    );
  }
}

const primaryTerminalPaneKey = 'primary';
const primaryTerminalPane = TerminalPaneNode(primaryTerminalPaneKey);

String terminalPaneKey(String terminalId) => 'terminal:$terminalId';

String emptyTerminalPaneKey(String splitId) => 'empty:$splitId';

bool isEmptyTerminalPaneKey(String paneKey) => paneKey.startsWith('empty:');

String? terminalIdForPane(String paneKey, String? primaryTerminalId) {
  if (paneKey == primaryTerminalPaneKey) return primaryTerminalId;
  const prefix = 'terminal:';
  return paneKey.startsWith(prefix) ? paneKey.substring(prefix.length) : null;
}

List<String> terminalPaneKeys(TerminalSplitNode root) {
  return switch (root) {
    TerminalPaneNode(:final paneKey) => [paneKey],
    TerminalSplitBranch(:final first, :final second) => [
      ...terminalPaneKeys(first),
      ...terminalPaneKeys(second),
    ],
  };
}

TerminalSplitNode splitTerminalPane({
  required TerminalSplitNode root,
  required String targetPaneKey,
  required String terminalId,
  required TerminalSplitDirection direction,
  required String splitId,
  TerminalSplitPlacement placement = TerminalSplitPlacement.after,
}) {
  final newPaneKey = terminalPaneKey(terminalId);
  if (newPaneKey == targetPaneKey) return root;

  final withoutExisting =
      removeTerminalPane(root, newPaneKey) ?? primaryTerminalPane;
  return _insertTerminalPane(
    root: withoutExisting,
    targetPaneKey: targetPaneKey,
    newPaneKey: newPaneKey,
    direction: direction,
    splitId: splitId,
    placement: placement,
  );
}

TerminalSplitNode splitTerminalPaneEmpty({
  required TerminalSplitNode root,
  required String targetPaneKey,
  required TerminalSplitDirection direction,
  required String splitId,
  TerminalSplitPlacement placement = TerminalSplitPlacement.after,
}) {
  return _insertTerminalPane(
    root: root,
    targetPaneKey: targetPaneKey,
    newPaneKey: emptyTerminalPaneKey(splitId),
    direction: direction,
    splitId: splitId,
    placement: placement,
  );
}

TerminalSplitNode assignTerminalToPane({
  required TerminalSplitNode root,
  required String targetPaneKey,
  required String terminalId,
}) {
  if (!terminalPaneKeys(root).contains(targetPaneKey)) return root;
  final newPaneKey = terminalPaneKey(terminalId);
  if (newPaneKey == targetPaneKey) return root;
  final withoutExisting = removeTerminalPane(root, newPaneKey) ?? root;
  if (!terminalPaneKeys(withoutExisting).contains(targetPaneKey)) return root;
  return _replaceTerminalPane(
    withoutExisting,
    targetPaneKey,
    TerminalPaneNode(newPaneKey),
  );
}

TerminalSplitNode _insertTerminalPane({
  required TerminalSplitNode root,
  required String targetPaneKey,
  required String newPaneKey,
  required TerminalSplitDirection direction,
  required String splitId,
  required TerminalSplitPlacement placement,
}) {
  if (!terminalPaneKeys(root).contains(targetPaneKey) ||
      terminalPaneKeys(root).contains(newPaneKey)) {
    return root;
  }

  final first = placement == TerminalSplitPlacement.before
      ? TerminalPaneNode(newPaneKey)
      : TerminalPaneNode(targetPaneKey);
  final second = placement == TerminalSplitPlacement.before
      ? TerminalPaneNode(targetPaneKey)
      : TerminalPaneNode(newPaneKey);
  return _replaceTerminalPane(
    root,
    targetPaneKey,
    TerminalSplitBranch(
      id: splitId,
      direction: direction,
      ratio: 50,
      first: first,
      second: second,
    ),
  );
}

TerminalSplitNode? removeTerminalPane(TerminalSplitNode root, String paneKey) {
  if (paneKey == primaryTerminalPaneKey) return root;
  return switch (root) {
    TerminalPaneNode() => root.paneKey == paneKey ? null : root,
    TerminalSplitBranch() => _removeTerminalPaneFromBranch(root, paneKey),
  };
}

TerminalSplitNode _removeTerminalPaneFromBranch(
  TerminalSplitBranch root,
  String paneKey,
) {
  final first = removeTerminalPane(root.first, paneKey);
  final second = removeTerminalPane(root.second, paneKey);
  if (first == null) return second!;
  if (second == null) return first;
  if (identical(first, root.first) && identical(second, root.second)) {
    return root;
  }
  return root.copyWith(first: first, second: second);
}

TerminalSplitNode updateTerminalSplitRatio(
  TerminalSplitNode root,
  String splitId,
  double ratio,
) {
  return switch (root) {
    TerminalPaneNode() => root,
    TerminalSplitBranch() when root.id == splitId => root.copyWith(
      ratio: ratio.clamp(20, 80).toDouble(),
    ),
    TerminalSplitBranch() => _updateNestedTerminalSplitRatio(
      root,
      splitId,
      ratio,
    ),
  };
}

TerminalSplitNode _updateNestedTerminalSplitRatio(
  TerminalSplitBranch root,
  String splitId,
  double ratio,
) {
  final first = updateTerminalSplitRatio(root.first, splitId, ratio);
  final second = updateTerminalSplitRatio(root.second, splitId, ratio);
  if (identical(first, root.first) && identical(second, root.second)) {
    return root;
  }
  return root.copyWith(first: first, second: second);
}

TerminalSplitNode _replaceTerminalPane(
  TerminalSplitNode root,
  String paneKey,
  TerminalSplitNode replacement,
) {
  return switch (root) {
    TerminalPaneNode() => root.paneKey == paneKey ? replacement : root,
    TerminalSplitBranch() => _replaceNestedTerminalPane(
      root,
      paneKey,
      replacement,
    ),
  };
}

TerminalSplitNode _replaceNestedTerminalPane(
  TerminalSplitBranch root,
  String paneKey,
  TerminalSplitNode replacement,
) {
  final first = _replaceTerminalPane(root.first, paneKey, replacement);
  final second = _replaceTerminalPane(root.second, paneKey, replacement);
  if (identical(first, root.first) && identical(second, root.second)) {
    return root;
  }
  return root.copyWith(first: first, second: second);
}
