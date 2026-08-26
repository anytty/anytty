package com.anytty.app

import android.content.Context

object AnyTTYDebugLog {
    @JvmStatic fun init(context: Context) = AnyTTYDiagnosticStore.init(context)
    @JvmStatic fun event(code: AnyTTYDebugEvent) = AnyTTYDiagnosticStore.event(code, null)
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Int) = AnyTTYDiagnosticStore.event(code, value.toString())
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Long) = AnyTTYDiagnosticStore.event(code, value.toString())
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Boolean) = AnyTTYDiagnosticStore.event(code, value.toString())
    @JvmStatic fun connection(value: String) = AnyTTYDiagnosticStore.connection(value)
}
