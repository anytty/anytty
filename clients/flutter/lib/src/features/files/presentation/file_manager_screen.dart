import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/providers.dart';
import '../../../shared/presentation/anytty_brand_mark.dart';
import '../../terminal/data/endpoint_session_client.dart';
import '../../../generated/proto/apipb/file.pb.dart';
import '../data/file_manager_path_store.dart';
import '../domain/file_preview_safety.dart';
import '../domain/file_web_preview.dart';
import '../domain/file_path.dart';
import 'file_transfer_controller.dart';
import 'file_transfer_sheet.dart';
import 'path_bookmarks_sheet.dart';
import 'file_web_preview_screen.dart';

part 'file_manager_root.dart';
part 'file_manager_actions.dart';
part 'file_manager_preview.dart';
