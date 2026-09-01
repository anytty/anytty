import '../../../generated/proto/apipb/terminal.pb.dart';

ResizeControl projectResizeControl({
  required ResizeControl incoming,
  required String surfaceId,
  required String viewId,
}) {
  final control = incoming.deepCopy();
  final ownerSurface = control.ownerSurfaceId.isNotEmpty
      ? control.ownerSurfaceId
      : control.hasOwnership()
      ? control.ownership.ownerSurfaceId
      : '';
  final ownerView = control.ownerViewId.isNotEmpty
      ? control.ownerViewId
      : control.hasOwnership()
      ? control.ownership.ownerViewId
      : '';
  final ownsResize =
      (ownerSurface == surfaceId &&
          (ownerView.isEmpty || ownerView == viewId)) ||
      (ownerSurface.isEmpty && control.canResize);
  final sizeLocked =
      control.sizeLocked ||
      (control.hasOwnership() && control.ownership.sizeLocked);
  control
    ..surfaceId = surfaceId
    ..ownerSurfaceId = ownerSurface
    ..ownerViewId = ownerView
    ..sizeLocked = sizeLocked
    ..canResize = ownsResize && !sizeLocked
    ..reason = ownsResize
        ? sizeLocked
              ? ResizeControlReason.RESIZE_CONTROL_REASON_SIZE_LOCKED
              : ResizeControlReason.RESIZE_CONTROL_REASON_OWNER
        : ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER;
  return control;
}
