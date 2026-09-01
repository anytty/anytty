const terminalPetalMaximumDepth = 2;

const terminalPetalActionIds = <String>[
  'history',
  'search',
  'selection',
  'paste',
  'quick-keys',
  'keyboard',
  'enter',
  'escape',
  'resources',
  'more',
  'command-bar',
  'copy-screen',
  'files',
  'input-tools',
  'tab',
  'backspace',
  'delete',
  'interrupt',
  'eof',
  'suspend',
  'clear',
  'navigation-tools',
  'arrow-left',
  'arrow-down',
  'arrow-up',
  'arrow-right',
  'home',
  'end',
  'page-up',
  'page-down',
  'session-tools',
  'split',
  'split-rows',
  'split-columns',
  'sync-input',
  'resize',
  'reconnect',
  'settings',
];

const terminalPetalDefaultLayout = <TerminalPetalMenuPlacement>[
  TerminalPetalMenuPlacement(id: 'history', depth: 0),
  TerminalPetalMenuPlacement(id: 'search', depth: 0),
  TerminalPetalMenuPlacement(id: 'selection', depth: 0),
  TerminalPetalMenuPlacement(id: 'paste', depth: 0),
  TerminalPetalMenuPlacement(id: 'quick-keys', depth: 0),
  TerminalPetalMenuPlacement(id: 'keyboard', depth: 0),
  TerminalPetalMenuPlacement(id: 'resources', depth: 0),
  TerminalPetalMenuPlacement(id: 'more', depth: 0),
  TerminalPetalMenuPlacement(id: 'command-bar', depth: 1),
  TerminalPetalMenuPlacement(id: 'copy-screen', depth: 1),
  TerminalPetalMenuPlacement(id: 'files', depth: 1),
  TerminalPetalMenuPlacement(id: 'input-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'enter', depth: 2),
  TerminalPetalMenuPlacement(id: 'escape', depth: 2),
  TerminalPetalMenuPlacement(id: 'tab', depth: 2),
  TerminalPetalMenuPlacement(id: 'backspace', depth: 2),
  TerminalPetalMenuPlacement(id: 'delete', depth: 2),
  TerminalPetalMenuPlacement(id: 'interrupt', depth: 2),
  TerminalPetalMenuPlacement(id: 'eof', depth: 2),
  TerminalPetalMenuPlacement(id: 'suspend', depth: 2),
  TerminalPetalMenuPlacement(id: 'clear', depth: 2),
  TerminalPetalMenuPlacement(id: 'navigation-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'arrow-left', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-down', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-up', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-right', depth: 2),
  TerminalPetalMenuPlacement(id: 'home', depth: 2),
  TerminalPetalMenuPlacement(id: 'end', depth: 2),
  TerminalPetalMenuPlacement(id: 'page-up', depth: 2),
  TerminalPetalMenuPlacement(id: 'page-down', depth: 2),
  TerminalPetalMenuPlacement(id: 'session-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'sync-input', depth: 2),
  TerminalPetalMenuPlacement(id: 'resize', depth: 2),
  TerminalPetalMenuPlacement(id: 'reconnect', depth: 2),
  TerminalPetalMenuPlacement(id: 'split', depth: 1),
  TerminalPetalMenuPlacement(id: 'split-rows', depth: 2),
  TerminalPetalMenuPlacement(id: 'split-columns', depth: 2),
  TerminalPetalMenuPlacement(id: 'settings', depth: 1),
];

const _terminalPetalVersion4DefaultLayout = <TerminalPetalMenuPlacement>[
  TerminalPetalMenuPlacement(id: 'history', depth: 0),
  TerminalPetalMenuPlacement(id: 'search', depth: 0),
  TerminalPetalMenuPlacement(id: 'selection', depth: 0),
  TerminalPetalMenuPlacement(id: 'paste', depth: 0),
  TerminalPetalMenuPlacement(id: 'quick-keys', depth: 0),
  TerminalPetalMenuPlacement(id: 'keyboard', depth: 0),
  TerminalPetalMenuPlacement(id: 'resources', depth: 0),
  TerminalPetalMenuPlacement(id: 'more', depth: 0),
  TerminalPetalMenuPlacement(id: 'command-bar', depth: 1),
  TerminalPetalMenuPlacement(id: 'copy-screen', depth: 1),
  TerminalPetalMenuPlacement(id: 'files', depth: 1),
  TerminalPetalMenuPlacement(id: 'input-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'enter', depth: 2),
  TerminalPetalMenuPlacement(id: 'escape', depth: 2),
  TerminalPetalMenuPlacement(id: 'tab', depth: 2),
  TerminalPetalMenuPlacement(id: 'backspace', depth: 2),
  TerminalPetalMenuPlacement(id: 'delete', depth: 2),
  TerminalPetalMenuPlacement(id: 'interrupt', depth: 2),
  TerminalPetalMenuPlacement(id: 'eof', depth: 2),
  TerminalPetalMenuPlacement(id: 'suspend', depth: 2),
  TerminalPetalMenuPlacement(id: 'clear', depth: 2),
  TerminalPetalMenuPlacement(id: 'navigation-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'arrow-left', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-down', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-up', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-right', depth: 2),
  TerminalPetalMenuPlacement(id: 'home', depth: 2),
  TerminalPetalMenuPlacement(id: 'end', depth: 2),
  TerminalPetalMenuPlacement(id: 'page-up', depth: 2),
  TerminalPetalMenuPlacement(id: 'page-down', depth: 2),
  TerminalPetalMenuPlacement(id: 'session-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'split', depth: 2),
  TerminalPetalMenuPlacement(id: 'sync-input', depth: 2),
  TerminalPetalMenuPlacement(id: 'resize', depth: 2),
  TerminalPetalMenuPlacement(id: 'reconnect', depth: 2),
  TerminalPetalMenuPlacement(id: 'settings', depth: 1),
];

const _terminalPetalVersion3DefaultLayout = <TerminalPetalMenuPlacement>[
  TerminalPetalMenuPlacement(id: 'history', depth: 0),
  TerminalPetalMenuPlacement(id: 'search', depth: 0),
  TerminalPetalMenuPlacement(id: 'selection', depth: 0),
  TerminalPetalMenuPlacement(id: 'paste', depth: 0),
  TerminalPetalMenuPlacement(id: 'quick-keys', depth: 0),
  TerminalPetalMenuPlacement(id: 'keyboard', depth: 0),
  TerminalPetalMenuPlacement(id: 'enter', depth: 0),
  TerminalPetalMenuPlacement(id: 'escape', depth: 0),
  TerminalPetalMenuPlacement(id: 'resources', depth: 0),
  TerminalPetalMenuPlacement(id: 'more', depth: 0),
  TerminalPetalMenuPlacement(id: 'command-bar', depth: 1),
  TerminalPetalMenuPlacement(id: 'copy-screen', depth: 1),
  TerminalPetalMenuPlacement(id: 'files', depth: 1),
  TerminalPetalMenuPlacement(id: 'input-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'tab', depth: 2),
  TerminalPetalMenuPlacement(id: 'backspace', depth: 2),
  TerminalPetalMenuPlacement(id: 'delete', depth: 2),
  TerminalPetalMenuPlacement(id: 'interrupt', depth: 2),
  TerminalPetalMenuPlacement(id: 'eof', depth: 2),
  TerminalPetalMenuPlacement(id: 'suspend', depth: 2),
  TerminalPetalMenuPlacement(id: 'clear', depth: 2),
  TerminalPetalMenuPlacement(id: 'navigation-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'arrow-left', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-down', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-up', depth: 2),
  TerminalPetalMenuPlacement(id: 'arrow-right', depth: 2),
  TerminalPetalMenuPlacement(id: 'home', depth: 2),
  TerminalPetalMenuPlacement(id: 'end', depth: 2),
  TerminalPetalMenuPlacement(id: 'page-up', depth: 2),
  TerminalPetalMenuPlacement(id: 'page-down', depth: 2),
  TerminalPetalMenuPlacement(id: 'session-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'split', depth: 2),
  TerminalPetalMenuPlacement(id: 'sync-input', depth: 2),
  TerminalPetalMenuPlacement(id: 'resize', depth: 2),
  TerminalPetalMenuPlacement(id: 'reconnect', depth: 2),
  TerminalPetalMenuPlacement(id: 'settings', depth: 1),
];

const _terminalPetalVersion2DefaultLayout = <TerminalPetalMenuPlacement>[
  TerminalPetalMenuPlacement(id: 'history', depth: 0),
  TerminalPetalMenuPlacement(id: 'search', depth: 0),
  TerminalPetalMenuPlacement(id: 'more', depth: 0),
  TerminalPetalMenuPlacement(id: 'command-bar', depth: 1),
  TerminalPetalMenuPlacement(id: 'resize', depth: 1),
  TerminalPetalMenuPlacement(id: 'settings', depth: 1),
  TerminalPetalMenuPlacement(id: 'input-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'quick-keys', depth: 2),
  TerminalPetalMenuPlacement(id: 'keyboard', depth: 2),
  TerminalPetalMenuPlacement(id: 'interrupt', depth: 2),
  TerminalPetalMenuPlacement(id: 'clear', depth: 2),
  TerminalPetalMenuPlacement(id: 'session-tools', depth: 1),
  TerminalPetalMenuPlacement(id: 'split', depth: 2),
  TerminalPetalMenuPlacement(id: 'sync-input', depth: 2),
  TerminalPetalMenuPlacement(id: 'selection', depth: 0),
  TerminalPetalMenuPlacement(id: 'paste', depth: 0),
];

const _legacyRootActionIds = <String>[
  'history',
  'search',
  'more',
  'selection',
  'paste',
];

const _legacyMoreActionIds = <String>['command-bar', 'resize', 'settings'];

final class TerminalPetalMenuPlacement {
  const TerminalPetalMenuPlacement({required this.id, required this.depth});

  final String id;
  final int depth;

  TerminalPetalMenuPlacement atDepth(int value) =>
      TerminalPetalMenuPlacement(id: id, depth: value);

  Map<String, Object> toJson() => <String, Object>{'id': id, 'depth': depth};

  static TerminalPetalMenuPlacement? fromJson(Object? value) {
    if (value is! Map) return null;
    final id = value['id'];
    final depth = value['depth'];
    if (id is! String || depth is! int) return null;
    return TerminalPetalMenuPlacement(id: id, depth: depth);
  }

  @override
  bool operator ==(Object other) =>
      other is TerminalPetalMenuPlacement &&
      other.id == id &&
      other.depth == depth;

  @override
  int get hashCode => Object.hash(id, depth);
}

final class TerminalPetalMenuPreferences {
  const TerminalPetalMenuPreferences({
    required this.enabled,
    required this.hapticsEnabled,
    required this.layout,
    required this.hiddenActionIds,
  });

  static const defaults = TerminalPetalMenuPreferences(
    enabled: true,
    hapticsEnabled: true,
    layout: terminalPetalDefaultLayout,
    hiddenActionIds: <String>{},
  );

  final bool enabled;
  final bool hapticsEnabled;
  final List<TerminalPetalMenuPlacement> layout;
  final Set<String> hiddenActionIds;

  List<TerminalPetalMenuPlacement> get visibleLayout {
    final result = <TerminalPetalMenuPlacement>[];
    final visibleAncestorDepths = List<int?>.filled(
      terminalPetalMaximumDepth + 1,
      null,
    );
    for (final placement in layout) {
      for (
        var depth = placement.depth;
        depth < visibleAncestorDepths.length;
        depth += 1
      ) {
        visibleAncestorDepths[depth] = null;
      }
      var parentDepth = -1;
      for (var depth = placement.depth - 1; depth >= 0; depth -= 1) {
        final visibleDepth = visibleAncestorDepths[depth];
        if (visibleDepth != null) {
          parentDepth = visibleDepth;
          break;
        }
      }
      if (hiddenActionIds.contains(placement.id)) continue;
      final visibleDepth = (parentDepth + 1).clamp(
        0,
        terminalPetalMaximumDepth,
      );
      result.add(placement.atDepth(visibleDepth));
      visibleAncestorDepths[placement.depth] = visibleDepth;
    }
    return List.unmodifiable(result);
  }

  List<String> get visibleRootActionIds => List.unmodifiable(
    visibleLayout
        .where((placement) => placement.depth == 0)
        .map((placement) => placement.id),
  );

  int get visibleActionCount => visibleLayout.length;

  bool isActionVisible(String id) => !hiddenActionIds.contains(id);

  TerminalPetalMenuPreferences copyWith({
    bool? enabled,
    bool? hapticsEnabled,
    List<TerminalPetalMenuPlacement>? layout,
    Set<String>? hiddenActionIds,
  }) => TerminalPetalMenuPreferences(
    enabled: enabled ?? this.enabled,
    hapticsEnabled: hapticsEnabled ?? this.hapticsEnabled,
    layout: layout ?? this.layout,
    hiddenActionIds: hiddenActionIds ?? this.hiddenActionIds,
  ).normalized();

  TerminalPetalMenuPreferences normalized() => TerminalPetalMenuPreferences(
    enabled: enabled,
    hapticsEnabled: hapticsEnabled,
    layout: _normalizedLayout(layout),
    hiddenActionIds: Set.unmodifiable(
      hiddenActionIds.where(terminalPetalActionIds.contains),
    ),
  );

  Map<String, Object> toJson() => <String, Object>{
    'version': 5,
    'enabled': enabled,
    'hapticsEnabled': hapticsEnabled,
    'layout': layout.map((placement) => placement.toJson()).toList(),
    'hiddenActionIds': hiddenActionIds.toList(growable: false),
  };

  factory TerminalPetalMenuPreferences.fromJson(Map<String, dynamic> json) {
    final parsedLayout = _placementList(json['layout']);
    final version = json['version'] as int? ?? 0;
    final layout = parsedLayout == null || parsedLayout.isEmpty
        ? _migrateLegacyLayout(json)
        : version < 5 &&
              (_sameItems(parsedLayout, _terminalPetalVersion2DefaultLayout) ||
                  _sameItems(
                    parsedLayout,
                    _terminalPetalVersion3DefaultLayout,
                  ) ||
                  _sameItems(parsedLayout, _terminalPetalVersion4DefaultLayout))
        ? terminalPetalDefaultLayout
        : parsedLayout;
    return TerminalPetalMenuPreferences(
      enabled: json['enabled'] as bool? ?? defaults.enabled,
      hapticsEnabled:
          json['hapticsEnabled'] as bool? ?? defaults.hapticsEnabled,
      layout: layout,
      hiddenActionIds:
          (_stringList(json['hiddenActionIds']) ?? const <String>[]).toSet(),
    ).normalized();
  }

  @override
  bool operator ==(Object other) =>
      other is TerminalPetalMenuPreferences &&
      other.enabled == enabled &&
      other.hapticsEnabled == hapticsEnabled &&
      _sameItems(other.layout, layout) &&
      _sameSet(other.hiddenActionIds, hiddenActionIds);

  @override
  int get hashCode => Object.hash(
    enabled,
    hapticsEnabled,
    Object.hashAll(layout),
    Object.hashAllUnordered(hiddenActionIds),
  );
}

List<TerminalPetalMenuPlacement> moveTerminalPetalSubtree(
  List<TerminalPetalMenuPlacement> layout, {
  required String actionId,
  required int offset,
}) {
  final index = layout.indexWhere((placement) => placement.id == actionId);
  if (index < 0 || offset == 0) return List.unmodifiable(layout);
  final depth = layout[index].depth;
  final end = _subtreeEnd(layout, index);
  if (offset < 0) {
    var previous = index - 1;
    while (previous >= 0 && layout[previous].depth > depth) {
      previous -= 1;
    }
    if (previous < 0 || layout[previous].depth != depth) {
      return List.unmodifiable(layout);
    }
    return _moveBlock(layout, index, end, previous);
  }
  if (end >= layout.length || layout[end].depth != depth) {
    return List.unmodifiable(layout);
  }
  final nextEnd = _subtreeEnd(layout, end);
  return _moveBlock(layout, index, end, nextEnd);
}

List<TerminalPetalMenuPlacement> reorderTerminalPetalSubtree(
  List<TerminalPetalMenuPlacement> layout, {
  required int oldIndex,
  required int newIndex,
}) {
  if (oldIndex < 0 || oldIndex >= layout.length || layout.isEmpty) {
    return List.unmodifiable(layout);
  }
  final end = _subtreeEnd(layout, oldIndex);
  if (newIndex >= oldIndex && newIndex < end) return List.unmodifiable(layout);
  final target = newIndex.clamp(0, layout.length - 1);
  final blockLength = end - oldIndex;
  final insertion = target > oldIndex ? target + blockLength : target;
  return _moveBlock(layout, oldIndex, end, insertion);
}

List<TerminalPetalMenuPlacement> indentTerminalPetalSubtree(
  List<TerminalPetalMenuPlacement> layout, {
  required String actionId,
}) {
  final index = layout.indexWhere((placement) => placement.id == actionId);
  if (index <= 0) return List.unmodifiable(layout);
  final depth = layout[index].depth;
  if (depth >= terminalPetalMaximumDepth) return List.unmodifiable(layout);
  var previousSibling = index - 1;
  while (previousSibling >= 0 && layout[previousSibling].depth > depth) {
    previousSibling -= 1;
  }
  if (previousSibling < 0 || layout[previousSibling].depth != depth) {
    return List.unmodifiable(layout);
  }
  return _shiftSubtreeDepth(layout, index, 1);
}

List<TerminalPetalMenuPlacement> outdentTerminalPetalSubtree(
  List<TerminalPetalMenuPlacement> layout, {
  required String actionId,
}) {
  final index = layout.indexWhere((placement) => placement.id == actionId);
  if (index < 0 || layout[index].depth == 0) return List.unmodifiable(layout);
  final depth = layout[index].depth;
  var parentIndex = index - 1;
  while (parentIndex >= 0 && layout[parentIndex].depth >= depth) {
    parentIndex -= 1;
  }
  if (parentIndex < 0 || layout[parentIndex].depth != depth - 1) {
    return List.unmodifiable(layout);
  }
  final end = _subtreeEnd(layout, index);
  final parentEnd = _subtreeEnd(layout, parentIndex);
  final promoted = [
    for (var cursor = index; cursor < end; cursor += 1)
      layout[cursor].atDepth(layout[cursor].depth - 1),
  ];
  final reordered = [...layout]..removeRange(index, end);
  reordered.insertAll(parentEnd - promoted.length, promoted);
  return List.unmodifiable(reordered);
}

List<TerminalPetalMenuPlacement> reparentTerminalPetalSubtree(
  List<TerminalPetalMenuPlacement> layout, {
  required String actionId,
  required String parentActionId,
}) {
  final index = layout.indexWhere((placement) => placement.id == actionId);
  final parentIndex = layout.indexWhere(
    (placement) => placement.id == parentActionId,
  );
  if (index < 0 || parentIndex < 0 || index == parentIndex) {
    return List.unmodifiable(layout);
  }
  final end = _subtreeEnd(layout, index);
  if (parentIndex > index && parentIndex < end) {
    return List.unmodifiable(layout);
  }
  final sourceDepth = layout[index].depth;
  final targetDepth = layout[parentIndex].depth + 1;
  final relativeHeight = layout
      .sublist(index, end)
      .map((placement) => placement.depth - sourceDepth)
      .fold(0, (maximum, depth) => depth > maximum ? depth : maximum);
  if (targetDepth + relativeHeight > terminalPetalMaximumDepth) {
    return List.unmodifiable(layout);
  }

  final moved = [
    for (var cursor = index; cursor < end; cursor += 1)
      layout[cursor].atDepth(targetDepth + layout[cursor].depth - sourceDepth),
  ];
  final reordered = [...layout]..removeRange(index, end);
  final updatedParentIndex = reordered.indexWhere(
    (placement) => placement.id == parentActionId,
  );
  final insertion = _subtreeEnd(reordered, updatedParentIndex);
  reordered.insertAll(insertion, moved);
  return List.unmodifiable(reordered);
}

List<TerminalPetalMenuPlacement> _shiftSubtreeDepth(
  List<TerminalPetalMenuPlacement> layout,
  int index,
  int delta,
) {
  final end = _subtreeEnd(layout, index);
  final shifted = [...layout];
  for (var cursor = index; cursor < end; cursor += 1) {
    shifted[cursor] = shifted[cursor].atDepth(shifted[cursor].depth + delta);
  }
  return List.unmodifiable(shifted);
}

List<TerminalPetalMenuPlacement> _moveBlock(
  List<TerminalPetalMenuPlacement> layout,
  int start,
  int end,
  int insertion,
) {
  final reordered = [...layout];
  final block = reordered.sublist(start, end);
  reordered.removeRange(start, end);
  var target = insertion;
  if (target > start) target -= block.length;
  target = target.clamp(0, reordered.length);
  reordered.insertAll(target, block);
  return List.unmodifiable(reordered);
}

int _subtreeEnd(List<TerminalPetalMenuPlacement> layout, int index) {
  final depth = layout[index].depth;
  var end = index + 1;
  while (end < layout.length && layout[end].depth > depth) {
    end += 1;
  }
  return end;
}

List<TerminalPetalMenuPlacement> _normalizedLayout(
  List<TerminalPetalMenuPlacement> value,
) {
  final valid = terminalPetalActionIds.toSet();
  final seen = <String>{};
  final normalized = <TerminalPetalMenuPlacement>[];
  for (final placement in value) {
    if (!valid.contains(placement.id) || !seen.add(placement.id)) continue;
    final maximum = normalized.isEmpty
        ? 0
        : (normalized.last.depth + 1).clamp(0, terminalPetalMaximumDepth);
    normalized.add(placement.atDepth(placement.depth.clamp(0, maximum)));
  }
  if (normalized.isEmpty) return terminalPetalDefaultLayout;

  final defaultParents = <String, String?>{};
  final parentAtDepth = List<String?>.filled(
    terminalPetalMaximumDepth + 1,
    null,
  );
  for (final placement in terminalPetalDefaultLayout) {
    defaultParents[placement.id] = placement.depth == 0
        ? null
        : parentAtDepth[placement.depth - 1];
    parentAtDepth[placement.depth] = placement.id;
  }
  for (final placement in terminalPetalDefaultLayout) {
    if (seen.contains(placement.id)) continue;
    final parentId = defaultParents[placement.id];
    final parentIndex = parentId == null
        ? -1
        : normalized.indexWhere((item) => item.id == parentId);
    if (parentIndex < 0) {
      normalized.add(placement.atDepth(0));
    } else {
      final insertion = _subtreeEnd(normalized, parentIndex);
      normalized.insert(
        insertion,
        placement.atDepth(
          (normalized[parentIndex].depth + 1).clamp(
            0,
            terminalPetalMaximumDepth,
          ),
        ),
      );
    }
    seen.add(placement.id);
  }
  return List.unmodifiable(normalized);
}

List<TerminalPetalMenuPlacement> _migrateLegacyLayout(
  Map<String, dynamic> json,
) {
  final roots = _normalizedLegacyOrder(
    _stringList(json['rootOrder']) ?? _legacyRootActionIds,
    _legacyRootActionIds,
  );
  final more = _normalizedLegacyOrder(
    _stringList(json['moreOrder']) ?? _legacyMoreActionIds,
    _legacyMoreActionIds,
  );
  final layout = <TerminalPetalMenuPlacement>[];
  for (final id in roots) {
    layout.add(TerminalPetalMenuPlacement(id: id, depth: 0));
    if (id != 'more') continue;
    layout.addAll(
      more.map((id) => TerminalPetalMenuPlacement(id: id, depth: 1)),
    );
    layout.addAll(const [
      TerminalPetalMenuPlacement(id: 'input-tools', depth: 1),
      TerminalPetalMenuPlacement(id: 'quick-keys', depth: 2),
      TerminalPetalMenuPlacement(id: 'keyboard', depth: 2),
      TerminalPetalMenuPlacement(id: 'interrupt', depth: 2),
      TerminalPetalMenuPlacement(id: 'clear', depth: 2),
      TerminalPetalMenuPlacement(id: 'session-tools', depth: 1),
      TerminalPetalMenuPlacement(id: 'split', depth: 2),
      TerminalPetalMenuPlacement(id: 'sync-input', depth: 2),
    ]);
  }
  return layout;
}

List<String> _normalizedLegacyOrder(List<String> value, List<String> defaults) {
  final valid = defaults.toSet();
  final seen = <String>{};
  return [
    for (final id in value)
      if (valid.contains(id) && seen.add(id)) id,
    for (final id in defaults)
      if (seen.add(id)) id,
  ];
}

List<TerminalPetalMenuPlacement>? _placementList(Object? value) {
  if (value is! List) return null;
  final placements = <TerminalPetalMenuPlacement>[];
  for (final item in value) {
    final placement = TerminalPetalMenuPlacement.fromJson(item);
    if (placement != null) placements.add(placement);
  }
  return placements;
}

List<String>? _stringList(Object? value) {
  if (value is! List) return null;
  return [
    for (final item in value)
      if (item is String) item,
  ];
}

bool _sameItems<T>(List<T> left, List<T> right) {
  if (identical(left, right)) return true;
  if (left.length != right.length) return false;
  for (var index = 0; index < left.length; index += 1) {
    if (left[index] != right[index]) return false;
  }
  return true;
}

bool _sameSet(Set<String> left, Set<String> right) =>
    left.length == right.length && left.containsAll(right);
