package com.anytty.app

import android.content.Context
import android.os.SystemClock
import java.io.File
import java.io.FileOutputStream

object AnyTTYDebugLog {
    private const val SHARE_DIR = "anytty-debug-share"
    private const val LOG_FILE = "anytty-debug-events.log"
	private const val MAX_BYTES = 2 * 1024 * 1024L
    private val lock = Any()
    @Volatile private var appContext: Context? = null

    @JvmStatic
    fun init(context: Context) {
        appContext = context.applicationContext
    }

    @JvmStatic fun event(code: AnyTTYDebugEvent) = write(code, null)
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Int) = write(code, value.toString())
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Long) = write(code, value.toString())
    @JvmStatic fun event(code: AnyTTYDebugEvent, value: Boolean) = write(code, value.toString())
	@JvmStatic fun connection(value: String) = writeLine("CONNECTION ${sanitize(value)}")
    private fun write(code: AnyTTYDebugEvent, value: String?) {
		writeLine(buildString {
			append(code.name)
			if (value != null) {
				append(' ')
				append(value)
			}
		})
	}

	private fun sanitize(value: String): String = value
		.replace(Regex("[^A-Za-z0-9_ =.-]"), "_")
		.take(512)

	private fun writeLine(value: String) {
        val context = appContext ?: return
        synchronized(lock) {
            val directory = File(context.cacheDir, SHARE_DIR)
            if (!directory.exists() && !directory.mkdirs()) return
            val file = File(directory, LOG_FILE)
            if (file.length() >= MAX_BYTES) file.writeBytes(ByteArray(0))
            val line = buildString {
                append(SystemClock.elapsedRealtime())
                append(' ')
				append(value)
                append('\n')
            }
            runCatching {
                FileOutputStream(file, true).use { it.write(line.toByteArray(Charsets.US_ASCII)) }
            }
        }
    }
}
