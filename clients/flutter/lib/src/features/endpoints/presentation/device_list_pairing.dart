part of 'device_list_screen.dart';

final class _PairingSheet extends ConsumerStatefulWidget {
  const _PairingSheet({this.reauthorizeEndpoint});

  final EndpointConfigV1? reauthorizeEndpoint;

  @override
  ConsumerState<_PairingSheet> createState() => _PairingSheetState();
}

final class _PairingSheetState extends ConsumerState<_PairingSheet> {
  final _controller = TextEditingController();
  final _scanner = MobileScannerController();
  bool _submitting = false;
  bool _scanAccepted = false;
  bool _pasteOpen = false;
  EndpointSharePreview? _sharePreview;
  String? _error;

  @override
  void dispose() {
    _controller.dispose();
    _scanner.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final sharePreview = _sharePreview;
    final reauthorizeEndpoint = widget.reauthorizeEndpoint;
    final reauthorizeLabel = reauthorizeEndpoint == null
        ? null
        : reauthorizeEndpoint.label.trim().isEmpty
        ? reauthorizeEndpoint.endpointId
        : reauthorizeEndpoint.label.trim();
    return Scaffold(
      appBar: AppBar(
        leading: Padding(
          padding: EdgeInsets.only(left: 12),
          child: Icon(
            sharePreview == null
                ? reauthorizeEndpoint == null
                      ? Icons.qr_code_scanner_rounded
                      : Icons.shield_outlined
                : Icons.move_to_inbox_outlined,
            size: 20,
          ),
        ),
        leadingWidth: 44,
        titleSpacing: 0,
        title: Text(
          sharePreview != null
              ? 'Import configuration'
              : reauthorizeEndpoint == null
              ? 'Pair device'
              : 'Authorize device',
          style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
        ),
        actions: [
          IconButton(
            tooltip: 'Close',
            onPressed: _submitting ? null : () => Navigator.pop(context),
            icon: const Icon(Icons.close_rounded),
          ),
        ],
      ),
      body: sharePreview == null
          ? SingleChildScrollView(
              padding: EdgeInsets.fromLTRB(
                16,
                20,
                16,
                24 + MediaQuery.viewInsetsOf(context).bottom,
              ),
              child: Center(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 448),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      if (reauthorizeLabel != null) ...[
                        _ReauthorizationTarget(
                          endpoint: reauthorizeEndpoint!,
                          label: reauthorizeLabel,
                        ),
                        const SizedBox(height: 12),
                      ],
                      Container(
                        height: 336,
                        padding: const EdgeInsets.all(8),
                        decoration: BoxDecoration(
                          color: const Color(0xff09090b),
                          border: Border.all(color: const Color(0xff27272a)),
                          borderRadius: BorderRadius.circular(12),
                        ),
                        child: ClipRRect(
                          borderRadius: BorderRadius.circular(8),
                          child: Stack(
                            fit: StackFit.expand,
                            children: [
                              MobileScanner(
                                controller: _scanner,
                                onDetect: _handleBarcode,
                              ),
                              Center(
                                child: IgnorePointer(
                                  child: Container(
                                    width: 216,
                                    height: 216,
                                    decoration: BoxDecoration(
                                      border: Border.all(
                                        color: Colors.white.withValues(
                                          alpha: 0.72,
                                        ),
                                        width: 2,
                                      ),
                                      borderRadius: BorderRadius.circular(8),
                                    ),
                                  ),
                                ),
                              ),
                              Positioned(
                                top: 4,
                                right: 4,
                                child: IconButton(
                                  tooltip: 'Toggle flashlight',
                                  color: Colors.white,
                                  style: IconButton.styleFrom(
                                    backgroundColor: Colors.black54,
                                  ),
                                  onPressed: _submitting
                                      ? null
                                      : _scanner.toggleTorch,
                                  icon: const Icon(
                                    Icons.flashlight_on_outlined,
                                    size: 20,
                                  ),
                                ),
                              ),
                              if (_submitting)
                                const ColoredBox(
                                  color: Color(0x9909090b),
                                  child: Center(
                                    child: CircularProgressIndicator(
                                      color: Colors.white,
                                      strokeWidth: 2,
                                    ),
                                  ),
                                ),
                            ],
                          ),
                        ),
                      ),
                      const SizedBox(height: 16),
                      OutlinedButton(
                        onPressed: _submitting
                            ? null
                            : () => setState(() => _pasteOpen = !_pasteOpen),
                        style: OutlinedButton.styleFrom(
                          minimumSize: const Size.fromHeight(48),
                          padding: const EdgeInsets.symmetric(horizontal: 12),
                        ),
                        child: Row(
                          children: [
                            const Icon(Icons.content_paste_rounded, size: 17),
                            const SizedBox(width: 8),
                            const Expanded(
                              child: Text(
                                'Paste pairing payload',
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            AnimatedRotation(
                              turns: _pasteOpen ? 0.5 : 0,
                              duration: const Duration(milliseconds: 200),
                              child: const Icon(
                                Icons.keyboard_arrow_down_rounded,
                                size: 18,
                              ),
                            ),
                          ],
                        ),
                      ),
                      if (_pasteOpen) ...[
                        const SizedBox(height: 16),
                        const Text(
                          'Pairing payload',
                          style: TextStyle(
                            fontSize: 14,
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        const SizedBox(height: 8),
                        TextField(
                          controller: _controller,
                          minLines: 4,
                          maxLines: 7,
                          enabled: !_submitting,
                          autocorrect: false,
                          decoration: InputDecoration(
                            hintText: 'Paste the pairing payload',
                            alignLabelWithHint: true,
                            errorText: _error,
                            errorMaxLines: 4,
                          ),
                          onChanged: (_) => setState(() => _error = null),
                        ),
                        const SizedBox(height: 12),
                        SizedBox(
                          height: 48,
                          child: FilledButton(
                            onPressed:
                                _submitting || _controller.text.trim().isEmpty
                                ? null
                                : _submit,
                            child: _submitting
                                ? const SizedBox.square(
                                    dimension: 18,
                                    child: CircularProgressIndicator(
                                      strokeWidth: 2,
                                    ),
                                  )
                                : const Text('Continue'),
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ),
            )
          : _EndpointSharePreviewView(
              preview: sharePreview,
              submitting: _submitting,
              error: _error,
              onCancel: _cancelSharePreview,
              onCommit: _commitShare,
            ),
    );
  }

  void _handleBarcode(BarcodeCapture capture) {
    if (_scanAccepted || _submitting) return;
    for (final barcode in capture.barcodes) {
      final value = barcode.rawValue?.trim();
      if (value == null || value.isEmpty) continue;
      _scanAccepted = true;
      _controller.text = value;
      setState(() => _error = null);
      _submit();
      return;
    }
  }

  Future<void> _submit() async {
    FocusManager.instance.primaryFocus?.unfocus();
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final repository = EndpointRepository(runtime);
      final input = _controller.text.trim();
      if (input.startsWith('anytty://share?payload=')) {
        if (widget.reauthorizeEndpoint != null) {
          throw const AnyttyOperationException(
            'A fresh pairing code is required to authorize this device',
          );
        }
        final preview = await repository.receiveEndpointShare(input);
        await _scanner.stop();
        if (!mounted) return;
        setState(() {
          _sharePreview = preview;
          _submitting = false;
          _pasteOpen = false;
        });
        return;
      }
      final result = await repository.importPairing(
        input,
        expectedEndpointId: widget.reauthorizeEndpoint?.endpointId,
      );
      if (!mounted) return;
      final messenger = ScaffoldMessenger.of(context);
      final label = result.endpoint.label.trim().isEmpty
          ? result.endpoint.endpointId
          : result.endpoint.label.trim();
      ref.invalidate(endpointRegistryProvider);
      Navigator.pop(context);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            result.authorizationRequired
                ? '$label still requires authorization'
                : widget.reauthorizeEndpoint == null
                ? '$label paired'
                : '$label authorized',
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _scanAccepted = false;
        _error = error.toString();
        _pasteOpen = true;
      });
    }
  }

  void _cancelSharePreview() {
    if (_submitting) return;
    setState(() {
      _sharePreview = null;
      _scanAccepted = false;
      _error = null;
    });
  }

  Future<void> _commitShare() async {
    final preview = _sharePreview;
    if (_submitting || preview == null) return;
    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final runtime = await ref.read(anyttyRuntimeProvider.future);
      final result = await EndpointRepository(runtime)
          .commitEndpointShare(preview.importToken);
      if (!mounted) return;
      final messenger = ScaffoldMessenger.of(context);
      final label = result.endpoint.label.trim().isEmpty
          ? result.endpoint.endpointId
          : result.endpoint.label.trim();
      ref.invalidate(endpointRegistryProvider);
      Navigator.pop(context);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            result.authorizationRequired
                ? '$label imported. Authorization is still required.'
                : '$label configuration imported',
          ),
        ),
      );
    } catch (error) {
      if (!mounted) return;
      setState(() {
        _submitting = false;
        _error = error.toString();
      });
    }
  }
}

final class _ReauthorizationTarget extends StatelessWidget {
  const _ReauthorizationTarget({required this.endpoint, required this.label});

  final EndpointConfigV1 endpoint;
  final String label;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Semantics(
      label: '$label, authorization required',
      child: Container(
        padding: const EdgeInsets.all(12),
        decoration: BoxDecoration(
          color: palette.surfaceRaised,
          border: Border.all(color: palette.border),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          children: [
            Icon(
              _platformIcon(endpoint.platform),
              color: palette.text,
              size: 22,
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    label,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  Text(
                    endpoint.endpointId,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(color: palette.muted, fontSize: 11),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 10),
            Text(
              'FRESH PAIRING',
              style: TextStyle(
                color: palette.warning,
                fontSize: 10,
                fontWeight: FontWeight.w700,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

final class _EndpointSharePreviewView extends StatelessWidget {
  const _EndpointSharePreviewView({
    required this.preview,
    required this.submitting,
    required this.error,
    required this.onCancel,
    required this.onCommit,
  });

  final EndpointSharePreview preview;
  final bool submitting;
  final String? error;
  final VoidCallback onCancel;
  final VoidCallback onCommit;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final label = preview.label.trim().isEmpty
        ? preview.endpointId
        : preview.label.trim();
    final fingerprint = preview.hasIdentity()
        ? preview.identity.deviceFingerprint.trim()
        : '';
    final policyChanges = <String>[
      if (preview.connectModeChanged) 'Connection mode',
      if (preview.selectionPolicyChanged) 'Route preference',
    ];
    return SafeArea(
      top: false,
      child: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 520),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Container(
                    padding: const EdgeInsets.all(16),
                    decoration: BoxDecoration(
                      color: palette.surface,
                      border: Border.all(color: palette.border),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Container(
                          width: 44,
                          height: 44,
                          alignment: Alignment.center,
                          decoration: BoxDecoration(
                            color: palette.surfaceRaised,
                            border: Border.all(color: palette.border),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Icon(
                            Icons.computer_rounded,
                            size: 23,
                            color: palette.text,
                          ),
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                label,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: const TextStyle(
                                  fontSize: 17,
                                  fontWeight: FontWeight.w600,
                                ),
                              ),
                              const SizedBox(height: 2),
                              Text(
                                preview.endpointId,
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                                style: TextStyle(
                                  color: palette.muted,
                                  fontSize: 12,
                                ),
                              ),
                            ],
                          ),
                        ),
                        Container(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 8,
                            vertical: 4,
                          ),
                          decoration: BoxDecoration(
                            color: palette.success.withValues(alpha: 0.12),
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Text(
                            'VERIFIED',
                            style: TextStyle(
                              color: palette.success,
                              fontSize: 10,
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                        ),
                      ],
                    ),
                  ),
                  if (fingerprint.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    _ShareDetail(
                      label: 'DEVICE IDENTITY',
                      child: SelectionArea(
                        child: Text(
                          fingerprint,
                          style: TextStyle(
                            color: palette.muted,
                            fontFamily: 'JetBrainsMono',
                            fontSize: 11,
                            height: 1.5,
                          ),
                        ),
                      ),
                    ),
                  ],
                  const SizedBox(height: 20),
                  Text(
                    'ROUTE CHANGES',
                    style: TextStyle(
                      color: palette.muted,
                      fontSize: 11,
                      fontWeight: FontWeight.w700,
                    ),
                  ),
                  const SizedBox(height: 8),
                  if (preview.routeDiffs.isEmpty)
                    _ShareDetail(
                      label: 'NO ROUTE CHANGES',
                      child: Text(
                        'Existing routes stay unchanged.',
                        style: TextStyle(color: palette.muted, fontSize: 12),
                      ),
                    )
                  else
                    Container(
                      clipBehavior: Clip.antiAlias,
                      decoration: BoxDecoration(
                        color: palette.surface,
                        border: Border.all(color: palette.border),
                        borderRadius: BorderRadius.circular(8),
                      ),
                      child: Column(
                        children: [
                          for (
                            var index = 0;
                            index < preview.routeDiffs.length;
                            index++
                          ) ...[
                            _ShareRouteDiffRow(diff: preview.routeDiffs[index]),
                            if (index != preview.routeDiffs.length - 1)
                              Divider(height: 1, color: palette.border),
                          ],
                        ],
                      ),
                    ),
                  const SizedBox(height: 12),
                  _ShareDetail(
                    label: policyChanges.isEmpty
                        ? 'POLICY UNCHANGED'
                        : 'POLICY CHANGES',
                    child: Text(
                      policyChanges.isEmpty
                          ? 'Connection policy stays unchanged.'
                          : '${policyChanges.join(' and ')} will be updated.',
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                  ),
                  const SizedBox(height: 12),
                  _ShareDetail(
                    label: 'CREDENTIALS STAY LOCAL',
                    icon: Icons.key_off_outlined,
                    child: Text(
                      preview.credentialDescriptors.isEmpty
                          ? 'No local credential setup is required by this import.'
                          : '${preview.credentialDescriptors.length} route credential${preview.credentialDescriptors.length == 1 ? '' : 's'} must be prepared on this device after import.',
                      style: TextStyle(color: palette.muted, fontSize: 12),
                    ),
                  ),
                  if (error != null) ...[
                    const SizedBox(height: 12),
                    Semantics(
                      liveRegion: true,
                      child: Text(
                        error!,
                        style: TextStyle(color: palette.danger, fontSize: 12),
                      ),
                    ),
                  ],
                  if (submitting) ...[
                    const SizedBox(height: 12),
                    const LinearProgressIndicator(minHeight: 2),
                  ],
                  const SizedBox(height: 20),
                  SizedBox(
                    height: 48,
                    child: FilledButton.icon(
                      onPressed: submitting ? null : onCommit,
                      icon: const Icon(Icons.download_rounded, size: 18),
                      label: const Text('Import configuration'),
                    ),
                  ),
                  const SizedBox(height: 8),
                  TextButton(
                    onPressed: submitting ? null : onCancel,
                    child: const Text('Scan a different code'),
                  ),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }
}

final class _ShareDetail extends StatelessWidget {
  const _ShareDetail({required this.label, required this.child, this.icon});

  final String label;
  final Widget child;
  final IconData? icon;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: palette.surfaceRaised,
        border: Border.all(color: palette.border),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (icon != null) ...[
            Icon(icon, color: palette.muted, size: 18),
            const SizedBox(width: 10),
          ],
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  label,
                  style: TextStyle(
                    color: palette.muted,
                    fontSize: 10,
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 5),
                child,
              ],
            ),
          ),
        ],
      ),
    );
  }
}

final class _ShareRouteDiffRow extends StatelessWidget {
  const _ShareRouteDiffRow({required this.diff});

  final EndpointShareRouteDiff diff;

  @override
  Widget build(BuildContext context) {
    final palette = AnyttyPalette.of(context);
    final action = diff.action.trim().isEmpty ? 'update' : diff.action.trim();
    final actionColor = switch (action.toLowerCase()) {
      'add' || 'added' || 'create' => palette.success,
      'remove' || 'removed' || 'delete' => palette.danger,
      _ => palette.accent,
    };
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      child: Row(
        children: [
          Icon(Icons.alt_route_rounded, color: palette.muted, size: 18),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  diff.routeId,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(
                    fontSize: 13,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                Text(
                  diff.routeKind.trim().isEmpty
                      ? 'Route'
                      : diff.routeKind.trim(),
                  style: TextStyle(color: palette.muted, fontSize: 11),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          Text(
            action.toUpperCase(),
            style: TextStyle(
              color: actionColor,
              fontSize: 10,
              fontWeight: FontWeight.w700,
            ),
          ),
        ],
      ),
    );
  }
}
