package com.anytty.app

import android.app.Activity
import android.content.Context
import android.text.InputType
import android.view.Gravity
import android.view.KeyEvent
import android.view.View
import android.view.ViewGroup
import android.view.inputmethod.BaseInputConnection
import android.view.inputmethod.EditorInfo
import android.view.inputmethod.InputConnection
import android.view.inputmethod.InputMethodManager
import android.widget.FrameLayout
import io.flutter.plugin.common.BinaryMessenger
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel

/** Owns the Android IME connection used by terminal surfaces. */
class AndroidTerminalInputBridge(
    private val activity: Activity,
    messenger: BinaryMessenger,
) {
    private val channel = MethodChannel(messenger, CHANNEL_NAME)
    private val inputMethodManager =
        activity.getSystemService(Context.INPUT_METHOD_SERVICE) as InputMethodManager
    private val inputView = TerminalInputView(activity, ::publishEvent)
    private var activeOwner: String? = null

    init {
        channel.setMethodCallHandler(::handleMethodCall)
        inputView.id = View.generateViewId()
        activity.window.decorView.post(::ensureInputViewAttached)
    }

    fun close() {
        channel.setMethodCallHandler(null)
        activeOwner = null
        inputMethodManager.hideSoftInputFromWindow(inputView.windowToken, 0)
        inputView.clearFocus()
        (inputView.parent as? ViewGroup)?.removeView(inputView)
    }

    private fun handleMethodCall(call: MethodCall, result: MethodChannel.Result) {
        when (call.method) {
            SHOW_METHOD -> {
                val owner = call.argument<String>(OWNER_ARGUMENT)?.trim().orEmpty()
                if (owner.isEmpty()) {
                    result.error("invalid_owner", "Terminal input owner is required", null)
                    return
                }
                activeOwner = owner
                activity.window.decorView.post {
                    if (activeOwner != owner) {
                        result.success(null)
                        return@post
                    }
                    ensureInputViewAttached()
                    inputView.requestFocus()
                    inputView.post {
                        if (activeOwner != owner) {
                            result.success(null)
                            return@post
                        }
                        inputMethodManager.restartInput(inputView)
                        inputMethodManager.showSoftInput(
                            inputView,
                            InputMethodManager.SHOW_IMPLICIT,
                        )
                        result.success(null)
                    }
                }
            }

            HIDE_METHOD, RELEASE_METHOD -> {
                val owner = call.argument<String>(OWNER_ARGUMENT)?.trim().orEmpty()
                if (owner.isNotEmpty() && activeOwner == owner) {
                    activeOwner = null
                    inputMethodManager.hideSoftInputFromWindow(inputView.windowToken, 0)
                    inputView.clearFocus()
                }
                result.success(null)
            }

            else -> result.notImplemented()
        }
    }

    private fun publishEvent(event: Map<String, Any>) {
        val owner = activeOwner ?: return
        channel.invokeMethod(EVENT_METHOD, event + (OWNER_ARGUMENT to owner))
    }

    private fun ensureInputViewAttached() {
        if (inputView.parent != null) return
        val content = activity.findViewById<ViewGroup>(android.R.id.content) ?: return
        content.addView(
            inputView,
            FrameLayout.LayoutParams(1, 1, Gravity.START or Gravity.BOTTOM),
        )
    }

    private class TerminalInputView(
        context: Context,
        private val publish: (Map<String, Any>) -> Unit,
    ) : View(context) {
        init {
            isFocusable = true
            isFocusableInTouchMode = true
            importantForAutofill = IMPORTANT_FOR_AUTOFILL_NO_EXCLUDE_DESCENDANTS
            alpha = 0.01f
        }

        override fun onCheckIsTextEditor(): Boolean = true

        override fun onCreateInputConnection(outAttrs: EditorInfo): InputConnection {
            outAttrs.inputType = InputType.TYPE_CLASS_TEXT or
                InputType.TYPE_TEXT_VARIATION_NORMAL or
                InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS
            outAttrs.imeOptions = EditorInfo.IME_ACTION_NONE or
                EditorInfo.IME_FLAG_NO_FULLSCREEN or
                EditorInfo.IME_FLAG_NO_EXTRACT_UI
            outAttrs.initialSelStart = 0
            outAttrs.initialSelEnd = 0
            return TerminalInputConnection()
        }

        override fun onKeyDown(keyCode: Int, event: KeyEvent): Boolean =
            handleKeyEvent(event) || super.onKeyDown(keyCode, event)

        override fun onKeyUp(keyCode: Int, event: KeyEvent): Boolean =
            handleKeyEvent(event) || super.onKeyUp(keyCode, event)

        private fun handleKeyEvent(event: KeyEvent): Boolean {
            val terminalKey = isTerminalKey(event.keyCode)
            val unicodeCodePoint = event.getUnicodeChar(
                event.metaState and (KeyEvent.META_CTRL_MASK or KeyEvent.META_META_MASK).inv(),
            )
            if (!terminalKey && unicodeCodePoint == 0) return false
            if (event.action == KeyEvent.ACTION_DOWN) {
                publish(
                    mapOf(
                        TYPE_ARGUMENT to KEY_TYPE,
                        KEY_CODE_ARGUMENT to event.keyCode,
                        MODIFIERS_ARGUMENT to terminalModifiers(event),
                        UNSHIFTED_CODEPOINT_ARGUMENT to event.getUnicodeChar(0),
                        TEXT_ARGUMENT to unicodeText(unicodeCodePoint),
                    ),
                )
            }
            return event.action == KeyEvent.ACTION_DOWN || event.action == KeyEvent.ACTION_UP
        }

        private fun terminalModifiers(event: KeyEvent): Int {
            var modifiers = 0
            if (event.isShiftPressed) modifiers = modifiers or SHIFT_MODIFIER
            if (event.isCtrlPressed) modifiers = modifiers or CONTROL_MODIFIER
            if (event.isAltPressed) modifiers = modifiers or ALT_MODIFIER
            if (event.isMetaPressed) modifiers = modifiers or SUPER_MODIFIER
            return modifiers
        }

        private fun unicodeText(codePoint: Int): String =
            if (codePoint <= 0 || !Character.isValidCodePoint(codePoint)) {
                ""
            } else {
                String(Character.toChars(codePoint))
            }

        private fun isTerminalKey(keyCode: Int): Boolean =
            keyCode in KeyEvent.KEYCODE_0..KeyEvent.KEYCODE_9 ||
                keyCode in KeyEvent.KEYCODE_A..KeyEvent.KEYCODE_Z ||
                keyCode in KeyEvent.KEYCODE_F1..KeyEvent.KEYCODE_F12 ||
                keyCode in TERMINAL_KEY_CODES

        private inner class TerminalInputConnection :
            BaseInputConnection(this@TerminalInputView, true) {
            override fun setComposingText(text: CharSequence, newCursorPosition: Int): Boolean {
                val composing = super.setComposingText(text, newCursorPosition)
                publishComposition(activeOverride = true)
                return composing
            }

            override fun setComposingRegion(start: Int, end: Int): Boolean {
                val composing = super.setComposingRegion(start, end)
                publishComposition()
                return composing
            }

            override fun commitText(text: CharSequence, newCursorPosition: Int): Boolean {
                val committed = super.commitText(text, newCursorPosition)
                publishComposition(activeOverride = false)
                flushEditable()
                return committed
            }

            override fun finishComposingText(): Boolean {
                val finished = super.finishComposingText()
                publishComposition(activeOverride = false)
                flushEditable()
                return finished
            }

            override fun deleteSurroundingText(beforeLength: Int, afterLength: Int): Boolean =
                deleteOrEditComposition(beforeLength, afterLength, inCodePoints = false)

            override fun deleteSurroundingTextInCodePoints(
                beforeLength: Int,
                afterLength: Int,
            ): Boolean = deleteOrEditComposition(beforeLength, afterLength, inCodePoints = true)

            override fun performEditorAction(actionCode: Int): Boolean {
                publish(mapOf(TYPE_ARGUMENT to ENTER_TYPE))
                return true
            }

            override fun sendKeyEvent(event: KeyEvent): Boolean {
                if (handleKeyEvent(event)) return true
                return super.sendKeyEvent(event)
            }

            private fun deleteOrEditComposition(
                beforeLength: Int,
                afterLength: Int,
                inCodePoints: Boolean,
            ): Boolean {
                if (hasComposingText()) {
                    val edited = if (inCodePoints) {
                        super.deleteSurroundingTextInCodePoints(beforeLength, afterLength)
                    } else {
                        super.deleteSurroundingText(beforeLength, afterLength)
                    }
                    publishComposition()
                    return edited
                }
                val count = when {
                    beforeLength > 0 -> beforeLength
                    beforeLength == 0 && afterLength == 0 -> 1
                    else -> 0
                }.coerceAtMost(MAX_DELETE_REPEAT)
                if (count > 0) {
                    publish(mapOf(TYPE_ARGUMENT to BACKSPACE_TYPE, COUNT_ARGUMENT to count))
                }
                return true
            }

            private fun hasComposingText(): Boolean {
                val content = editable ?: return false
                val start = getComposingSpanStart(content)
                val end = getComposingSpanEnd(content)
                return start >= 0 && end > start
            }

            private fun publishComposition(activeOverride: Boolean? = null) {
                val content = editable
                val start = content?.let(::getComposingSpanStart) ?: -1
                val end = content?.let(::getComposingSpanEnd) ?: -1
                val active = activeOverride ?: (start >= 0 && end > start)
                val text = if (active && content != null && start >= 0 && end >= start) {
                    content.subSequence(start, end).toString()
                } else {
                    ""
                }
                publish(
                    mapOf(
                        TYPE_ARGUMENT to COMPOSITION_TYPE,
                        TEXT_ARGUMENT to text,
                        ACTIVE_ARGUMENT to active,
                    ),
                )
            }

            private fun flushEditable() {
                val content = editable ?: return
                if (content.isEmpty()) return
                val text = content.toString()
                content.clear()
                publish(mapOf(TYPE_ARGUMENT to TEXT_TYPE, TEXT_ARGUMENT to text))
            }
        }

        private companion object {
            const val SHIFT_MODIFIER = 1 shl 0
            const val CONTROL_MODIFIER = 1 shl 1
            const val ALT_MODIFIER = 1 shl 2
            const val SUPER_MODIFIER = 1 shl 3
            const val MAX_DELETE_REPEAT = 64

            val TERMINAL_KEY_CODES = setOf(
                KeyEvent.KEYCODE_ENTER,
                KeyEvent.KEYCODE_ESCAPE,
                KeyEvent.KEYCODE_DEL,
                KeyEvent.KEYCODE_TAB,
                KeyEvent.KEYCODE_SPACE,
                KeyEvent.KEYCODE_MINUS,
                KeyEvent.KEYCODE_EQUALS,
                KeyEvent.KEYCODE_LEFT_BRACKET,
                KeyEvent.KEYCODE_RIGHT_BRACKET,
                KeyEvent.KEYCODE_BACKSLASH,
                KeyEvent.KEYCODE_SEMICOLON,
                KeyEvent.KEYCODE_APOSTROPHE,
                KeyEvent.KEYCODE_GRAVE,
                KeyEvent.KEYCODE_COMMA,
                KeyEvent.KEYCODE_PERIOD,
                KeyEvent.KEYCODE_SLASH,
                KeyEvent.KEYCODE_INSERT,
                KeyEvent.KEYCODE_MOVE_HOME,
                KeyEvent.KEYCODE_PAGE_UP,
                KeyEvent.KEYCODE_FORWARD_DEL,
                KeyEvent.KEYCODE_MOVE_END,
                KeyEvent.KEYCODE_PAGE_DOWN,
                KeyEvent.KEYCODE_DPAD_RIGHT,
                KeyEvent.KEYCODE_DPAD_LEFT,
                KeyEvent.KEYCODE_DPAD_DOWN,
                KeyEvent.KEYCODE_DPAD_UP,
            )
        }
    }

    private companion object {
        const val CHANNEL_NAME = "com.anytty.app/terminal-input"
        const val SHOW_METHOD = "show"
        const val HIDE_METHOD = "hide"
        const val RELEASE_METHOD = "release"
        const val EVENT_METHOD = "event"
        const val OWNER_ARGUMENT = "owner"
        const val TYPE_ARGUMENT = "type"
        const val TEXT_ARGUMENT = "text"
        const val ACTIVE_ARGUMENT = "active"
        const val COUNT_ARGUMENT = "count"
        const val KEY_CODE_ARGUMENT = "keyCode"
        const val MODIFIERS_ARGUMENT = "modifiers"
        const val UNSHIFTED_CODEPOINT_ARGUMENT = "unshiftedCodepoint"
        const val TEXT_TYPE = "text"
        const val COMPOSITION_TYPE = "composition"
        const val BACKSPACE_TYPE = "backspace"
        const val ENTER_TYPE = "enter"
        const val KEY_TYPE = "key"
    }
}
