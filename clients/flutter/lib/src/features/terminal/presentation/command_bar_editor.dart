part of 'terminal_command_bar.dart';


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
