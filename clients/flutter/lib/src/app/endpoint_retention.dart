import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../native/anytty_runtime.dart';
import 'providers.dart';

/// App 级“连接保留”控制器。
///
/// 引擎（EndpointSupervisor）按 demand 租约决定是否维持某台机器的连接：
/// 没有任何租约时，连接在最后一个界面退出后约 2 秒被回收。
/// 界面层（endpointDemandProvider）的租约随页面生命周期释放，所以
/// 在 session 内切换到另一台机器时，上一台机器会被默认断开。
///
/// 本控制器为用户“进入过”的每台机器额外持有一个租约，直到用户显式
/// 断开/移除该设备，切换机器不再默认断开连接。
final class EndpointRetentionController {
  EndpointRetentionController(this._retainer);

  final EndpointDemandRetainer _retainer;
  final Map<String, EndpointDemandLease> _leases = {};
  bool _closed = false;

  /// 当前被保留连接的 endpointId（排序后的只读副本）。
  List<String> get retainedEndpointIds =>
      List<String>.unmodifiable(_leases.keys.toList()..sort());

  bool isRetained(String endpointId) => _leases.containsKey(_normalize(endpointId));

  /// 为 endpointId 追加一个保留租约；重复调用是幂等的。
  void retain(String endpointId) {
    final id = _normalize(endpointId);
    if (id == null || _closed) return;
    _leases.putIfAbsent(
      id,
      () => _retainer.retainEndpointDemand(id),
    );
  }

  /// 释放保留租约；不持有的 endpointId 是空操作。
  void release(String endpointId) {
    final id = _normalize(endpointId);
    if (id == null) return;
    _leases.remove(id)?.release();
  }

  /// 释放全部保留租约（进程退出时使用）。
  void releaseAll() {
    if (_closed) return;
    _closed = true;
    final leases = List<EndpointDemandLease>.of(_leases.values);
    _leases.clear();
    for (final lease in leases) {
      lease.release();
    }
  }

  static String? _normalize(String endpointId) {
    final trimmed = endpointId.trim();
    return trimmed.isEmpty ? null : trimmed;
  }
}

final endpointRetentionProvider =
    FutureProvider<EndpointRetentionController>((ref) async {
      final runtime = await ref.watch(anyttyRuntimeProvider.future);
      final controller = EndpointRetentionController(runtime);
      ref.onDispose(controller.releaseAll);
      return controller;
    });
/// 进入某台机器（终端工作区 / 浏览器会话）时调用。
///
/// 保留是尽力而为：运行时不可用时静默失败，界面层的租约仍然生效。
Future<void> retainEndpointConnection(WidgetRef ref, String endpointId) async {
  final trimmed = endpointId.trim();
  if (trimmed.isEmpty) return;
  try {
    final controller = await ref.read(endpointRetentionProvider.future);
    controller.retain(trimmed);
  } catch (_) {
    // 运行时尚未就绪或已关闭；忽略，不阻塞界面。
  }
}

/// 用户显式断开/移除某台机器时调用，避免保留租约触发自动重连。
Future<void> releaseEndpointConnection(WidgetRef ref, String endpointId) async {
  final trimmed = endpointId.trim();
  if (trimmed.isEmpty) return;
  try {
    final controller = await ref.read(endpointRetentionProvider.future);
    controller.release(trimmed);
  } catch (_) {
    // 运行时尚未就绪或已关闭；忽略。
  }
}
