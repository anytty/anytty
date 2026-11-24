import 'package:anytty_native/src/features/terminal/domain/terminal_settings.dart';
import 'package:anytty_native/src/features/terminal/presentation/terminal_swipe_switcher.dart';
import 'package:flutter/gestures.dart' show Velocity;
import 'package:flutter_test/flutter_test.dart';

Velocity _fling(double dx) => Velocity(pixelsPerSecond: Offset(dx, 0));

void main() {
  group('TerminalSwipeLogic 角度门控', () {
    test('横向占优才接管，竖向拖动一律让给滚动', () {
      final logic = TerminalSwipeLogic(
        sensitivity: TerminalSwipeSensitivity.normal,
      );
      expect(logic.shouldEngage(-20, 2), isTrue);
      expect(logic.shouldEngage(-20, 20), isFalse, reason: '45° 应让给竖向');
      expect(logic.shouldEngage(-20, 15), isFalse, reason: '超过 27° 不接管');
      expect(logic.shouldEngage(4, 0), isFalse, reason: '未过 engageSlop');
    });

    test('灵敏度只影响阈值，不放松角度门控', () {
      final strict = TerminalSwipeLogic(
        sensitivity: TerminalSwipeSensitivity.strict,
      );
      expect(strict.commitThreshold, greaterThan(32.0));
      expect(strict.shouldEngage(-20, 15), isFalse);
    });
  });

  group('TerminalSwipeLogic 画面位移', () {
    final logic = TerminalSwipeLogic(
      sensitivity: TerminalSwipeSensitivity.normal,
    );

    test('界内完全跟手', () {
      expect(
        logic.visualOffset(-80, canNext: true, canPrevious: true),
        -80.0,
      );
      expect(logic.visualOffset(40, canNext: true, canPrevious: true), 40.0);
    });

    test('边界外橡皮筋衰减，位移明显小于手指位移', () {
      final offset = logic.visualOffset(
        -200,
        canNext: false,
        canPrevious: true,
      );
      expect(offset, greaterThan(-200.0));
      expect(offset, lessThan(0.0));
    });
  });

  group('TerminalSwipeLogic 松手决策', () {
    final logic = TerminalSwipeLogic(
      sensitivity: TerminalSwipeSensitivity.normal,
    );

    test('位移过阈值即切换', () {
      expect(
        logic.decide(
          -40,
          Velocity.zero,
          canNext: true,
          canPrevious: true,
        ),
        TerminalSwipeDecision.next,
      );
      expect(
        logic.decide(40, Velocity.zero, canNext: true, canPrevious: true),
        TerminalSwipeDecision.previous,
      );
    });

    test('位移不足但同向甩动也切换', () {
      expect(
        logic.decide(-20, _fling(-500), canNext: true, canPrevious: true),
        TerminalSwipeDecision.next,
      );
      expect(
        logic.decide(-20, _fling(500), canNext: true, canPrevious: true),
        TerminalSwipeDecision.cancel,
        reason: '反向速度不算甩动',
      );
    });

    test('位移与速度都不足则弹回', () {
      expect(
        logic.decide(-12, _fling(-100), canNext: true, canPrevious: true),
        TerminalSwipeDecision.cancel,
      );
    });

    test('到头继续甩出打开切换面板', () {
      expect(
        logic.decide(-120, Velocity.zero, canNext: false, canPrevious: true),
        TerminalSwipeDecision.openSwitcher,
      );
      expect(
        logic.decide(-30, Velocity.zero, canNext: false, canPrevious: true),
        TerminalSwipeDecision.cancel,
        reason: '界内小幅拖动只弹回',
      );
      expect(
        logic.decide(120, Velocity.zero, canNext: true, canPrevious: false),
        TerminalSwipeDecision.openSwitcher,
      );
    });
  });

  group('TerminalSwipeSensitivity', () {
    test('阈值乘数递增', () {
      expect(TerminalSwipeSensitivity.relaxed.thresholdFactor, lessThan(1));
      expect(TerminalSwipeSensitivity.normal.thresholdFactor, 1);
      expect(
        TerminalSwipeSensitivity.strict.thresholdFactor,
        greaterThan(1),
      );
    });
  });
}
