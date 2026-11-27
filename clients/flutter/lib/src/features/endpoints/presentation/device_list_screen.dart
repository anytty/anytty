import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:mobile_scanner/mobile_scanner.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/endpoint_retention.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../../shared/presentation/fuzzy_highlight_text.dart';
import '../../../shared/presentation/anytty_brand_mark.dart';
import '../../browser/data/browser_device_data.dart';
import '../../files/data/file_manager_path_store.dart';
import '../../files/presentation/file_transfer_sheet.dart';
import '../data/endpoint_repository.dart';
import '../domain/device_search.dart';

part 'device_list_root.dart';
part 'device_list_header.dart';
part 'device_list_row.dart';
part 'device_list_sheets.dart';
part 'device_list_pairing.dart';
