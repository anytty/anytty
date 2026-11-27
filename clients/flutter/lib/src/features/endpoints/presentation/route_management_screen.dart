import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../../terminal/data/endpoint_session_client.dart';
import '../data/endpoint_repository.dart';
import '../domain/route_management.dart';

part 'route_management_root.dart';
part 'route_management_editor.dart';
part 'route_management_rows.dart';
