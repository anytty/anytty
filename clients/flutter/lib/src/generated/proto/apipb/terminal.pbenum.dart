// This is a generated file - do not edit.
//
// Generated from apipb/terminal.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;

class TerminalState extends $pb.ProtobufEnum {
  static const TerminalState TERMINAL_STATE_UNSPECIFIED =
      TerminalState._(0, _omitEnumNames ? '' : 'TERMINAL_STATE_UNSPECIFIED');
  static const TerminalState TERMINAL_STATE_CREATED =
      TerminalState._(1, _omitEnumNames ? '' : 'TERMINAL_STATE_CREATED');
  static const TerminalState TERMINAL_STATE_RUNNING =
      TerminalState._(2, _omitEnumNames ? '' : 'TERMINAL_STATE_RUNNING');
  static const TerminalState TERMINAL_STATE_EXITED =
      TerminalState._(3, _omitEnumNames ? '' : 'TERMINAL_STATE_EXITED');
  static const TerminalState TERMINAL_STATE_REMOVED =
      TerminalState._(4, _omitEnumNames ? '' : 'TERMINAL_STATE_REMOVED');

  static const $core.List<TerminalState> values = <TerminalState>[
    TERMINAL_STATE_UNSPECIFIED,
    TERMINAL_STATE_CREATED,
    TERMINAL_STATE_RUNNING,
    TERMINAL_STATE_EXITED,
    TERMINAL_STATE_REMOVED,
  ];

  static final $core.List<TerminalState?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static TerminalState? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const TerminalState._(super.value, super.name);
}

class AttachmentMode extends $pb.ProtobufEnum {
  static const AttachmentMode ATTACHMENT_MODE_UNSPECIFIED =
      AttachmentMode._(0, _omitEnumNames ? '' : 'ATTACHMENT_MODE_UNSPECIFIED');
  static const AttachmentMode ATTACHMENT_MODE_COLLABORATOR =
      AttachmentMode._(1, _omitEnumNames ? '' : 'ATTACHMENT_MODE_COLLABORATOR');
  static const AttachmentMode ATTACHMENT_MODE_OBSERVER =
      AttachmentMode._(2, _omitEnumNames ? '' : 'ATTACHMENT_MODE_OBSERVER');

  static const $core.List<AttachmentMode> values = <AttachmentMode>[
    ATTACHMENT_MODE_UNSPECIFIED,
    ATTACHMENT_MODE_COLLABORATOR,
    ATTACHMENT_MODE_OBSERVER,
  ];

  static final $core.List<AttachmentMode?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 2);
  static AttachmentMode? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const AttachmentMode._(super.value, super.name);
}

class ResizePolicy extends $pb.ProtobufEnum {
  static const ResizePolicy RESIZE_POLICY_UNSPECIFIED =
      ResizePolicy._(0, _omitEnumNames ? '' : 'RESIZE_POLICY_UNSPECIFIED');
  static const ResizePolicy RESIZE_POLICY_OWNER =
      ResizePolicy._(1, _omitEnumNames ? '' : 'RESIZE_POLICY_OWNER');
  static const ResizePolicy RESIZE_POLICY_FOLLOWER =
      ResizePolicy._(2, _omitEnumNames ? '' : 'RESIZE_POLICY_FOLLOWER');
  static const ResizePolicy RESIZE_POLICY_OBSERVER =
      ResizePolicy._(3, _omitEnumNames ? '' : 'RESIZE_POLICY_OBSERVER');

  static const $core.List<ResizePolicy> values = <ResizePolicy>[
    RESIZE_POLICY_UNSPECIFIED,
    RESIZE_POLICY_OWNER,
    RESIZE_POLICY_FOLLOWER,
    RESIZE_POLICY_OBSERVER,
  ];

  static final $core.List<ResizePolicy?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 3);
  static ResizePolicy? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ResizePolicy._(super.value, super.name);
}

class ResizeControlReason extends $pb.ProtobufEnum {
  static const ResizeControlReason RESIZE_CONTROL_REASON_UNSPECIFIED =
      ResizeControlReason._(
          0, _omitEnumNames ? '' : 'RESIZE_CONTROL_REASON_UNSPECIFIED');
  static const ResizeControlReason RESIZE_CONTROL_REASON_OWNER =
      ResizeControlReason._(
          1, _omitEnumNames ? '' : 'RESIZE_CONTROL_REASON_OWNER');
  static const ResizeControlReason RESIZE_CONTROL_REASON_FOLLOWER =
      ResizeControlReason._(
          2, _omitEnumNames ? '' : 'RESIZE_CONTROL_REASON_FOLLOWER');
  static const ResizeControlReason RESIZE_CONTROL_REASON_OBSERVER =
      ResizeControlReason._(
          3, _omitEnumNames ? '' : 'RESIZE_CONTROL_REASON_OBSERVER');
  static const ResizeControlReason RESIZE_CONTROL_REASON_SIZE_LOCKED =
      ResizeControlReason._(
          4, _omitEnumNames ? '' : 'RESIZE_CONTROL_REASON_SIZE_LOCKED');

  static const $core.List<ResizeControlReason> values = <ResizeControlReason>[
    RESIZE_CONTROL_REASON_UNSPECIFIED,
    RESIZE_CONTROL_REASON_OWNER,
    RESIZE_CONTROL_REASON_FOLLOWER,
    RESIZE_CONTROL_REASON_OBSERVER,
    RESIZE_CONTROL_REASON_SIZE_LOCKED,
  ];

  static final $core.List<ResizeControlReason?> _byValue =
      $pb.ProtobufEnum.$_initByValueList(values, 4);
  static ResizeControlReason? valueOf($core.int value) =>
      value < 0 || value >= _byValue.length ? null : _byValue[value];

  const ResizeControlReason._(super.value, super.name);
}

const $core.bool _omitEnumNames =
    $core.bool.fromEnvironment('protobuf.omit_enum_names');
