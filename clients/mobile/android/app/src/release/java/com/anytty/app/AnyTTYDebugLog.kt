package com.anytty.app

import android.content.Context

object AnyTTYDebugLog {
    @JvmStatic fun init(context: Context) = Unit
    @JvmStatic fun event(code: AnyTTYDebugEvent) = Unit
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Int) = Unit
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Long) = Unit
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Boolean) = Unit
}
