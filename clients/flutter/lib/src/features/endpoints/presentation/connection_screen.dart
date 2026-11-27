import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';

import '../../../app/anytty_theme.dart';
import '../../../app/anytty_localizations.dart';
import '../../../app/providers.dart';
import '../../../generated/proto/bindingpb/client_binding.pb.dart';
import '../../../generated/proto/remoteauthpb/remote_auth.pb.dart';
import '../data/connection_repository.dart';
import '../data/endpoint_repository.dart';

part 'connection_screen_root.dart';
part 'connection_screen_routes.dart';
part 'connection_screen_policy.dart';
part 'connection_screen_footer.dart';
