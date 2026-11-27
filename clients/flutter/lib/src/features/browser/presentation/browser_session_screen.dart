import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:lucide_icons_flutter/lucide_icons.dart';
import 'package:webview_flutter/webview_flutter.dart';

import '../../../app/anytty_localizations.dart';
import '../../../app/anytty_theme.dart';
import '../../../app/endpoint_retention.dart';
import '../../../app/providers.dart';
import '../../../native/browser_proxy_platform.dart';
import '../../../shared/presentation/anytty_brand_mark.dart';
import '../data/browser_bookmark_store.dart';
import '../data/browser_http_proxy.dart';
import '../data/browser_history_store.dart';
import '../data/browser_session_store.dart';
import '../domain/browser_load_progress.dart';
import '../domain/browser_navigation.dart';
import '../domain/browser_session.dart';
import '../domain/browser_session_recovery.dart';
import '../../terminal/data/endpoint_session_client.dart';
import 'browser_endpoint_picker_sheet.dart';
import 'browser_new_tab_page.dart';
import 'browser_address_bar.dart';

part 'browser_session_root.dart';
part 'browser_session_toolbar.dart';
part 'browser_session_overlays.dart';
