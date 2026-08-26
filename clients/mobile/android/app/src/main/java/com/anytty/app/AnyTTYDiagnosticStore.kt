package com.anytty.app

import android.content.Context
import android.os.Build
import android.os.SystemClock
import java.io.BufferedOutputStream
import java.io.File
import java.io.FileOutputStream
import java.time.Instant
import java.time.ZoneOffset
import java.time.format.DateTimeFormatter
import java.util.zip.ZipEntry
import java.util.zip.ZipOutputStream

internal data class AnyTTYDiagnosticBundle(
    val file: File,
    val name: String,
    val entryCount: Int,
)

/** Stores only structured, redacted diagnostics in app-private no-backup storage. */
internal object AnyTTYDiagnosticStore {
    private const val DIRECTORY = "anytty-diagnostics"
    private const val EVENT_FILE = "app-events.log"
    private const val NATIVE_FILE = "native-connection.log"
    private const val MAX_FILE_BYTES = 512L * 1024L
    private const val RETAINED_FILES = 4
    private const val MAX_VALUE_CHARS = 512
    private const val EXPORT_DIRECTORY = "anytty-diagnostic-exports"
    private val invalidCharacters = Regex("[^A-Za-z0-9_ =.-]")
    private val lock = Any()
    @Volatile private var appContext: Context? = null
    private var eventFiles: RollingDiagnosticFiles? = null

    fun init(context: Context) {
        synchronized(lock) {
            val applicationContext = context.applicationContext
            if (appContext === applicationContext && eventFiles != null) return
            appContext = applicationContext
            val directory = diagnosticDirectory(applicationContext)
            eventFiles = RollingDiagnosticFiles(directory, EVENT_FILE, MAX_FILE_BYTES, RETAINED_FILES)
            discardLegacyLogs(applicationContext)
        }
    }

    fun event(code: AnyTTYDebugEvent, value: String?) {
        writeLine(buildString {
            append(code.name)
            if (value != null) {
                append(' ')
                append(value)
            }
        })
    }

    fun connection(value: String) {
        writeLine("CONNECTION ${sanitize(value)}")
    }

    fun nativeLogPath(context: Context): String {
        init(context)
        return File(diagnosticDirectory(context.applicationContext), NATIVE_FILE).absolutePath
    }

    fun createBundle(context: Context): AnyTTYDiagnosticBundle {
        init(context)
        connection("diagnostic_bundle requested")
        synchronized(lock) {
            val applicationContext = context.applicationContext
            val exportDirectory = File(applicationContext.cacheDir, EXPORT_DIRECTORY).apply { mkdirs() }
            exportDirectory.listFiles()?.forEach { it.delete() }
            val createdAt = Instant.now()
            val name = "anytty-diagnostics-${EXPORT_TIMESTAMP.format(createdAt)}.zip"
            val target = File(exportDirectory, name)
            val sources = diagnosticSources(applicationContext)
            val manifest = diagnosticManifest(createdAt, sources)
            writeDiagnosticZip(target, manifest, sources)
            return AnyTTYDiagnosticBundle(target, name, sources.size + 1)
        }
    }

    internal fun sanitize(value: String): String = value
        .replace(invalidCharacters, "_")
        .take(MAX_VALUE_CHARS)

    private fun writeLine(value: String) {
        synchronized(lock) {
            val files = eventFiles ?: return
            val line = buildString {
                append(SystemClock.elapsedRealtime())
                append(" wall_ms=")
                append(System.currentTimeMillis())
                append(' ')
                append(value)
                append('\n')
            }.toByteArray(Charsets.US_ASCII)
            runCatching { files.append(line) }
        }
    }

    private fun diagnosticDirectory(context: Context): File =
        File(context.noBackupFilesDir, DIRECTORY).apply { mkdirs() }

    private fun diagnosticSources(context: Context): List<File> {
        val directory = diagnosticDirectory(context)
        return listOf(EVENT_FILE, NATIVE_FILE).flatMap { baseName ->
            buildList {
                for (index in RETAINED_FILES - 1 downTo 1) add(File(directory, "$baseName.$index"))
                add(File(directory, baseName))
            }
        }.filter { it.isFile && it.length() > 0L }
    }

    private fun diagnosticManifest(createdAt: Instant, sources: List<File>): String = buildString {
        appendLine("format=anytty-diagnostics-v1")
        appendLine("created_at=${DateTimeFormatter.ISO_INSTANT.format(createdAt)}")
        appendLine("app_version=${sanitize(BuildConfig.VERSION_NAME)}")
        appendLine("app_version_code=${BuildConfig.VERSION_CODE}")
        appendLine("android_sdk=${Build.VERSION.SDK_INT}")
        appendLine("abi=${sanitize(Build.SUPPORTED_ABIS.firstOrNull().orEmpty())}")
        appendLine("log_policy=redacted_structured_user_shared")
        appendLine("automatic_upload=false")
        appendLine("stream_file_limit_bytes=$MAX_FILE_BYTES")
        appendLine("retained_files_per_stream=$RETAINED_FILES")
        appendLine("entries=${sources.size}")
    }

    private fun discardLegacyLogs(context: Context) {
        listOf("anytty-debug-share", "anytty-debug-logs").forEach { name ->
            val legacy = File(context.cacheDir, name)
            legacy.listFiles()?.forEach { file -> if (file.isFile) file.delete() }
            legacy.delete()
        }
    }

    private val EXPORT_TIMESTAMP = DateTimeFormatter
        .ofPattern("yyyyMMdd'T'HHmmss'Z'")
        .withZone(ZoneOffset.UTC)
}

internal class RollingDiagnosticFiles(
    private val directory: File,
    private val baseName: String,
    private val maxFileBytes: Long,
    private val retainedFiles: Int,
) {
    init {
        require(baseName.isNotBlank() && !baseName.contains('/') && !baseName.contains('\\'))
        require(maxFileBytes > 0L && retainedFiles > 0)
        directory.mkdirs()
    }

    @Synchronized
    fun append(bytes: ByteArray) {
        require(bytes.size.toLong() <= maxFileBytes) { "diagnostic record exceeds the file limit" }
        val active = File(directory, baseName)
        if (active.length() + bytes.size > maxFileBytes) rotate()
        FileOutputStream(active, true).use { it.write(bytes) }
    }

    private fun rotate() {
        File(directory, "$baseName.${retainedFiles - 1}").delete()
        for (index in retainedFiles - 2 downTo 1) {
            val source = File(directory, "$baseName.$index")
            if (source.exists()) source.renameTo(File(directory, "$baseName.${index + 1}"))
        }
        val active = File(directory, baseName)
        if (retainedFiles > 1 && active.exists()) active.renameTo(File(directory, "$baseName.1"))
        else active.delete()
    }
}

internal fun writeDiagnosticZip(target: File, manifest: String, sources: List<File>) {
    ZipOutputStream(BufferedOutputStream(FileOutputStream(target))).use { zip ->
        zip.putNextEntry(ZipEntry("manifest.txt"))
        zip.write(manifest.toByteArray(Charsets.US_ASCII))
        zip.closeEntry()
        sources.forEach { source ->
            zip.putNextEntry(ZipEntry("logs/${source.name}"))
            source.inputStream().use { it.copyTo(zip) }
            zip.closeEntry()
        }
    }
}
