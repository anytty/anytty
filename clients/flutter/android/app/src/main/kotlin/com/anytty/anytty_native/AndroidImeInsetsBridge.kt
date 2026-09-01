package com.anytty.app

import android.view.View
import androidx.core.view.ViewCompat
import androidx.core.view.WindowInsetsAnimationCompat
import androidx.core.view.WindowInsetsCompat
import io.flutter.plugin.common.BinaryMessenger
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import kotlin.math.abs
import kotlin.math.roundToInt

/** Publishes the system IME's actual animated inset without owning app layout. */
class AndroidImeInsetsBridge(
    private val decorView: View,
    messenger: BinaryMessenger,
) {
    private val channel = MethodChannel(messenger, CHANNEL_NAME)
    private var lastInset = -1
    private var imeStartInset = 0
    private var imeEndInset = 0

    private val layoutListener = View.OnLayoutChangeListener { _, _, _, _, _, _, _, _, _ ->
        publishCurrentInset()
    }

    private val animationCallback = object : WindowInsetsAnimationCompat.Callback(
        WindowInsetsAnimationCompat.Callback.DISPATCH_MODE_CONTINUE_ON_SUBTREE,
    ) {
        override fun onPrepare(animation: WindowInsetsAnimationCompat) {
            super.onPrepare(animation)
            if (!animation.animatesIme) return
            imeStartInset = currentInset()
            imeEndInset = imeStartInset
            publishInset(imeStartInset, force = true)
        }

        override fun onStart(
            animation: WindowInsetsAnimationCompat,
            bounds: WindowInsetsAnimationCompat.BoundsCompat,
        ): WindowInsetsAnimationCompat.BoundsCompat {
            if (animation.animatesIme) {
                val lower = bounds.lowerBound.bottom
                val upper = bounds.upperBound.bottom
                imeEndInset = if (abs(imeStartInset - lower) <= abs(imeStartInset - upper)) {
                    upper
                } else {
                    lower
                }
            }
            return bounds
        }

        override fun onProgress(
            insets: WindowInsetsCompat,
            runningAnimations: MutableList<WindowInsetsAnimationCompat>,
        ): WindowInsetsCompat {
            val imeAnimation = runningAnimations.lastOrNull { it.animatesIme }
            if (imeAnimation == null) {
                publishInset(insets.getInsets(WindowInsetsCompat.Type.ime()).bottom)
                return insets
            }
            val progress = imeAnimation.interpolatedFraction.coerceIn(0f, 1f)
            publishInset(
                (imeStartInset + (imeEndInset - imeStartInset) * progress).roundToInt(),
            )
            return insets
        }

        override fun onEnd(animation: WindowInsetsAnimationCompat) {
            super.onEnd(animation)
            publishCurrentInset(force = true)
        }
    }

    init {
        channel.setMethodCallHandler(::handleMethodCall)
        ViewCompat.setWindowInsetsAnimationCallback(decorView, animationCallback)
        decorView.addOnLayoutChangeListener(layoutListener)
        decorView.post { publishCurrentInset(force = true) }
    }

    fun close() {
        decorView.removeOnLayoutChangeListener(layoutListener)
        ViewCompat.setWindowInsetsAnimationCallback(decorView, null)
        channel.setMethodCallHandler(null)
    }

    private fun handleMethodCall(call: MethodCall, result: MethodChannel.Result) {
        if (call.method != CURRENT_INSET_METHOD) {
            result.notImplemented()
            return
        }
        result.success(currentInset())
    }

    private fun publishCurrentInset(force: Boolean = false) {
        publishInset(currentInset(), force)
    }

    private fun currentInset(): Int {
        val insets = ViewCompat.getRootWindowInsets(decorView) ?: return 0
        if (!insets.isVisible(WindowInsetsCompat.Type.ime())) return 0
        return insets.getInsets(WindowInsetsCompat.Type.ime()).bottom
    }

    private fun publishInset(inset: Int, force: Boolean = false) {
        if (!force && inset == lastInset) return
        lastInset = inset
        channel.invokeMethod(INSET_CHANGED_METHOD, inset)
    }

    private val WindowInsetsAnimationCompat.animatesIme: Boolean
        get() = typeMask and WindowInsetsCompat.Type.ime() != 0

    private companion object {
        const val CHANNEL_NAME = "com.anytty.app/ime-insets"
        const val CURRENT_INSET_METHOD = "currentInset"
        const val INSET_CHANGED_METHOD = "insetChanged"
    }
}
