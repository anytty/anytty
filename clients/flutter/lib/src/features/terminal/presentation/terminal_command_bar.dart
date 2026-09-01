import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';
import '../domain/terminal_modifiers.dart';
import '../domain/terminal_quick_action.dart';

typedef TerminalQuickActionSaver = Future<void> Function(
  List<TerminalQuickAction> actions,
);

typedef TerminalQuickActionColors = ({
  Color foreground,
  Color background,
  Color border,
});

TerminalQuickActionColors terminalQuickActionColors(
  TerminalQuickAction action, {
  bool active = false,
  bool enabled = true,
  AnyttyPalette? palette,
}) {
  if (palette != null) {
    if (!enabled) {
      return (
        foreground: palette.faint,
        background: palette.surface,
        border: palette.border,
      );
    }
    if (active) {
      return (
        foreground: palette.accent,
        background: Color.lerp(palette.surface, palette.accent, 0.16)!,
        border: Color.lerp(palette.border, palette.accent, 0.48)!,
      );
    }
    final tone = switch (action.kind) {
      TerminalQuickActionKind.key => palette.text,
      TerminalQuickActionKind.chord => palette.accent,
      TerminalQuickActionKind.text => palette.warning,
    };
    return (
      foreground: tone,
      background: Color.lerp(palette.surface, tone, 0.06)!,
      border: Color.lerp(palette.border, tone, 0.24)!,
    );
  }
  if (!enabled) {
    return (
      foreground: const Color(0xff626a67),
      background: const Color(0xff1b1f1e),
      border: const Color(0xff292e2c),
    );
  }
  if (active) {
    return (
      foreground: const Color(0xff8fe3d5),
      background: const Color(0xff263633),
      border: const Color(0xff41615b),
    );
  }
  return switch (action.kind) {
    TerminalQuickActionKind.key => (
      foreground: const Color(0xffd3d9d7),
      background: const Color(0xff202523),
      border: const Color(0xff303633),
    ),
    TerminalQuickActionKind.chord => (
      foreground: const Color(0xff9fe6d8),
      background: const Color(0xff1c2725),
      border: const Color(0xff35514b),
    ),
    TerminalQuickActionKind.text => (
      foreground: const Color(0xffe8d39a),
      background: const Color(0xff27231c),
      border: const Color(0xff51472e),
    ),
  };
}

final class TerminalCommandBar extends StatefulWidget {
  const TerminalCommandBar({
    super.key,
    required this.actions,
    required this.inputEnabled,
    required this.modifiers,
    required this.keyboardControl,
    required this.onAction,
    required this.onConfigure,
    required this.functionKeysActive,
    required this.onFunctionKeys,
  });

  final List<TerminalQuickAction> actions;
  final bool inputEnabled;
  final TerminalModifierState modifiers;
  final Widget keyboardControl;
  final ValueChanged<TerminalQuickAction> onAction;
  final VoidCallback onConfigure;
  final bool functionKeysActive;
  final VoidCallback onFunctionKeys;

  @override
  State<TerminalCommandBar> createState() => _TerminalCommandBarState();
}

final class _TerminalCommandBarState extends State<TerminalCommandBar> {
  static const _itemExtent = 52.0;

  late ScrollController _controller;

  @override
  void initState() {
    super.initState();
    _resetController();
  }

  @override
  void didUpdateWidget(TerminalCommandBar oldWidget) {
    super.didUpdateWidget(oldWidget);
    final oldIds = oldWidget.actions.map((action) => action.id).join('\n');
    final nextIds = widget.actions.map((action) => action.id).join('\n');
    if (oldIds != nextIds) {
      _controller.dispose();
      _resetController();
    }
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _resetController() {
    _controller = ScrollController();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return RepaintBoundary(
      child: SafeArea(
        top: false,
        minimum: const EdgeInsets.only(bottom: 1),
        child: Container(
          height: 38,
          padding: const EdgeInsets.symmetric(horizontal: 3, vertical: 2),
          decoration: BoxDecoration(
            color: palette.surface,
            border: Border(top: BorderSide(color: palette.border)),
          ),
          child: Row(
            children: [
              _BarAnchorButton(
                tooltip: widget.functionKeysActive
                    ? 'Close function keys'
                    : 'Show function keys',
                label: 'Fn',
                active: widget.functionKeysActive,
                onPressed: widget.onFunctionKeys,
              ),
              const SizedBox(width: 2),
              Expanded(child: _buildReel(context)),
              const SizedBox(width: 2),
              SizedBox.square(dimension: 34, child: widget.keyboardControl),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildReel(BuildContext context) {
    if (widget.actions.isEmpty) {
      return Semantics(
        button: true,
        label: 'Add command bar action',
        child: InkWell(
          key: const ValueKey('command-bar-empty'),
          onTap: widget.onConfigure,
          borderRadius: BorderRadius.circular(6),
          child: Center(
            child: Text(
              'Add command',
              style: TextStyle(
                color: AnyttyPalette.of(context).muted,
                fontSize: 12,
              ),
            ),
          ),
        ),
      );
    }
    return Semantics(
      container: true,
      enabled: widget.inputEnabled,
      label: 'Command bar',
      child: ListView.builder(
        key: const ValueKey('command-reel'),
        controller: _controller,
        scrollDirection: Axis.horizontal,
        physics: const ClampingScrollPhysics(),
        itemExtent: _itemExtent,
        itemCount: widget.actions.length,
        itemBuilder: (context, itemIndex) {
          final action = widget.actions[itemIndex];
          return Padding(
            padding: const EdgeInsets.symmetric(horizontal: 1),
            child: _CommandReelItem(
              key: ValueKey('${action.id}-$itemIndex'),
              action: action,
              active: _modifierActive(action),
              enabled: widget.inputEnabled,
              onPressed: () {
                HapticFeedback.selectionClick();
                widget.onAction(action);
              },
              onConfigure: widget.onConfigure,
            ),
          );
        },
      ),
    );
  }

  bool _modifierActive(TerminalQuickAction action) {
    final modifier = action.kind == TerminalQuickActionKind.key
        ? action.key?.modifier
        : null;
    return modifier != null &&
        widget.modifiers.stateOf(modifier) != TerminalModifierLatch.off;
  }
}

final class _CommandReelItem extends StatelessWidget {
  const _CommandReelItem({
    super.key,
    required this.action,
    required this.active,
    required this.enabled,
    required this.onPressed,
    required this.onConfigure,
  });

  final TerminalQuickAction action;
  final bool active;
  final bool enabled;
  final VoidCallback onPressed;
  final VoidCallback onConfigure;

  @override
  Widget build(BuildContext context) {
    final colors = terminalQuickActionColors(
      action,
      active: active,
      enabled: enabled,
      palette: AnyttyPalette.of(context),
    );
    return Semantics(
      button: true,
      enabled: enabled,
      toggled: active,
      label: 'Run ${action.displayLabel}',
      hint: 'Long press to edit command bar',
      child: AnimatedContainer(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        curve: Curves.easeOutCubic,
        margin: const EdgeInsets.symmetric(vertical: 1),
        decoration: BoxDecoration(
          color: colors.background,
          borderRadius: BorderRadius.circular(4),
          border: Border.all(color: colors.border),
        ),
        child: InkWell(
          onTap: enabled ? onPressed : null,
          onLongPress: onConfigure,
          borderRadius: BorderRadius.circular(4),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Center(
              child: Text(
                action.displayLabel,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                textAlign: TextAlign.center,
                style: TextStyle(
                  color: colors.foreground,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 9,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

final class _BarAnchorButton extends StatelessWidget {
  const _BarAnchorButton({
    required this.tooltip,
    required this.label,
    required this.active,
    required this.onPressed,
  });

  final String tooltip;
  final String label;
  final bool active;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox.square(
      dimension: 34,
      child: Tooltip(
        message: tooltip,
        child: Semantics(
          button: true,
          label: tooltip,
          child: InkWell(
            onTap: onPressed,
            borderRadius: BorderRadius.circular(4),
            child: DecoratedBox(
              decoration: BoxDecoration(
                color: active
                    ? Color.lerp(palette.surface, palette.accent, 0.16)
                    : palette.surfaceRaised,
                borderRadius: const BorderRadius.all(Radius.circular(4)),
              ),
              child: Center(
                child: Text(
                  label,
                  style: TextStyle(
                    color: active ? palette.accent : palette.text,
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

Future<bool?> showAnyttyCommandBarEditor({
  required BuildContext context,
  required List<TerminalQuickAction> actions,
  required TerminalQuickActionSaver onSave,
}) {
  final palette = AnyttyPalette.of(context);
  return showGeneralDialog<bool>(
    context: context,
    barrierDismissible: false,
    barrierLabel: 'Command bar editor',
    barrierColor: palette.overlay,
    transitionDuration: Duration.zero,
    pageBuilder: (context, _, _) =>
        _CommandEditorRoute(actions: actions, onSave: onSave),
    transitionBuilder: (context, animation, _, child) => child,
  );
}

final class _CommandEditorRoute extends StatelessWidget {
  const _CommandEditorRoute({required this.actions, required this.onSave});

  final List<TerminalQuickAction> actions;
  final TerminalQuickActionSaver onSave;

  @override
  Widget build(BuildContext context) {
    final media = MediaQuery.of(context);
    final available = math.max(
      280.0,
      media.size.height - media.viewInsets.bottom - media.padding.top - 12,
    );
    return SafeArea(
      top: false,
      child: AnimatedPadding(
        duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
        curve: Curves.easeOutCubic,
        padding: EdgeInsets.only(bottom: media.viewInsets.bottom),
        child: Align(
          alignment: Alignment.bottomCenter,
          child: SizedBox(
            width: double.infinity,
            height: math.min(available, 680),
            child: TerminalCommandEditorTray(actions: actions, onSave: onSave),
          ),
        ),
      ),
    );
  }
}

enum _EditorPage { list, form, keyPicker }

final class TerminalCommandEditorTray extends StatefulWidget {
  const TerminalCommandEditorTray({
    super.key,
    required this.actions,
    required this.onSave,
  });

  final List<TerminalQuickAction> actions;
  final TerminalQuickActionSaver onSave;

  @override
  State<TerminalCommandEditorTray> createState() =>
      _TerminalCommandEditorTrayState();
}

final class _TerminalCommandEditorTrayState
    extends State<TerminalCommandEditorTray> {
  late List<TerminalQuickAction> _actions;
  final _labelController = TextEditingController();
  final _textController = TextEditingController();
  final _searchController = TextEditingController();
  _EditorPage _page = _EditorPage.list;
  TerminalQuickAction? _draft;
  int? _editingIndex;
  bool _dirty = false;
  bool _saving = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _actions = [...widget.actions];
    _searchController.addListener(_refresh);
  }

  @override
  void dispose() {
    _labelController.dispose();
    _textController.dispose();
    _searchController
      ..removeListener(_refresh)
      ..dispose();
    super.dispose();
  }

  void _refresh() => setState(() {});

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return PopScope<void>(
      canPop: !_dirty,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop) _requestClose();
      },
      child: Material(
        color: palette.surface,
        clipBehavior: Clip.antiAlias,
        shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(8)),
        ),
        child: Column(
          children: [
            _buildHeader(context),
            Divider(height: 1, color: palette.border),
            Expanded(
              child: AnimatedSwitcher(
                duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
                child: switch (_page) {
                  _EditorPage.list => _buildList(context),
                  _EditorPage.form => _buildForm(context),
                  _EditorPage.keyPicker => _buildKeyPicker(context),
                },
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildHeader(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final inList = _page == _EditorPage.list;
    final title = switch (_page) {
      _EditorPage.list => 'Command bar',
      _EditorPage.form =>
        _editingIndex == null ? 'New command' : 'Edit command',
      _EditorPage.keyPicker => 'Choose key',
    };
    return SizedBox(
      height: 60,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 4),
        child: Row(
          children: [
            _TrayIconButton(
              tooltip: inList ? 'Close' : 'Back',
              icon: inList ? LucideIcons.x : LucideIcons.chevronLeft,
              onPressed: inList ? _requestClose : _goBack,
            ),
            const SizedBox(width: 4),
            Expanded(
              child: Text(
                title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  color: palette.text,
                  fontSize: 16,
                  fontWeight: FontWeight.w700,
                ),
              ),
            ),
            if (inList) ...[
              _TrayIconButton(
                tooltip: 'Add command',
                icon: LucideIcons.plus,
                onPressed: _startAdd,
              ),
              const SizedBox(width: 2),
              _TrayTextButton(
                label: _saving ? 'Saving' : 'Done',
                enabled: !_saving,
                emphasized: true,
                onPressed: _save,
              ),
            ] else if (_page == _EditorPage.form)
              _TrayTextButton(
                label: 'Apply',
                enabled: _draftCanApply,
                emphasized: true,
                onPressed: _applyDraft,
              ),
          ],
        ),
      ),
    );
  }

  Widget _buildList(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    if (_actions.isEmpty) {
      return KeyedSubtree(
        key: const ValueKey('command-editor-list-empty'),
        child: Center(
          child: _TrayTextButton(
            label: 'Add command',
            enabled: true,
            emphasized: true,
            onPressed: _startAdd,
          ),
        ),
      );
    }
    return KeyedSubtree(
      key: const ValueKey('command-editor-list'),
      child: Column(
        children: [
          if (_error case final message?)
            Container(
              width: double.infinity,
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
              color: palette.danger.withValues(alpha: 0.1),
              child: Text(
                message,
                style: TextStyle(color: palette.danger, fontSize: 12),
              ),
            ),
          Expanded(
            child: ReorderableListView.builder(
              padding: const EdgeInsets.only(bottom: 8),
              buildDefaultDragHandles: false,
              itemCount: _actions.length,
              onReorderItem: _reorder,
              itemBuilder: (context, index) {
                final action = _actions[index];
                return _CommandEditorRow(
                  key: ValueKey(action.id),
                  action: action,
                  index: index,
                  onEdit: () => _startEdit(index),
                  onDelete: () => _delete(index),
                );
              },
            ),
          ),
          Divider(height: 1, color: palette.border),
          SizedBox(
            height: 50,
            child: Center(
              child: _TrayTextButton(
                label: 'Restore defaults',
                enabled: !_saving,
                emphasized: false,
                onPressed: _restoreDefaults,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildForm(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final draft = _draft!;
    return KeyedSubtree(
      key: const ValueKey('command-editor-form'),
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 18, 16, 32),
        children: [
          Text(
            'TYPE',
            style: TextStyle(
              color: palette.muted,
              fontSize: 11,
              fontWeight: FontWeight.w700,
            ),
          ),
          const SizedBox(height: 8),
          _CommandKindPicker(value: draft.kind, onChanged: _changeKind),
          const SizedBox(height: 20),
          if (draft.kind != TerminalQuickActionKind.text) ...[
            Text(
              'KEY',
              style: TextStyle(
                color: palette.muted,
                fontSize: 11,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            _KeySelectionButton(
              label: draft.key?.label ?? 'Choose key',
              onPressed: () => setState(() {
                _searchController.clear();
                _page = _EditorPage.keyPicker;
              }),
            ),
          ],
          if (draft.kind == TerminalQuickActionKind.chord) ...[
            const SizedBox(height: 20),
            Text(
              'MODIFIERS',
              style: TextStyle(
                color: palette.muted,
                fontSize: 11,
                fontWeight: FontWeight.w700,
              ),
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                _ModifierChoice(
                  label: 'Ctrl',
                  selected: draft.modifiers & terminalModifierControlBit != 0,
                  onPressed: () =>
                      _toggleDraftModifier(terminalModifierControlBit),
                ),
                _ModifierChoice(
                  label: 'Alt',
                  selected: draft.modifiers & terminalModifierAltBit != 0,
                  onPressed: () => _toggleDraftModifier(terminalModifierAltBit),
                ),
                _ModifierChoice(
                  label: 'Shift',
                  selected: draft.modifiers & terminalModifierShiftBit != 0,
                  onPressed: () =>
                      _toggleDraftModifier(terminalModifierShiftBit),
                ),
                _ModifierChoice(
                  label: 'Super',
                  selected: draft.modifiers & terminalModifierSuperBit != 0,
                  onPressed: () =>
                      _toggleDraftModifier(terminalModifierSuperBit),
                ),
              ],
            ),
          ],
          if (draft.kind == TerminalQuickActionKind.text) ...[
            TextField(
              key: const ValueKey('command-text-field'),
              controller: _textController,
              autofocus: true,
              minLines: 2,
              maxLines: 5,
              decoration: const InputDecoration(
                labelText: 'Text',
                hintText: 'git status',
              ),
              onChanged: (value) => setState(() {
                _draft = _draft!.copyWith(text: value);
              }),
            ),
            const SizedBox(height: 12),
            _InlineToggle(
              label: 'Send Enter after text',
              value: draft.sendEnter,
              onChanged: (value) => setState(() {
                _draft = _draft!.copyWith(sendEnter: value);
              }),
            ),
          ],
          const SizedBox(height: 20),
          TextField(
            controller: _labelController,
            maxLines: 1,
            decoration: const InputDecoration(
              labelText: 'Label (optional)',
              hintText: 'Uses the key or text when empty',
            ),
            onChanged: (value) => setState(() {
              _draft = _draft!.copyWith(label: value);
            }),
          ),
        ],
      ),
    );
  }

  Widget _buildKeyPicker(BuildContext context) {
    final draft = _draft!;
    final query = _searchController.text.trim().toLowerCase();
    final values = terminalQuickKeys
        .where((key) {
          if (draft.kind == TerminalQuickActionKind.chord &&
              !key.supportsChord) {
            return false;
          }
          return query.isEmpty ||
              key.label.toLowerCase().contains(query) ||
              key.id.contains(query);
        })
        .toList(growable: false);
    return KeyedSubtree(
      key: const ValueKey('command-editor-key-picker'),
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
            child: TextField(
              key: const ValueKey('command-key-search'),
              controller: _searchController,
              autofocus: true,
              decoration: const InputDecoration(
                hintText: 'Search keys',
                prefixIcon: Icon(LucideIcons.search, size: 18),
              ),
            ),
          ),
          Expanded(
            child: ListView.builder(
              itemCount: values.length,
              itemBuilder: (context, index) {
                final key = values[index];
                final selected = key.id == draft.keyId;
                return _KeyPickerRow(
                  key: ValueKey('quick-key-${key.id}'),
                  value: key,
                  selected: selected,
                  onPressed: () => setState(() {
                    _draft = draft.copyWith(keyId: key.id);
                    _page = _EditorPage.form;
                  }),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  void _startAdd() {
    final action = TerminalQuickAction(
      id: 'custom-${DateTime.now().microsecondsSinceEpoch}',
      kind: TerminalQuickActionKind.key,
      keyId: 'escape',
    );
    _openDraft(action, null);
  }

  void _startEdit(int index) => _openDraft(_actions[index], index);

  void _openDraft(TerminalQuickAction action, int? index) {
    _labelController.text = action.label;
    _textController.text = action.text;
    setState(() {
      _editingIndex = index;
      _draft = action;
      _page = _EditorPage.form;
    });
  }

  void _changeKind(TerminalQuickActionKind kind) {
    final draft = _draft!;
    final next = switch (kind) {
      TerminalQuickActionKind.key => TerminalQuickAction(
        id: draft.id,
        kind: kind,
        keyId: terminalQuickKeyById(draft.keyId) != null
            ? draft.keyId
            : 'escape',
        label: draft.label,
      ),
      TerminalQuickActionKind.chord => TerminalQuickAction(
        id: draft.id,
        kind: kind,
        keyId: terminalQuickKeyById(draft.keyId)?.supportsChord == true
            ? draft.keyId
            : 'c',
        modifiers: draft.modifiers == 0
            ? terminalModifierControlBit
            : draft.modifiers,
        label: draft.label,
      ),
      TerminalQuickActionKind.text => TerminalQuickAction(
        id: draft.id,
        kind: kind,
        text: _textController.text,
        sendEnter: draft.sendEnter,
        label: draft.label,
      ),
    };
    setState(() => _draft = next);
  }

  void _toggleDraftModifier(int bit) {
    final draft = _draft!;
    setState(() => _draft = draft.copyWith(modifiers: draft.modifiers ^ bit));
  }

  void _applyDraft() {
    final current = _draft;
    if (current == null) return;
    final draft = current.copyWith(
      text: current.kind == TerminalQuickActionKind.text
          ? _textController.text
          : current.text,
      label: _labelController.text,
    );
    if (!draft.isValid) return;
    setState(() {
      if (_editingIndex case final index?) {
        _actions[index] = draft;
      } else {
        _actions.add(draft);
      }
      _dirty = true;
      _editingIndex = null;
      _draft = null;
      _page = _EditorPage.list;
    });
  }

  bool get _draftCanApply {
    final draft = _draft;
    if (draft == null) return false;
    if (draft.kind == TerminalQuickActionKind.text) {
      return _textController.text.isNotEmpty;
    }
    return draft.isValid;
  }

  void _delete(int index) {
    setState(() {
      _actions.removeAt(index);
      _dirty = true;
    });
  }

  void _reorder(int oldIndex, int newIndex) {
    setState(() {
      final action = _actions.removeAt(oldIndex);
      _actions.insert(newIndex, action);
      _dirty = true;
    });
  }

  void _restoreDefaults() {
    setState(() {
      _actions = [...defaultTerminalQuickActions];
      _dirty = true;
      _error = null;
    });
  }

  void _goBack() {
    setState(() {
      _page = _page == _EditorPage.keyPicker
          ? _EditorPage.form
          : _EditorPage.list;
      if (_page == _EditorPage.list) {
        _editingIndex = null;
        _draft = null;
      }
    });
  }

  Future<void> _save() async {
    if (_saving) return;
    setState(() {
      _saving = true;
      _error = null;
    });
    try {
      await widget.onSave(_actions);
      if (!mounted) return;
      _dirty = false;
      Navigator.pop(context, true);
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _saving = false;
        _error = 'Could not save the command bar. Try again.';
      });
    }
  }

  Future<void> _requestClose() async {
    if (!_dirty) {
      Navigator.pop(context, false);
      return;
    }
    final discard = await _confirmDiscard(context);
    if (discard && mounted) Navigator.pop(context, false);
  }
}

final class _CommandEditorRow extends StatelessWidget {
  const _CommandEditorRow({
    super.key,
    required this.action,
    required this.index,
    required this.onEdit,
    required this.onDelete,
  });

  final TerminalQuickAction action;
  final int index;
  final VoidCallback onEdit;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final kind = switch (action.kind) {
      TerminalQuickActionKind.key => 'Key',
      TerminalQuickActionKind.chord => 'Chord',
      TerminalQuickActionKind.text =>
        action.sendEnter ? 'Text + Enter' : 'Text',
    };
    return Material(
      color: palette.surface,
      child: InkWell(
        onTap: onEdit,
        child: Container(
          constraints: const BoxConstraints(minHeight: 58),
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: palette.border)),
          ),
          padding: const EdgeInsets.only(left: 16, right: 4),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      action.displayLabel,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                      style: TextStyle(
                        color: palette.text,
                        fontFamily: 'JetBrainsMonoNerd',
                        fontSize: 13,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      kind,
                      style: TextStyle(color: palette.muted, fontSize: 11),
                    ),
                  ],
                ),
              ),
              _TrayIconButton(
                tooltip: 'Delete ${action.displayLabel}',
                icon: LucideIcons.trash2,
                danger: true,
                onPressed: onDelete,
              ),
              ReorderableDragStartListener(
                index: index,
                child: Semantics(
                  label: 'Reorder ${action.displayLabel}',
                  child: SizedBox.square(
                    dimension: 48,
                    child: Icon(
                      LucideIcons.gripVertical,
                      color: palette.muted,
                      size: 19,
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _CommandKindPicker extends StatelessWidget {
  const _CommandKindPicker({required this.value, required this.onChanged});

  final TerminalQuickActionKind value;
  final ValueChanged<TerminalQuickActionKind> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      height: 48,
      padding: const EdgeInsets.all(3),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        children: [
          for (final option in TerminalQuickActionKind.values)
            Expanded(
              child: _KindOption(
                label: switch (option) {
                  TerminalQuickActionKind.key => 'Key',
                  TerminalQuickActionKind.chord => 'Chord',
                  TerminalQuickActionKind.text => 'Text',
                },
                selected: option == value,
                onPressed: () => onChanged(option),
              ),
            ),
        ],
      ),
    );
  }
}

final class _KindOption extends StatelessWidget {
  const _KindOption({
    required this.label,
    required this.selected,
    required this.onPressed,
  });

  final String label;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(4),
        child: AnimatedContainer(
          duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: selected ? palette.surface : Colors.transparent,
            borderRadius: BorderRadius.circular(4),
            border: Border.all(
              color: selected ? palette.borderStrong : Colors.transparent,
            ),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: selected ? palette.text : palette.muted,
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}

final class _KeySelectionButton extends StatelessWidget {
  const _KeySelectionButton({required this.label, required this.onPressed});

  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      label: 'Choose key, current value $label',
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(6),
        child: Container(
          constraints: const BoxConstraints(minHeight: 50),
          padding: const EdgeInsets.symmetric(horizontal: 14),
          decoration: BoxDecoration(
            color: palette.surface,
            border: Border.all(color: palette.borderStrong),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(
                    color: palette.text,
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ),
              Icon(LucideIcons.chevronRight, color: palette.muted, size: 18),
            ],
          ),
        ),
      ),
    );
  }
}

final class _ModifierChoice extends StatelessWidget {
  const _ModifierChoice({
    required this.label,
    required this.selected,
    required this.onPressed,
  });

  final String label;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      button: true,
      selected: selected,
      child: InkWell(
        onTap: onPressed,
        borderRadius: BorderRadius.circular(6),
        child: AnimatedContainer(
          duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
          constraints: const BoxConstraints(minWidth: 68, minHeight: 44),
          alignment: Alignment.center,
          padding: const EdgeInsets.symmetric(horizontal: 12),
          decoration: BoxDecoration(
            color: selected
                ? palette.accent.withValues(alpha: 0.12)
                : palette.surface,
            border: Border.all(
              color: selected ? palette.accent : palette.borderStrong,
            ),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Text(
            label,
            style: TextStyle(
              color: selected ? palette.accent : palette.text,
              fontFamily: 'JetBrainsMonoNerd',
              fontSize: 12,
              fontWeight: FontWeight.w600,
            ),
          ),
        ),
      ),
    );
  }
}

final class _InlineToggle extends StatelessWidget {
  const _InlineToggle({
    required this.label,
    required this.value,
    required this.onChanged,
  });

  final String label;
  final bool value;
  final ValueChanged<bool> onChanged;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      toggled: value,
      label: label,
      child: InkWell(
        onTap: () => onChanged(!value),
        borderRadius: BorderRadius.circular(6),
        child: Container(
          constraints: const BoxConstraints(minHeight: 50),
          padding: const EdgeInsets.symmetric(horizontal: 14),
          decoration: BoxDecoration(
            color: palette.surface,
            border: Border.all(color: palette.borderStrong),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  label,
                  style: TextStyle(color: palette.text, fontSize: 13),
                ),
              ),
              AnimatedContainer(
                duration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
                width: 24,
                height: 24,
                alignment: Alignment.center,
                decoration: BoxDecoration(
                  color: value ? palette.accent : Colors.transparent,
                  border: Border.all(
                    color: value ? palette.accent : palette.borderStrong,
                  ),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: value
                    ? Icon(
                        LucideIcons.check,
                        color: palette.accentText,
                        size: 16,
                      )
                    : null,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

final class _KeyPickerRow extends StatelessWidget {
  const _KeyPickerRow({
    super.key,
    required this.value,
    required this.selected,
    required this.onPressed,
  });

  final TerminalQuickKey value;
  final bool selected;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return InkWell(
      onTap: onPressed,
      child: Container(
        constraints: const BoxConstraints(minHeight: 52),
        padding: const EdgeInsets.symmetric(horizontal: 16),
        decoration: BoxDecoration(
          color: selected ? palette.accent.withValues(alpha: 0.08) : null,
          border: Border(bottom: BorderSide(color: palette.border)),
        ),
        child: Row(
          children: [
            Expanded(
              child: Text(
                value.label,
                style: TextStyle(
                  color: palette.text,
                  fontFamily: 'JetBrainsMonoNerd',
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            if (selected)
              Icon(LucideIcons.check, color: palette.accent, size: 18),
          ],
        ),
      ),
    );
  }
}

final class _TrayIconButton extends StatelessWidget {
  const _TrayIconButton({
    required this.tooltip,
    required this.icon,
    required this.onPressed,
    this.danger = false,
  });

  final String tooltip;
  final IconData icon;
  final VoidCallback onPressed;
  final bool danger;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return SizedBox.square(
      dimension: 48,
      child: Tooltip(
        message: tooltip,
        child: IconButton(
          onPressed: onPressed,
          icon: Icon(icon, size: 19),
          color: danger ? palette.danger : palette.muted,
        ),
      ),
    );
  }
}

final class _TrayTextButton extends StatelessWidget {
  const _TrayTextButton({
    required this.label,
    required this.enabled,
    required this.emphasized,
    required this.onPressed,
  });

  final String label;
  final bool enabled;
  final bool emphasized;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return ConstrainedBox(
      constraints: const BoxConstraints(minWidth: 56, minHeight: 44),
      child: TextButton(
        onPressed: enabled ? onPressed : null,
        style: TextButton.styleFrom(
          foregroundColor: emphasized ? palette.accent : palette.muted,
          disabledForegroundColor: palette.faint,
          padding: const EdgeInsets.symmetric(horizontal: 10),
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
        ),
        child: Text(
          label,
          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
        ),
      ),
    );
  }
}

Future<bool> _confirmDiscard(BuildContext context) async {
  final palette = AnyttyPalette.of(context);
  final result = await showGeneralDialog<bool>(
    context: context,
    barrierDismissible: true,
    barrierLabel: 'Discard changes',
    barrierColor: palette.overlay,
    transitionDuration: AnyttyMotion.resolve(context, AnyttyMotion.quick),
    pageBuilder: (context, _, _) => Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 320),
        child: Material(
          color: palette.surface,
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
            side: BorderSide(color: palette.border),
          ),
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Text(
                  'Discard changes?',
                  style: TextStyle(
                    color: palette.text,
                    fontSize: 16,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 8),
                Text(
                  'Your command bar changes have not been saved.',
                  style: TextStyle(color: palette.muted, fontSize: 13),
                ),
                const SizedBox(height: 20),
                Row(
                  mainAxisAlignment: MainAxisAlignment.end,
                  children: [
                    _TrayTextButton(
                      label: 'Keep editing',
                      enabled: true,
                      emphasized: false,
                      onPressed: () => Navigator.pop(context, false),
                    ),
                    const SizedBox(width: 4),
                    _TrayTextButton(
                      label: 'Discard',
                      enabled: true,
                      emphasized: true,
                      onPressed: () => Navigator.pop(context, true),
                    ),
                  ],
                ),
              ],
            ),
          ),
        ),
      ),
    ),
  );
  return result ?? false;
}
