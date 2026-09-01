import 'package:anytty_native/src/features/terminal/domain/resize_control.dart';
import 'package:anytty_native/src/generated/proto/apipb/terminal.pb.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('projects the local owner as resize-capable', () {
    final projected = projectResizeControl(
      incoming: _control(ownerSurface: 'surface-a', ownerView: 'view-a'),
      surfaceId: 'surface-a',
      viewId: 'view-a',
    );

    expect(projected.canResize, isTrue);
    expect(projected.reason, ResizeControlReason.RESIZE_CONTROL_REASON_OWNER);
  });

  test('projects another surface as a follower', () {
    final projected = projectResizeControl(
      incoming: _control(ownerSurface: 'surface-b', ownerView: 'view-b'),
      surfaceId: 'surface-a',
      viewId: 'view-a',
    );

    expect(projected.canResize, isFalse);
    expect(
      projected.reason,
      ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER,
    );
    expect(projected.ownerSurfaceId, 'surface-b');
  });

  test('keeps size lock separate from ownership', () {
    final owner = projectResizeControl(
      incoming: _control(
        ownerSurface: 'surface-a',
        ownerView: 'view-a',
        locked: true,
      ),
      surfaceId: 'surface-a',
      viewId: 'view-a',
    );
    final follower = projectResizeControl(
      incoming: _control(
        ownerSurface: 'surface-a',
        ownerView: 'view-a',
        locked: true,
      ),
      surfaceId: 'surface-b',
      viewId: 'view-b',
    );

    expect(owner.canResize, isFalse);
    expect(owner.sizeLocked, isTrue);
    expect(owner.reason, ResizeControlReason.RESIZE_CONTROL_REASON_SIZE_LOCKED);
    expect(follower.canResize, isFalse);
    expect(follower.sizeLocked, isTrue);
    expect(follower.reason, ResizeControlReason.RESIZE_CONTROL_REASON_FOLLOWER);
  });
}

ResizeControl _control({
  required String ownerSurface,
  required String ownerView,
  bool locked = false,
}) {
  return ResizeControl(
    ownerSurfaceId: ownerSurface,
    ownerViewId: ownerView,
    sizeLocked: locked,
    ownership: ResizeOwnership(
      ownerSurfaceId: ownerSurface,
      ownerViewId: ownerView,
      sizeLocked: locked,
      size: TerminalSize(cols: 80, rows: 24),
    ),
  );
}
