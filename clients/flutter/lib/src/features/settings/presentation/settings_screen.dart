import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/app_appearance.dart';
import '../../../app/app_color_preferences.dart';
import '../../../app/app_language.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/background_preferences.dart';
import '../../../app/providers.dart';
import '../../../native/background_platform.dart';
import '../../../shared/presentation/anytty_brand_mark.dart';
import '../../terminal/domain/terminal_petal_menu_preferences.dart';
import '../../terminal/domain/terminal_settings.dart';

part 'settings_screen_root.dart';
part 'settings_screen_controls.dart';
part 'settings_screen_pickers.dart';
