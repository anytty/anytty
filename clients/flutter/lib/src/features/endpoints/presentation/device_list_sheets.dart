part of 'device_list_screen.dart';

final class _DeviceActionsSheet extends ConsumerStatefulWidget {
  const _DeviceActionsSheet({required this.endpoint, required this.isDefault});

  final EndpointConfigV1 endpoint;
  final bool isDefault;

  @override
  ConsumerState<_DeviceActionsSheet> createState() =>
      _DeviceActionsSheetState();
}

final class _DeviceActionsSheetState
    extends ConsumerState<_DeviceActionsSheet> {
  late final TextEditingController _labelController;
  bool _busy = false;
  String? _error;

  String get _label => widget.endpoint.label.trim().isEmpty
      ? widget.endpoint.endpointId
      : widget.endpoint.label.trim();

  @override
  void initState() {
    super.initState();
    _labelController = TextEditingController(text: _label);
  }

  @override
  void dispose() {
    _labelController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Padding(
      padding: EdgeInsets.only(
        bottom:
            MediaQuery.viewInsetsOf(context).bottom +
            MediaQuery.viewPaddingOf(context).bottom,
      ),
      child: SingleChildScrollView(
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 576),
            child: Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 20),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    children: [
                      Container(
                        width: 40,
                        height: 40,
                        alignment: Alignment.center,
                        decoration: BoxDecoration(
                          color: palette.surfaceRaised,
                          border: Border.all(color: palette.border),
                          borderRadius: BorderRadius.circular(6),
                        ),
                        child: Icon(
                          _platformIcon(widget.endpoint.platform),
                          size: 22,
                        ),
                      ),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              _label,
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                              style: const TextStyle(
                                fontSize: 17,
                                fontWeight: FontWeight.w600,
                              ),
                            ),
                            Text(
                              anyttyText(
                                context,
                                en: 'Device actions',
                                zh: '设备操作',
                              ),
                              style: TextStyle(
                                color: palette.muted,
                                fontSize: 12,
                              ),
                            ),
                          ],
                        ),
                      ),
                      IconButton(
                        tooltip: anyttyText(
                          context,
                          en: 'Close device actions',
                          zh: '关闭设备操作',
                        ),
                        onPressed: _busy ? null : () => Navigator.pop(context),
                        icon: const Icon(Icons.close_rounded),
                      ),
                    ],
                  ),
                  const SizedBox(height: 20),
                  TextField(
                    controller: _labelController,
                    enabled: !_busy,
                    textInputAction: TextInputAction.done,
                    onSubmitted: (_) => _saveLabel(),
                    decoration: InputDecoration(
                      labelText: anyttyText(
                        context,
                        en: 'Display name',
                        zh: '显示名称',
                      ),
                      suffixIcon: IconButton(
                        tooltip: anyttyText(
                          context,
                          en: 'Save display name',
                          zh: '保存显示名称',
                        ),
                        onPressed: _busy ? null : _saveLabel,
                        icon: const Icon(Icons.check_rounded),
                      ),
                    ),
                  ),
                  if (_busy) ...[
                    const SizedBox(height: 12),
                    const LinearProgressIndicator(minHeight: 2),
                  ],
                  if (_error != null) ...[
                    const SizedBox(height: 12),
                    Semantics(
                      liveRegion: true,
                      child: Text(
                        _error!,
                        style: TextStyle(color: palette.danger, fontSize: 12),
                      ),
                    ),
                  ],
                  const SizedBox(height: 12),
                  Divider(height: 1, color: palette.border),
                  if (_authorizationRequired(widget.endpoint)) ...[
                    ListTile(
                      contentPadding: EdgeInsets.zero,
                      enabled: !_busy,
                      leading: Icon(
                        Icons.shield_outlined,
                        color: palette.warning,
                      ),
                      title: Text(
                        anyttyText(context, en: 'Authorize device', zh: '授权设备'),
                      ),
                      subtitle: Text(
                        anyttyText(
                          context,
                          en: 'Fresh pairing required',
                          zh: '需要重新配对',
                        ),
                      ),
                      trailing: Icon(
                        Icons.chevron_right_rounded,
                        color: palette.faint,
                      ),
                      onTap: _authorize,
                    ),
                    Divider(height: 1, color: palette.border),
                  ],
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy && !widget.isDefault,
                    leading: const Icon(Icons.star_outline_rounded),
                    title: Text(
                      widget.isDefault
                          ? anyttyText(
                              context,
                              en: 'Default device',
                              zh: '默认设备',
                            )
                          : anyttyText(
                              context,
                              en: 'Set as default',
                              zh: '设为默认设备',
                            ),
                    ),
                    trailing: widget.isDefault
                        ? Icon(Icons.check_rounded, color: palette.success)
                        : null,
                    onTap: widget.isDefault ? null : _makeDefault,
                  ),
                  Divider(height: 1, color: palette.border),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy,
                    leading: const Icon(Icons.route_outlined),
                    title: Text(
                      anyttyText(context, en: 'Connection', zh: '网络连接'),
                    ),
                    subtitle: Text(
                      anyttyText(
                        context,
                        en: 'Route, latency, traffic, and policy',
                        zh: '线路、延迟与连接策略',
                      ),
                    ),
                    trailing: Icon(
                      Icons.chevron_right_rounded,
                      color: palette.faint,
                    ),
                    onTap: _openConnection,
                  ),
                  Divider(height: 1, color: palette.border),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy,
                    leading: const Icon(Icons.link_off_rounded),
                    title: Text(
                      anyttyText(context, en: 'Disconnect', zh: '断开连接'),
                    ),
                    subtitle: Text(
                      anyttyText(
                        context,
                        en: 'Close the current pooled connection',
                        zh: '关闭当前复用连接',
                      ),
                    ),
                    onTap: _disconnect,
                  ),
                  Divider(height: 1, color: palette.border),
                  ListTile(
                    contentPadding: EdgeInsets.zero,
                    enabled: !_busy,
                    leading: Icon(
                      Icons.delete_outline_rounded,
                      color: palette.danger,
                    ),
                    title: Text(
                      anyttyText(context, en: 'Remove device', zh: '移除设备'),
                      style: TextStyle(color: palette.danger),
                    ),
                    subtitle: Text(
                      anyttyText(
                        context,
                        en: 'Remove saved routes and authorization',
                        zh: '删除已保存的线路与授权',
                      ),
                    ),
                    onTap: _remove,
                  ),
                ],
              ),
            ),
          ),
        ),
      ),
    );
  }

  Future<EndpointRepository> _repository() async {
    final runtime = await ref.read(anyttyRuntimeProvider.future);
    return EndpointRepository(runtime);
  }

  Future<void> _saveLabel() async {
    final label = _labelController.text.trim();
    if (label.isEmpty) {
      setState(
        () => _error = anyttyText(
          context,
          en: 'Display name is required',
          zh: '请输入显示名称',
        ),
      );
      return;
    }
    if (label == widget.endpoint.label.trim()) return;
    final endpoint = widget.endpoint.deepCopy()
      ..label = label
      ..labelSource = EndpointSource.ENDPOINT_SOURCE_USER;
    await _perform(
      successMessage: anyttyText(
        context,
        en: 'Device name updated',
        zh: '设备名称已更新',
      ),
      action: () async {
        await (await _repository()).upsertEndpoint(
          endpoint,
          makeDefault: widget.isDefault,
        );
        ref.invalidate(endpointRegistryProvider);
      },
    );
  }

  Future<void> _makeDefault() => _perform(
    successMessage: anyttyText(
      context,
      en: 'Default device updated',
      zh: '默认设备已更新',
    ),
    action: () async {
      await (await _repository()).upsertEndpoint(
        widget.endpoint,
        makeDefault: true,
      );
      ref.invalidate(endpointRegistryProvider);
    },
  );

  void _openConnection() {
    final router = GoRouter.of(context);
    final location =
        '/connection/${Uri.encodeComponent(widget.endpoint.endpointId)}'
        '?label=${Uri.encodeQueryComponent(_label)}';
    Navigator.pop(context);
    unawaited(router.push(location));
  }

  void _authorize() {
    final navigator = Navigator.of(context, rootNavigator: true);
    final route = MaterialPageRoute<void>(
      fullscreenDialog: true,
      builder: (_) => _PairingSheet(reauthorizeEndpoint: widget.endpoint),
    );
    Navigator.pop(context);
    unawaited(navigator.push(route));
  }

  Future<void> _disconnect() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(
          anyttyText(context, en: 'Disconnect device?', zh: '断开设备连接？'),
        ),
        content: Text(
          anyttyText(
            context,
            en: 'The current connection to $_label will be closed.',
            zh: '将关闭与 $_label 的当前连接。',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(context, true),
            child: Text(anyttyText(context, en: 'Disconnect', zh: '断开')),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    // 显式断开：先释放 App 级连接保留，避免保留租约触发自动重连。
    await releaseEndpointConnection(ref, widget.endpoint.endpointId);
    if (!mounted) return;
    await _perform(
      successMessage: anyttyText(
        context,
        en: 'Device disconnected',
        zh: '设备连接已断开',
      ),
      action: () async {
        await (await _repository()).disconnectEndpoint(
          widget.endpoint.endpointId,
        );
      },
    );
  }

  Future<void> _remove() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(anyttyText(context, en: 'Remove device?', zh: '移除设备？')),
        content: Text(
          anyttyText(
            context,
            en: '$_label and its saved routes and authorization will be removed from this app.',
            zh: '将从本应用移除 $_label，以及已保存的线路和授权。',
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context, false),
            child: Text(anyttyText(context, en: 'Cancel', zh: '取消')),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red.shade700),
            onPressed: () => Navigator.pop(context, true),
            child: Text(anyttyText(context, en: 'Remove', zh: '移除')),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    // 移除设备同样释放 App 级连接保留。
    await releaseEndpointConnection(ref, widget.endpoint.endpointId);
    if (!mounted) return;
    await _perform(
      successMessage: anyttyText(context, en: 'Device removed', zh: '设备已移除'),
      action: () async {
        final endpointId = widget.endpoint.endpointId;
        await (await _repository()).deleteEndpoint(endpointId);
        await Future.wait([
          clearBrowserDeviceData(endpointId),
          const FileManagerPathStore().remove(endpointId),
        ]);
        ref.invalidate(endpointRegistryProvider);
      },
    );
  }

  Future<void> _perform({
    required String successMessage,
    required Future<void> Function() action,
  }) async {
    if (_busy) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await action();
      if (!mounted) return;
      final messenger = ScaffoldMessenger.of(context);
      Navigator.pop(context);
      messenger.showSnackBar(SnackBar(content: Text(successMessage)));
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _busy = false;
        _error = error.toString();
      });
    }
  }
}

final class _EmptyDevices extends StatelessWidget {
  const _EmptyDevices({required this.onPair});

  final VoidCallback onPair;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(24),
      decoration: BoxDecoration(
        color: palette.surface,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            width: 40,
            height: 40,
            alignment: Alignment.center,
            decoration: BoxDecoration(
              color: palette.surfaceRaised,
              border: Border.all(color: palette.border),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(Icons.dns_outlined, size: 24, color: palette.text),
          ),
          const SizedBox(height: 20),
          Text(
            anyttyText(context, en: 'No paired devices', zh: '暂无已配对设备'),
            style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Text(
            anyttyText(
              context,
              en: 'Pair this app with an AnyTTY endpoint to begin.',
              zh: '与 AnyTTY 端点配对后即可开始使用。',
            ),
            style: TextStyle(color: palette.muted, height: 1.5),
          ),
          const SizedBox(height: 24),
          SizedBox(
            width: double.infinity,
            height: 48,
            child: FilledButton.icon(
              onPressed: onPair,
              icon: const Icon(Icons.qr_code_scanner_rounded, size: 18),
              label: Text(anyttyText(context, en: 'Scan service', zh: '扫描服务')),
            ),
          ),
        ],
      ),
    );
  }
}

final class _RegistryError extends StatelessWidget {
  const _RegistryError({
    required this.message,
    required this.busy,
    required this.onRetry,
    required this.onCopyDiagnostics,
    required this.onReset,
  });

  final String message;
  final bool busy;
  final VoidCallback onRetry;
  final VoidCallback onCopyDiagnostics;
  final VoidCallback onReset;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.cloud_off_outlined, color: palette.danger, size: 36),
            const SizedBox(height: 12),
            const Text(
              'Could not load devices',
              style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
            ),
            const SizedBox(height: 6),
            Text(
              message,
              maxLines: 3,
              overflow: TextOverflow.ellipsis,
              textAlign: TextAlign.center,
              style: TextStyle(color: palette.muted, fontSize: 12),
            ),
            const SizedBox(height: 18),
            ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 48),
              child: FilledButton.icon(
                onPressed: busy ? null : onRetry,
                icon: busy
                    ? const SizedBox.square(
                        dimension: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.refresh_rounded),
                label: const Text('Retry startup'),
              ),
            ),
            const SizedBox(height: 8),
            ConstrainedBox(
              constraints: const BoxConstraints(minHeight: 48),
              child: OutlinedButton.icon(
                onPressed: busy ? null : onCopyDiagnostics,
                icon: const Icon(Icons.copy_all_rounded),
                label: const Text('Copy redacted diagnostics'),
              ),
            ),
            const SizedBox(height: 4),
            TextButton(
              onPressed: busy ? null : onReset,
              style: TextButton.styleFrom(
                minimumSize: const Size(48, 48),
                foregroundColor: palette.danger,
              ),
              child: const Text('Reset saved devices'),
            ),
          ],
        ),
      ),
    );
  }
}

final class _DeviceLoading extends StatelessWidget {
  const _DeviceLoading();

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Center(
      child: Semantics(
        liveRegion: true,
        label: 'Loading devices',
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            SizedBox.square(
              dimension: 18,
              child: CircularProgressIndicator(
                color: palette.text,
                strokeWidth: 2,
              ),
            ),
            const SizedBox(height: 12),
            Text(
              'Loading devices',
              style: TextStyle(color: palette.muted, fontSize: 13),
            ),
          ],
        ),
      ),
    );
  }
}

IconData _platformIcon(String value) {
  return switch (value.toLowerCase()) {
    'darwin' || 'macos' => Icons.laptop_mac_rounded,
    'windows' => Icons.desktop_windows_rounded,
    'linux' => Icons.terminal_rounded,
    _ => Icons.computer_rounded,
  };
}

Future<void> _showPairingSheet(
  BuildContext context, {
  EndpointConfigV1? reauthorizeEndpoint,
}) async {
  await Navigator.of(context).push<void>(
    MaterialPageRoute(
      fullscreenDialog: true,
      builder: (_) => _PairingSheet(reauthorizeEndpoint: reauthorizeEndpoint),
    ),
  );
}
