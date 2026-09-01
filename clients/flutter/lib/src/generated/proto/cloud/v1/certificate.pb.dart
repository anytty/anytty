// This is a generated file - do not edit.
//
// Generated from cloud/v1/certificate.proto.

// @dart = 3.3

// ignore_for_file: annotate_overrides, camel_case_types, comment_references
// ignore_for_file: constant_identifier_names
// ignore_for_file: curly_braces_in_flow_control_structures
// ignore_for_file: deprecated_member_use_from_same_package, library_prefixes
// ignore_for_file: non_constant_identifier_names, prefer_relative_imports

import 'dart:core' as $core;

import 'package:protobuf/protobuf.dart' as $pb;
import 'package:protobuf/well_known_types/google/protobuf/timestamp.pb.dart'
    as $0;

export 'package:protobuf/protobuf.dart' show GeneratedMessageGenericExtensions;

/// EdgePublicCertificateStatus is the current automatically managed TLS
/// credential presented by an Edge public endpoint. Private key material never
/// leaves the Edge and is never part of this projection.
class EdgePublicCertificateStatus extends $pb.GeneratedMessage {
  factory EdgePublicCertificateStatus({
    $core.String? publicEndpoint,
    $core.List<$core.int>? certificateSha256,
    $0.Timestamp? notBefore,
    $0.Timestamp? notAfter,
    $core.bool? renewalPending,
    $core.String? lastErrorCode,
    $core.String? lastErrorMessage,
    $0.Timestamp? lastAttemptAt,
    $0.Timestamp? appliedAt,
  }) {
    final result = create();
    if (publicEndpoint != null) result.publicEndpoint = publicEndpoint;
    if (certificateSha256 != null) result.certificateSha256 = certificateSha256;
    if (notBefore != null) result.notBefore = notBefore;
    if (notAfter != null) result.notAfter = notAfter;
    if (renewalPending != null) result.renewalPending = renewalPending;
    if (lastErrorCode != null) result.lastErrorCode = lastErrorCode;
    if (lastErrorMessage != null) result.lastErrorMessage = lastErrorMessage;
    if (lastAttemptAt != null) result.lastAttemptAt = lastAttemptAt;
    if (appliedAt != null) result.appliedAt = appliedAt;
    return result;
  }

  EdgePublicCertificateStatus._();

  factory EdgePublicCertificateStatus.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgePublicCertificateStatus.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgePublicCertificateStatus',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'publicEndpoint')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'certificateSha256', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(3, _omitFieldNames ? '' : 'notBefore',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(4, _omitFieldNames ? '' : 'notAfter',
        subBuilder: $0.Timestamp.create)
    ..aOB(5, _omitFieldNames ? '' : 'renewalPending')
    ..aOS(6, _omitFieldNames ? '' : 'lastErrorCode')
    ..aOS(7, _omitFieldNames ? '' : 'lastErrorMessage')
    ..aOM<$0.Timestamp>(8, _omitFieldNames ? '' : 'lastAttemptAt',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(9, _omitFieldNames ? '' : 'appliedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateStatus clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateStatus copyWith(
          void Function(EdgePublicCertificateStatus) updates) =>
      super.copyWith(
              (message) => updates(message as EdgePublicCertificateStatus))
          as EdgePublicCertificateStatus;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateStatus create() =>
      EdgePublicCertificateStatus._();
  @$core.override
  EdgePublicCertificateStatus createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateStatus getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgePublicCertificateStatus>(create);
  static EdgePublicCertificateStatus? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get publicEndpoint => $_getSZ(0);
  @$pb.TagNumber(1)
  set publicEndpoint($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasPublicEndpoint() => $_has(0);
  @$pb.TagNumber(1)
  void clearPublicEndpoint() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get certificateSha256 => $_getN(1);
  @$pb.TagNumber(2)
  set certificateSha256($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCertificateSha256() => $_has(1);
  @$pb.TagNumber(2)
  void clearCertificateSha256() => $_clearField(2);

  @$pb.TagNumber(3)
  $0.Timestamp get notBefore => $_getN(2);
  @$pb.TagNumber(3)
  set notBefore($0.Timestamp value) => $_setField(3, value);
  @$pb.TagNumber(3)
  $core.bool hasNotBefore() => $_has(2);
  @$pb.TagNumber(3)
  void clearNotBefore() => $_clearField(3);
  @$pb.TagNumber(3)
  $0.Timestamp ensureNotBefore() => $_ensure(2);

  @$pb.TagNumber(4)
  $0.Timestamp get notAfter => $_getN(3);
  @$pb.TagNumber(4)
  set notAfter($0.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasNotAfter() => $_has(3);
  @$pb.TagNumber(4)
  void clearNotAfter() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Timestamp ensureNotAfter() => $_ensure(3);

  @$pb.TagNumber(5)
  $core.bool get renewalPending => $_getBF(4);
  @$pb.TagNumber(5)
  set renewalPending($core.bool value) => $_setBool(4, value);
  @$pb.TagNumber(5)
  $core.bool hasRenewalPending() => $_has(4);
  @$pb.TagNumber(5)
  void clearRenewalPending() => $_clearField(5);

  @$pb.TagNumber(6)
  $core.String get lastErrorCode => $_getSZ(5);
  @$pb.TagNumber(6)
  set lastErrorCode($core.String value) => $_setString(5, value);
  @$pb.TagNumber(6)
  $core.bool hasLastErrorCode() => $_has(5);
  @$pb.TagNumber(6)
  void clearLastErrorCode() => $_clearField(6);

  @$pb.TagNumber(7)
  $core.String get lastErrorMessage => $_getSZ(6);
  @$pb.TagNumber(7)
  set lastErrorMessage($core.String value) => $_setString(6, value);
  @$pb.TagNumber(7)
  $core.bool hasLastErrorMessage() => $_has(6);
  @$pb.TagNumber(7)
  void clearLastErrorMessage() => $_clearField(7);

  @$pb.TagNumber(8)
  $0.Timestamp get lastAttemptAt => $_getN(7);
  @$pb.TagNumber(8)
  set lastAttemptAt($0.Timestamp value) => $_setField(8, value);
  @$pb.TagNumber(8)
  $core.bool hasLastAttemptAt() => $_has(7);
  @$pb.TagNumber(8)
  void clearLastAttemptAt() => $_clearField(8);
  @$pb.TagNumber(8)
  $0.Timestamp ensureLastAttemptAt() => $_ensure(7);

  @$pb.TagNumber(9)
  $0.Timestamp get appliedAt => $_getN(8);
  @$pb.TagNumber(9)
  set appliedAt($0.Timestamp value) => $_setField(9, value);
  @$pb.TagNumber(9)
  $core.bool hasAppliedAt() => $_has(8);
  @$pb.TagNumber(9)
  void clearAppliedAt() => $_clearField(9);
  @$pb.TagNumber(9)
  $0.Timestamp ensureAppliedAt() => $_ensure(8);
}

/// EdgePublicCertificateRenewRequest is sent only inside an authenticated
/// EdgeControl stream. The new private key is generated and retained by Edge.
class EdgePublicCertificateRenewRequest extends $pb.GeneratedMessage {
  factory EdgePublicCertificateRenewRequest({
    $core.String? requestId,
    $core.List<$core.int>? csrPem,
    $core.List<$core.int>? currentCertificateSha256,
    $0.Timestamp? requestedAt,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (csrPem != null) result.csrPem = csrPem;
    if (currentCertificateSha256 != null)
      result.currentCertificateSha256 = currentCertificateSha256;
    if (requestedAt != null) result.requestedAt = requestedAt;
    return result;
  }

  EdgePublicCertificateRenewRequest._();

  factory EdgePublicCertificateRenewRequest.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgePublicCertificateRenewRequest.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgePublicCertificateRenewRequest',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'csrPem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(3,
        _omitFieldNames ? '' : 'currentCertificateSha256', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(4, _omitFieldNames ? '' : 'requestedAt',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateRenewRequest clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateRenewRequest copyWith(
          void Function(EdgePublicCertificateRenewRequest) updates) =>
      super.copyWith((message) =>
              updates(message as EdgePublicCertificateRenewRequest))
          as EdgePublicCertificateRenewRequest;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateRenewRequest create() =>
      EdgePublicCertificateRenewRequest._();
  @$core.override
  EdgePublicCertificateRenewRequest createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateRenewRequest getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgePublicCertificateRenewRequest>(
          create);
  static EdgePublicCertificateRenewRequest? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get csrPem => $_getN(1);
  @$pb.TagNumber(2)
  set csrPem($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCsrPem() => $_has(1);
  @$pb.TagNumber(2)
  void clearCsrPem() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get currentCertificateSha256 => $_getN(2);
  @$pb.TagNumber(3)
  set currentCertificateSha256($core.List<$core.int> value) =>
      $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCurrentCertificateSha256() => $_has(2);
  @$pb.TagNumber(3)
  void clearCurrentCertificateSha256() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.Timestamp get requestedAt => $_getN(3);
  @$pb.TagNumber(4)
  set requestedAt($0.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasRequestedAt() => $_has(3);
  @$pb.TagNumber(4)
  void clearRequestedAt() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Timestamp ensureRequestedAt() => $_ensure(3);
}

/// EdgePublicCertificateRenewResponse contains a serverAuth chain signed for
/// the Controller-owned public_endpoint of the authenticated Edge.
class EdgePublicCertificateRenewResponse extends $pb.GeneratedMessage {
  factory EdgePublicCertificateRenewResponse({
    $core.String? requestId,
    $core.List<$core.int>? certificatePem,
    $core.List<$core.int>? certificateSha256,
    $0.Timestamp? notBefore,
    $0.Timestamp? notAfter,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (certificatePem != null) result.certificatePem = certificatePem;
    if (certificateSha256 != null) result.certificateSha256 = certificateSha256;
    if (notBefore != null) result.notBefore = notBefore;
    if (notAfter != null) result.notAfter = notAfter;
    return result;
  }

  EdgePublicCertificateRenewResponse._();

  factory EdgePublicCertificateRenewResponse.fromBuffer(
          $core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgePublicCertificateRenewResponse.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgePublicCertificateRenewResponse',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..a<$core.List<$core.int>>(
        2, _omitFieldNames ? '' : 'certificatePem', $pb.PbFieldType.OY)
    ..a<$core.List<$core.int>>(
        3, _omitFieldNames ? '' : 'certificateSha256', $pb.PbFieldType.OY)
    ..aOM<$0.Timestamp>(4, _omitFieldNames ? '' : 'notBefore',
        subBuilder: $0.Timestamp.create)
    ..aOM<$0.Timestamp>(5, _omitFieldNames ? '' : 'notAfter',
        subBuilder: $0.Timestamp.create)
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateRenewResponse clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateRenewResponse copyWith(
          void Function(EdgePublicCertificateRenewResponse) updates) =>
      super.copyWith((message) =>
              updates(message as EdgePublicCertificateRenewResponse))
          as EdgePublicCertificateRenewResponse;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateRenewResponse create() =>
      EdgePublicCertificateRenewResponse._();
  @$core.override
  EdgePublicCertificateRenewResponse createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateRenewResponse getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgePublicCertificateRenewResponse>(
          create);
  static EdgePublicCertificateRenewResponse? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  $core.List<$core.int> get certificatePem => $_getN(1);
  @$pb.TagNumber(2)
  set certificatePem($core.List<$core.int> value) => $_setBytes(1, value);
  @$pb.TagNumber(2)
  $core.bool hasCertificatePem() => $_has(1);
  @$pb.TagNumber(2)
  void clearCertificatePem() => $_clearField(2);

  @$pb.TagNumber(3)
  $core.List<$core.int> get certificateSha256 => $_getN(2);
  @$pb.TagNumber(3)
  set certificateSha256($core.List<$core.int> value) => $_setBytes(2, value);
  @$pb.TagNumber(3)
  $core.bool hasCertificateSha256() => $_has(2);
  @$pb.TagNumber(3)
  void clearCertificateSha256() => $_clearField(3);

  @$pb.TagNumber(4)
  $0.Timestamp get notBefore => $_getN(3);
  @$pb.TagNumber(4)
  set notBefore($0.Timestamp value) => $_setField(4, value);
  @$pb.TagNumber(4)
  $core.bool hasNotBefore() => $_has(3);
  @$pb.TagNumber(4)
  void clearNotBefore() => $_clearField(4);
  @$pb.TagNumber(4)
  $0.Timestamp ensureNotBefore() => $_ensure(3);

  @$pb.TagNumber(5)
  $0.Timestamp get notAfter => $_getN(4);
  @$pb.TagNumber(5)
  set notAfter($0.Timestamp value) => $_setField(5, value);
  @$pb.TagNumber(5)
  $core.bool hasNotAfter() => $_has(4);
  @$pb.TagNumber(5)
  void clearNotAfter() => $_clearField(5);
  @$pb.TagNumber(5)
  $0.Timestamp ensureNotAfter() => $_ensure(4);
}

/// EdgePublicCertificateApplied reports atomic persistence and TLS hot reload.
/// A failed application leaves the previously loaded credential untouched.
class EdgePublicCertificateApplied extends $pb.GeneratedMessage {
  factory EdgePublicCertificateApplied({
    $core.String? requestId,
    EdgePublicCertificateStatus? status,
    $core.bool? applied,
    $core.String? errorCode,
    $core.String? errorMessage,
  }) {
    final result = create();
    if (requestId != null) result.requestId = requestId;
    if (status != null) result.status = status;
    if (applied != null) result.applied = applied;
    if (errorCode != null) result.errorCode = errorCode;
    if (errorMessage != null) result.errorMessage = errorMessage;
    return result;
  }

  EdgePublicCertificateApplied._();

  factory EdgePublicCertificateApplied.fromBuffer($core.List<$core.int> data,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromBuffer(data, registry);
  factory EdgePublicCertificateApplied.fromJson($core.String json,
          [$pb.ExtensionRegistry registry = $pb.ExtensionRegistry.EMPTY]) =>
      create()..mergeFromJson(json, registry);

  static final $pb.BuilderInfo _i = $pb.BuilderInfo(
      _omitMessageNames ? '' : 'EdgePublicCertificateApplied',
      package:
          const $pb.PackageName(_omitMessageNames ? '' : 'anytty.cloud.v1'),
      createEmptyInstance: create)
    ..aOS(1, _omitFieldNames ? '' : 'requestId')
    ..aOM<EdgePublicCertificateStatus>(2, _omitFieldNames ? '' : 'status',
        subBuilder: EdgePublicCertificateStatus.create)
    ..aOB(3, _omitFieldNames ? '' : 'applied')
    ..aOS(4, _omitFieldNames ? '' : 'errorCode')
    ..aOS(5, _omitFieldNames ? '' : 'errorMessage')
    ..hasRequiredFields = false;

  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateApplied clone() => deepCopy();
  @$core.Deprecated('See https://github.com/google/protobuf.dart/issues/998.')
  EdgePublicCertificateApplied copyWith(
          void Function(EdgePublicCertificateApplied) updates) =>
      super.copyWith(
              (message) => updates(message as EdgePublicCertificateApplied))
          as EdgePublicCertificateApplied;

  @$core.override
  $pb.BuilderInfo get info_ => _i;

  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateApplied create() =>
      EdgePublicCertificateApplied._();
  @$core.override
  EdgePublicCertificateApplied createEmptyInstance() => create();
  @$core.pragma('dart2js:noInline')
  static EdgePublicCertificateApplied getDefault() => _defaultInstance ??=
      $pb.GeneratedMessage.$_defaultFor<EdgePublicCertificateApplied>(create);
  static EdgePublicCertificateApplied? _defaultInstance;

  @$pb.TagNumber(1)
  $core.String get requestId => $_getSZ(0);
  @$pb.TagNumber(1)
  set requestId($core.String value) => $_setString(0, value);
  @$pb.TagNumber(1)
  $core.bool hasRequestId() => $_has(0);
  @$pb.TagNumber(1)
  void clearRequestId() => $_clearField(1);

  @$pb.TagNumber(2)
  EdgePublicCertificateStatus get status => $_getN(1);
  @$pb.TagNumber(2)
  set status(EdgePublicCertificateStatus value) => $_setField(2, value);
  @$pb.TagNumber(2)
  $core.bool hasStatus() => $_has(1);
  @$pb.TagNumber(2)
  void clearStatus() => $_clearField(2);
  @$pb.TagNumber(2)
  EdgePublicCertificateStatus ensureStatus() => $_ensure(1);

  @$pb.TagNumber(3)
  $core.bool get applied => $_getBF(2);
  @$pb.TagNumber(3)
  set applied($core.bool value) => $_setBool(2, value);
  @$pb.TagNumber(3)
  $core.bool hasApplied() => $_has(2);
  @$pb.TagNumber(3)
  void clearApplied() => $_clearField(3);

  @$pb.TagNumber(4)
  $core.String get errorCode => $_getSZ(3);
  @$pb.TagNumber(4)
  set errorCode($core.String value) => $_setString(3, value);
  @$pb.TagNumber(4)
  $core.bool hasErrorCode() => $_has(3);
  @$pb.TagNumber(4)
  void clearErrorCode() => $_clearField(4);

  @$pb.TagNumber(5)
  $core.String get errorMessage => $_getSZ(4);
  @$pb.TagNumber(5)
  set errorMessage($core.String value) => $_setString(4, value);
  @$pb.TagNumber(5)
  $core.bool hasErrorMessage() => $_has(4);
  @$pb.TagNumber(5)
  void clearErrorMessage() => $_clearField(5);
}

const $core.bool _omitFieldNames =
    $core.bool.fromEnvironment('protobuf.omit_field_names');
const $core.bool _omitMessageNames =
    $core.bool.fromEnvironment('protobuf.omit_message_names');
