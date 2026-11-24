import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'src/app/anytty_app.dart';
import 'src/features/terminal/domain/terminal_perf_counters.dart';

void main() {
  WidgetsFlutterBinding.ensureInitialized();
  TerminalPerfCounters.instance.start();
  runApp(const ProviderScope(child: AnyttyApp()));
}
