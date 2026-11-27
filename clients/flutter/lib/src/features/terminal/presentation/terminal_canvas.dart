import 'dart:async';
import 'dart:math' as math;
import 'dart:ui' as ui;

import 'package:flutter/material.dart';

import '../../../generated/proto/apipb/history.pb.dart';
import '../domain/history_interaction.dart';
import '../domain/live_screen_store.dart';
import '../domain/terminal_links.dart';
import '../domain/terminal_metrics.dart';
import '../domain/terminal_perf_counters.dart';
import '../domain/terminal_settings.dart';

part 'terminal_canvas_live.dart';
part 'terminal_canvas_history.dart';
part 'terminal_canvas_painters.dart';
