package com.anytty.app

import android.content.ContentValues
import android.content.Context
import android.net.Uri
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import java.io.File
import java.io.FileInputStream
import java.io.FileOutputStream
import java.io.OutputStream
import java.security.MessageDigest

/**
 * AndroidDownloadStore 是 Android 下载落盘的唯一平台 primitive。
 * 网络与 transfer lifecycle 由 Go resource stream 持有；partial bytes 始终落在 app-private 文件中，
 * 完成摘要校验后才发布到 Downloads/AnyTTY。
 */
class AndroidDownloadStore(private val context: Context) {
    data class SavedDownload(val uri: Uri, val path: String, val bytes: Long, val sha256: String)

    /** save 原子写入 Downloads/AnyTTY，并返回可由测试和 UI 校验的真实持久化结果。 */
    fun save(rawName: String, mimeType: String, bytes: ByteArray): SavedDownload {
        val name = safeName(rawName)
        val digest = MessageDigest.getInstance("SHA-256").digest(bytes)
        return publish(name, mimeType, bytes.size.toLong(), digest) { output -> output.write(bytes) }
    }

    @Synchronized
    fun resumeOffset(machineId: String, remotePath: String, totalSize: Long): Long {
        if (totalSize <= 0) return 0
        val file = partialFile(machineId, remotePath, totalSize)
        val length = file.length()
        if (length > totalSize) {
            file.delete()
            return 0
        }
        return length
    }

    @Synchronized
    fun appendPartial(machineId: String, remotePath: String, totalSize: Long, offset: Long, bytes: ByteArray): Long {
        require(totalSize >= 0 && offset >= 0 && offset <= totalSize) { "download partial offset is invalid" }
        require(bytes.isNotEmpty() && offset + bytes.size <= totalSize) { "download partial chunk is invalid" }
        val file = partialFile(machineId, remotePath, totalSize)
        require(file.length() == offset) { "download partial offset does not match persisted bytes" }
        FileOutputStream(file, true).use { output ->
            output.write(bytes)
            output.fd.sync()
        }
        return file.length()
    }

    @Synchronized
    fun commitPartial(
        rawName: String,
        mimeType: String,
        machineId: String,
        remotePath: String,
        totalSize: Long,
        expectedSHA256: ByteArray,
    ): SavedDownload {
        require(expectedSHA256.size == 32) { "download SHA-256 is invalid" }
        val part = partialFile(machineId, remotePath, totalSize)
        require(part.isFile && part.length() == totalSize) { "download partial size is incomplete" }
        val actual = digest(part)
        require(MessageDigest.isEqual(actual, expectedSHA256)) { "download SHA-256 mismatch" }
        val saved = publish(safeName(rawName), mimeType, totalSize, actual) { output ->
            FileInputStream(part).use { input -> input.copyTo(output) }
        }
        part.delete()
        return saved
    }

    @Synchronized
    fun discardPartial(machineId: String, remotePath: String, totalSize: Long): Boolean {
        val file = partialFile(machineId, remotePath, totalSize)
        return !file.exists() || file.delete()
    }

    private fun publish(name: String, mimeType: String, bytes: Long, digest: ByteArray, writer: (OutputStream) -> Unit): SavedDownload {
        val digestHex = digest.joinToString("") { "%02x".format(it) }
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) saveMediaStore(name, mimeType, bytes, digestHex, writer)
        else saveLegacy(name, bytes, digestHex, writer)
    }

    private fun saveMediaStore(name: String, mimeType: String, bytes: Long, digest: String, writer: (OutputStream) -> Unit): SavedDownload {
        val resolver = context.contentResolver
        val relativePath = Environment.DIRECTORY_DOWNLOADS + "/AnyTTY"
        resolver.query(
            MediaStore.Downloads.EXTERNAL_CONTENT_URI,
            arrayOf(MediaStore.Downloads._ID),
            "${MediaStore.Downloads.DISPLAY_NAME}=? AND ${MediaStore.Downloads.RELATIVE_PATH}=?",
            arrayOf(name, "$relativePath/"),
            null,
        )?.use { cursor ->
            while (cursor.moveToNext()) {
                resolver.delete(Uri.withAppendedPath(MediaStore.Downloads.EXTERNAL_CONTENT_URI, cursor.getLong(0).toString()), null, null)
            }
        }
        val values = ContentValues().apply {
            put(MediaStore.Downloads.DISPLAY_NAME, name)
            put(MediaStore.Downloads.MIME_TYPE, mimeType)
            put(MediaStore.Downloads.RELATIVE_PATH, relativePath)
            put(MediaStore.Downloads.IS_PENDING, 1)
        }
        val uri = resolver.insert(MediaStore.Downloads.EXTERNAL_CONTENT_URI, values)
            ?: throw IllegalStateException("Android MediaStore rejected download")
        try {
            resolver.openOutputStream(uri, "w")?.use(writer)
                ?: throw IllegalStateException("Android MediaStore output is unavailable")
            resolver.update(uri, ContentValues().apply { put(MediaStore.Downloads.IS_PENDING, 0) }, null, null)
        } catch (failure: Throwable) {
            resolver.delete(uri, null, null)
            throw failure
        }
        return SavedDownload(uri, "Downloads/AnyTTY/$name", bytes, digest)
    }

    @Suppress("DEPRECATION")
    private fun saveLegacy(name: String, bytes: Long, digest: String, writer: (OutputStream) -> Unit): SavedDownload {
        val directory = File(context.getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS), "AnyTTY").apply { mkdirs() }
        val target = File(directory, name)
        target.outputStream().use(writer)
        return SavedDownload(Uri.fromFile(target), target.absolutePath, bytes, digest)
    }

    private fun partialFile(machineId: String, remotePath: String, totalSize: Long): File {
        val key = MessageDigest.getInstance("SHA-256")
            .digest("$machineId\u0000$remotePath\u0000$totalSize".toByteArray(Charsets.UTF_8))
            .joinToString("") { "%02x".format(it) }
        val directory = File(context.filesDir, "file-transfers").apply { mkdirs() }
        return File(directory, "$key.part")
    }

    private fun digest(file: File): ByteArray {
        val digest = MessageDigest.getInstance("SHA-256")
        FileInputStream(file).use { input ->
            val buffer = ByteArray(256 * 1024)
            while (true) {
                val count = input.read(buffer)
                if (count < 0) break
                if (count > 0) digest.update(buffer, 0, count)
            }
        }
        return digest.digest()
    }

    private fun safeName(rawName: String): String {
        val name = rawName.trim()
        require(name.isNotEmpty() && name != "." && name != ".." && !name.contains('/') && !name.contains('\\') && !name.contains('\u0000')) {
            "download file name is invalid"
        }
        return name
    }
}
