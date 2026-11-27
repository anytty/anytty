part of 'terminal_workspace_screen.dart';

enum _TerminalAction { rename, restart, end, remove }

enum _TerminalToolAction {
  commandBar,
  quickKeys,
  keyboard,
  enter,
  escape,
  tab,
  backspace,
  delete,
  interrupt,
  eof,
  suspend,
  clear,
  arrowLeft,
  arrowDown,
  arrowUp,
  arrowRight,
  home,
  end,
  pageUp,
  pageDown,
  paste,
  history,
  search,
  selection,
  copyScreen,
  resources,
  files,
  split,
  splitRows,
  splitColumns,
  syncInput,
  resize,
  reconnect,
  settings,
}

enum _TerminalRowAction {
  rename,
  restart,
  end,
  remove,
  pin,
  unpin,
  moveUp,
  moveDown,
}

bool _terminalCanEnd(TerminalInfo terminal) =>
    terminal.state == TerminalState.TERMINAL_STATE_RUNNING ||
    terminal.state == TerminalState.TERMINAL_STATE_CREATED;

bool _terminalCanDelete(TerminalInfo terminal) =>
    terminal.state == TerminalState.TERMINAL_STATE_EXITED;

final class _TerminalCreateInput {
  const _TerminalCreateInput({
    required this.name,
    required this.command,
    required this.cwd,
    required this.environment,
    required this.cols,
    required this.rows,
    required this.sizeLockMode,
  });

  final String name;
  final List<String> command;
  final String cwd;
  final List<String> environment;
  final int cols;
  final int rows;
  final String sizeLockMode;
}

Future<void> _createTerminal({
  required BuildContext context,
  required WidgetRef ref,
  required String endpointId,
  required String label,
}) async {
  final messenger = ScaffoldMessenger.of(context);
  messenger.showSnackBar(
    const SnackBar(content: Text('Loading terminal defaults')),
  );
  try {
    final session = await ref.read(endpointSessionProvider(endpointId).future);
    final defaults = await session.terminalDefaults();
    messenger.hideCurrentSnackBar();
    if (!context.mounted) return;
    final input = await showModalBottomSheet<_TerminalCreateInput>(
      context: context,
      sheetAnimationStyle: AnimationStyle.noAnimation,
      isScrollControlled: true,
      showDragHandle: false,
      backgroundColor: Colors.transparent,
      barrierColor: Colors.black54,
      builder: (context) => _TerminalCreateSheet(
        defaults: defaults,
        loadDirectories: (prefix) => session.listDirectories(prefix: prefix),
      ),
    );
    if (input == null || !context.mounted) return;
    messenger.showSnackBar(const SnackBar(content: Text('Creating terminal')));
    final terminal = await session.createTerminal(
      name: input.name,
      command: input.command,
      cwd: input.cwd,
      environment: input.environment,
      cols: input.cols,
      rows: input.rows,
      sizeLockMode: input.sizeLockMode,
    );
    ref.invalidate(terminalListProvider(endpointId));
    if (!context.mounted) return;
    messenger.hideCurrentSnackBar();
    context.go(
      '/terminal/${Uri.encodeComponent(endpointId)}'
      '?terminalId=${Uri.encodeQueryComponent(terminal.ref.terminalId)}'
      '&label=${Uri.encodeQueryComponent(label)}',
    );
  } catch (error) {
    messenger.hideCurrentSnackBar();
    if (!context.mounted) return;
    messenger.showSnackBar(SnackBar(content: Text(error.toString())));
  }
}

Future<bool> _manageTerminal({
  required BuildContext context,
  required WidgetRef ref,
  required String endpointId,
  required TerminalInfo terminal,
  required _TerminalAction action,
}) async {
  String? nextName;
  if (action == _TerminalAction.rename) {
    final controller = TextEditingController(text: terminal.name);
    nextName = await showDialog<String>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Rename terminal'),
        content: TextField(
          controller: controller,
          autofocus: true,
          maxLength: 120,
          textInputAction: TextInputAction.done,
          onSubmitted: (value) => Navigator.pop(context, value.trim()),
          decoration: const InputDecoration(labelText: 'Name'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, controller.text.trim()),
            child: const Text('Rename'),
          ),
        ],
      ),
    );
    controller.dispose();
    if (nextName == null || !context.mounted) return false;
  }

  if (action == _TerminalAction.end || action == _TerminalAction.remove) {
    final destructive = action == _TerminalAction.remove;
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(destructive ? 'Delete terminal record?' : 'End process?'),
        content: Text(
          destructive
              ? 'The exited terminal and its saved history will be removed.'
              : 'The process will stop. Its terminal record and history will remain.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            style: destructive
                ? FilledButton.styleFrom(
                    backgroundColor: const Color(0xffdc2626),
                  )
                : null,
            onPressed: () => Navigator.pop(context, true),
            child: Text(destructive ? 'Delete' : 'End'),
          ),
        ],
      ),
    );
    if (confirmed != true || !context.mounted) return false;
  }

  try {
    final session = await ref.read(endpointSessionProvider(endpointId).future);
    switch (action) {
      case _TerminalAction.rename:
        await session.renameTerminal(terminal, nextName!);
      case _TerminalAction.restart:
        await session.restartTerminal(terminal.ref);
      case _TerminalAction.end:
        await session.killTerminal(terminal.ref);
      case _TerminalAction.remove:
        await session.removeTerminal(terminal.ref);
    }
    ref.invalidate(terminalListProvider(endpointId));
    if (context.mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text(_terminalActionSuccess(action))));
    }
    return true;
  } catch (error) {
    if (context.mounted) {
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.toString())));
    }
    return false;
  }
}

String _terminalActionSuccess(_TerminalAction action) {
  return switch (action) {
    _TerminalAction.rename => 'Terminal renamed',
    _TerminalAction.restart => 'Terminal restarted',
    _TerminalAction.end => 'Terminal process ended',
    _TerminalAction.remove => 'Terminal record deleted',
  };
}

final class _TerminalCreateSheet extends StatefulWidget {
  const _TerminalCreateSheet({
    required this.defaults,
    required this.loadDirectories,
  });

  final TerminalDefaults defaults;
  final TerminalDirectoryLoader loadDirectories;

  @override
  State<_TerminalCreateSheet> createState() => _TerminalCreateSheetState();
}

final class _TerminalCreateSheetState extends State<_TerminalCreateSheet> {
  late final TextEditingController _name;
  late final TextEditingController _command;
  late final TextEditingController _cwd;
  late final TextEditingController _environmentPaste;
  final List<_TerminalEnvironmentControllers> _environment = [];
  String _sizeLockMode = 'off';
  bool _pasteOpen = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _name = TextEditingController();
    _command = TextEditingController(
      text: formatTerminalCommand(widget.defaults.defaultCommand),
    );
    _cwd = TextEditingController(text: widget.defaults.defaultCwd);
    _environmentPaste = TextEditingController();
  }

  @override
  void dispose() {
    _name.dispose();
    _command.dispose();
    _cwd.dispose();
    _environmentPaste.dispose();
    for (final entry in _environment) {
      entry.dispose();
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final keyboardInset = MediaQuery.viewInsetsOf(context).bottom;
    return AnimatedPadding(
      duration: AnyttyMotion.resolve(
        context,
        const Duration(milliseconds: 200),
      ),
      curve: Curves.easeOutCubic,
      padding: EdgeInsets.only(bottom: keyboardInset),
      child: FractionallySizedBox(
        alignment: Alignment.bottomCenter,
        heightFactor: 0.85,
        child: Material(
          color: palette.background,
          clipBehavior: Clip.antiAlias,
          shape: RoundedRectangleBorder(
            borderRadius: const BorderRadius.vertical(top: Radius.circular(12)),
            side: BorderSide(color: palette.border),
          ),
          child: SafeArea(
            top: false,
            child: Column(
              children: [
                const SizedBox(height: 10),
                Container(
                  width: 48,
                  height: 4,
                  decoration: BoxDecoration(
                    color: palette.borderStrong,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
                SizedBox(
                  height: 54,
                  child: Row(
                    children: [
                      const SizedBox(width: 20),
                      Expanded(
                        child: Text(
                          anyttyText(context, en: 'New terminal', zh: '新建终端'),
                          style: TextStyle(
                            color: palette.text,
                            fontSize: 17,
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      IconButton(
                        tooltip: anyttyText(
                          context,
                          en: 'Close New terminal',
                          zh: '关闭新建终端',
                        ),
                        onPressed: () => Navigator.pop(context),
                        icon: const Icon(Icons.close_rounded, size: 20),
                      ),
                      const SizedBox(width: 8),
                    ],
                  ),
                ),
                Divider(height: 1, color: palette.border),
                Expanded(
                  child: ListView(
                    padding: const EdgeInsets.fromLTRB(16, 16, 16, 20),
                    children: [
                      _labelledField(
                        context,
                        label: anyttyText(context, en: 'Name', zh: '名称'),
                        child: TextField(
                          controller: _name,
                          maxLength: 120,
                          textInputAction: TextInputAction.next,
                          decoration: _fieldDecoration(
                            context,
                            counterText: '',
                          ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _labelledField(
                        context,
                        label: anyttyText(context, en: 'Command', zh: '命令'),
                        child: TextField(
                          controller: _command,
                          autocorrect: false,
                          enableSuggestions: false,
                          textInputAction: TextInputAction.next,
                          style: const TextStyle(
                            fontFamily: 'JetBrainsMonoNerd',
                            fontSize: 15,
                          ),
                          decoration: _fieldDecoration(context),
                        ),
                      ),
                      const SizedBox(height: 16),
                      _labelledField(
                        context,
                        label: anyttyText(
                          context,
                          en: 'Working directory',
                          zh: '工作目录',
                        ),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Expanded(
                              child: TextField(
                                controller: _cwd,
                                autocorrect: false,
                                enableSuggestions: false,
                                textInputAction: TextInputAction.next,
                                style: const TextStyle(
                                  fontFamily: 'JetBrainsMonoNerd',
                                  fontSize: 15,
                                ),
                                decoration: _fieldDecoration(context),
                              ),
                            ),
                            const SizedBox(width: 8),
                            SizedBox(
                              width: 48,
                              height: 48,
                              child: IconButton(
                                key: const ValueKey(
                                  'terminal-browse-working-directory',
                                ),
                                onPressed: _browseWorkingDirectory,
                                tooltip: anyttyText(
                                  context,
                                  en: 'Browse working directory',
                                  zh: '浏览工作目录',
                                ),
                                icon: const Icon(LucideIcons.folderOpen),
                              ),
                            ),
                          ],
                        ),
                      ),
                      const SizedBox(height: 16),
                      _buildEnvironmentEditor(context),
                      const SizedBox(height: 16),
                      _labelledField(
                        context,
                        label: anyttyText(
                          context,
                          en: 'Size policy',
                          zh: '尺寸策略',
                        ),
                        child: DropdownButtonFormField<String>(
                          initialValue: _sizeLockMode,
                          isExpanded: true,
                          decoration: _fieldDecoration(context),
                          items: [
                            DropdownMenuItem(
                              value: 'off',
                              child: Text(
                                anyttyText(
                                  context,
                                  en: 'Resizable',
                                  zh: '可调整大小',
                                ),
                              ),
                            ),
                            DropdownMenuItem(
                              value: 'warn',
                              child: Text(
                                anyttyText(
                                  context,
                                  en: 'Warn before resizing',
                                  zh: '调整大小前提醒',
                                ),
                              ),
                            ),
                            DropdownMenuItem(
                              value: 'lock',
                              child: Text(
                                anyttyText(
                                  context,
                                  en: 'Locked size',
                                  zh: '锁定尺寸',
                                ),
                              ),
                            ),
                          ],
                          onChanged: (value) {
                            if (value != null) _sizeLockMode = value;
                          },
                        ),
                      ),
                      if (_error case final error?) ...[
                        const SizedBox(height: 12),
                        Text(
                          error,
                          style: const TextStyle(
                            color: Color(0xffdc2626),
                            fontSize: 13,
                            fontWeight: FontWeight.w500,
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
                Container(
                  decoration: BoxDecoration(
                    color: palette.background,
                    border: Border(top: BorderSide(color: palette.border)),
                  ),
                  padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
                  child: SizedBox(
                    width: double.infinity,
                    height: 48,
                    child: FilledButton.icon(
                      onPressed: _submit,
                      icon: const Icon(Icons.add_rounded, size: 18),
                      label: Text(
                        anyttyText(context, en: 'Create terminal', zh: '创建终端'),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildEnvironmentEditor(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          anyttyText(context, en: 'Environment', zh: '环境变量'),
          style: TextStyle(
            color: palette.text,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        for (var index = 0; index < _environment.length; index++) ...[
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                flex: 9,
                child: TextField(
                  controller: _environment[index].key,
                  autocorrect: false,
                  enableSuggestions: false,
                  decoration: _fieldDecoration(
                    context,
                    hintText: anyttyText(context, en: 'Key', zh: '键'),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                flex: 11,
                child: TextField(
                  controller: _environment[index].value,
                  autocorrect: false,
                  enableSuggestions: false,
                  decoration: _fieldDecoration(
                    context,
                    hintText: anyttyText(context, en: 'Value', zh: '值'),
                  ),
                ),
              ),
              const SizedBox(width: 8),
              SizedBox(
                width: 48,
                height: 48,
                child: OutlinedButton(
                  onPressed: () => _removeEnvironment(index),
                  style: OutlinedButton.styleFrom(
                    padding: EdgeInsets.zero,
                    foregroundColor: palette.muted,
                    side: BorderSide(color: palette.borderStrong),
                  ),
                  child: const Icon(Icons.delete_outline_rounded, size: 18),
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
        ],
        Row(
          children: [
            Expanded(
              child: SizedBox(
                height: 48,
                child: OutlinedButton.icon(
                  onPressed: _addEnvironment,
                  icon: const Icon(Icons.add_rounded, size: 18),
                  label: Text(anyttyText(context, en: 'Add', zh: '添加')),
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: SizedBox(
                height: 48,
                child: OutlinedButton.icon(
                  onPressed: () => setState(() => _pasteOpen = true),
                  icon: const Icon(Icons.content_paste_rounded, size: 17),
                  label: Text(anyttyText(context, en: 'Paste', zh: '粘贴')),
                ),
              ),
            ),
          ],
        ),
        if (_pasteOpen) ...[
          const SizedBox(height: 8),
          Container(
            decoration: BoxDecoration(
              color: palette.surface,
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(6),
            ),
            padding: const EdgeInsets.all(12),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                Row(
                  children: [
                    Expanded(
                      child: Text(
                        anyttyText(
                          context,
                          en: 'Paste environment',
                          zh: '粘贴环境变量',
                        ),
                        style: TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                    ),
                    IconButton(
                      tooltip: anyttyText(
                        context,
                        en: 'Close environment paste',
                        zh: '关闭环境变量粘贴',
                      ),
                      onPressed: () => setState(() => _pasteOpen = false),
                      icon: const Icon(Icons.close_rounded, size: 18),
                    ),
                  ],
                ),
                TextField(
                  controller: _environmentPaste,
                  minLines: 4,
                  maxLines: 7,
                  autocorrect: false,
                  enableSuggestions: false,
                  style: const TextStyle(
                    fontFamily: 'JetBrainsMonoNerd',
                    fontSize: 14,
                  ),
                  decoration: _fieldDecoration(
                    context,
                    hintText: 'KEY=value\nexport OTHER=value',
                  ),
                ),
                const SizedBox(height: 8),
                SizedBox(
                  height: 48,
                  child: FilledButton(
                    onPressed: _applyEnvironmentPaste,
                    child: Text(
                      anyttyText(
                        context,
                        en: 'Apply environment',
                        zh: '应用环境变量',
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ],
      ],
    );
  }

  Widget _labelledField(
    BuildContext context, {
    required String label,
    required Widget child,
  }) {
    final palette = AnyttyPalette.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          label,
          style: TextStyle(
            color: palette.text,
            fontSize: 14,
            fontWeight: FontWeight.w600,
          ),
        ),
        const SizedBox(height: 8),
        child,
      ],
    );
  }

  InputDecoration _fieldDecoration(
    BuildContext context, {
    String? hintText,
    String? counterText,
  }) {
    final palette = AnyttyPalette.of(context);
    return InputDecoration(
      hintText: hintText,
      counterText: counterText,
      filled: true,
      fillColor: palette.surface,
      isDense: true,
      contentPadding: const EdgeInsets.symmetric(horizontal: 14, vertical: 14),
      enabledBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(6),
        borderSide: BorderSide(color: palette.borderStrong),
      ),
      focusedBorder: OutlineInputBorder(
        borderRadius: BorderRadius.circular(6),
        borderSide: BorderSide(color: palette.accent, width: 2),
      ),
    );
  }

  void _addEnvironment() {
    setState(() => _environment.add(_TerminalEnvironmentControllers()));
  }

  Future<void> _browseWorkingDirectory() async {
    FocusManager.instance.primaryFocus?.unfocus();
    final selected = await showTerminalPathPicker(
      context: context,
      initialPath: _cwd.text.trim().isEmpty
          ? widget.defaults.defaultCwd
          : _cwd.text.trim(),
      loadDirectories: widget.loadDirectories,
    );
    if (!mounted || selected == null) return;
    _cwd.value = TextEditingValue(
      text: selected,
      selection: TextSelection.collapsed(offset: selected.length),
    );
  }

  void _removeEnvironment(int index) {
    final removed = _environment.removeAt(index);
    removed.dispose();
    setState(() {});
  }

  void _applyEnvironmentPaste() {
    try {
      final parsed = parseTerminalEnvironment(_environmentPaste.text);
      for (final entry in _environment) {
        entry.dispose();
      }
      _environment
        ..clear()
        ..addAll(parsed.map(_TerminalEnvironmentControllers.fromEntry));
      setState(() {
        _environmentPaste.clear();
        _pasteOpen = false;
        _error = null;
      });
    } on FormatException catch (error) {
      setState(() => _error = error.message);
    }
  }

  void _submit() {
    try {
      final command = parseTerminalCommand(_command.text);
      final environment = parseTerminalEnvironment(
        _environment
            .where(
              (entry) =>
                  entry.key.text.trim().isNotEmpty ||
                  entry.value.text.isNotEmpty,
            )
            .map((entry) => '${entry.key.text.trim()}=${entry.value.text}')
            .join('\n'),
      );
      Navigator.pop(
        context,
        _TerminalCreateInput(
          name: _name.text.trim(),
          command: command,
          cwd: _cwd.text.trim(),
          environment: environment,
          cols: 80,
          rows: 24,
          sizeLockMode: _sizeLockMode,
        ),
      );
    } on FormatException catch (error) {
      setState(() => _error = error.message);
    }
  }
}

final class _TerminalEnvironmentControllers {
  _TerminalEnvironmentControllers({String key = '', String value = ''})
    : key = TextEditingController(text: key),
      value = TextEditingController(text: value);

  factory _TerminalEnvironmentControllers.fromEntry(String entry) {
    final separator = entry.indexOf('=');
    return _TerminalEnvironmentControllers(
      key: separator < 0 ? entry : entry.substring(0, separator),
      value: separator < 0 ? '' : entry.substring(separator + 1),
    );
  }

  final TextEditingController key;
  final TextEditingController value;

  void dispose() {
    key.dispose();
    value.dispose();
  }
}
