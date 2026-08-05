package com.anytty.app

import android.app.Activity
import android.content.Intent
import android.database.Cursor
import android.net.Uri
import android.provider.OpenableColumns
import android.util.Base64
import androidx.activity.result.ActivityResult
import com.getcapacitor.JSArray
import com.getcapacitor.JSObject
import com.getcapacitor.Plugin
import com.getcapacitor.PluginCall
import com.getcapacitor.PluginMethod
import com.getcapacitor.annotation.ActivityCallback
import com.getcapacitor.annotation.CapacitorPlugin
import java.io.InputStream
import java.security.MessageDigest
import java.util.UUID
import java.util.concurrent.ConcurrentHashMap
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch

/**
 * NativeFilePickerPlugin — 使用 Android SAF (ACTION_OPEN_DOCUMENT) 选择文件
 *
 * 返回 content:// URI，供 WebView 读取并通过 Go resource stream 上传。
 */
@CapacitorPlugin(name = "NativeFilePicker")
class NativeFilePickerPlugin : Plugin() {

    companion object {
        private const val MAX_TRANSFER_CHUNK_BYTES = 4 * 1024 * 1024
    }

    private data class UploadSource(
        val input: InputStream,
        val digest: MessageDigest,
        val totalSize: Long,
        var offset: Long,
    )

    private val ioScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private val uploadSources = ConcurrentHashMap<String, UploadSource>()
    private val downloadStore by lazy { AndroidDownloadStore(context.applicationContext) }

    override fun handleOnDestroy() {
        uploadSources.values.forEach { runCatching { it.input.close() } }
        uploadSources.clear()
        ioScope.cancel()
        super.handleOnDestroy()
    }

    @PluginMethod
    fun pickFiles(call: PluginCall) {
        val multiple = call.getBoolean("multiple", false) ?: false

        val intent = Intent(Intent.ACTION_OPEN_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            addFlags(Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION)
            type = "*/*"
            if (multiple) {
                putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true)
            }
        }

        startActivityForResult(call, intent, "pickFilesResult")
    }

    /** saveFile 在 Go resource stream 完整校验后把下载内容明确提交到 Android Downloads/AnyTTY。 */
    @PluginMethod
    fun saveFile(call: PluginCall) {
        val name = call.getString("name").orEmpty()
        val mimeType = call.getString("mimeType") ?: "application/octet-stream"
        val encoded = call.getString("dataBase64").orEmpty()
        ioScope.launch {
            try {
                val bytes = Base64.decode(encoded, Base64.DEFAULT)
                call.resolve(savedDownload(downloadStore.save(name, mimeType, bytes)))
            } catch (failure: Exception) {
                AnyTTYDebugLog.event(AnyTTYDebugEvent.FILE_SAVE_FAILED)
                call.reject("failed to save download: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun getDownloadResumeOffset(call: PluginCall) {
        val machineId = call.getString("machineId").orEmpty()
        val remotePath = call.getString("remotePath").orEmpty()
        val totalSize = call.data.optLong("totalSize", 0L)
        ioScope.launch {
            try {
                val offset = downloadStore.resumeOffset(machineId, remotePath, totalSize)
                call.resolve(JSObject().put("offset", offset))
            } catch (failure: Exception) {
                call.reject("failed to inspect download partial: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun appendDownloadPartial(call: PluginCall) {
        val machineId = call.getString("machineId").orEmpty()
        val remotePath = call.getString("remotePath").orEmpty()
        val totalSize = call.data.optLong("totalSize", 0L)
        val offset = call.data.optLong("offset", -1L)
        val encoded = call.getString("dataBase64").orEmpty()
        ioScope.launch {
            try {
                val bytes = Base64.decode(encoded, Base64.DEFAULT)
                require(bytes.size in 1..MAX_TRANSFER_CHUNK_BYTES) { "download partial chunk is too large" }
                val persisted = downloadStore.appendPartial(machineId, remotePath, totalSize, offset, bytes)
                call.resolve(JSObject().put("offset", persisted))
            } catch (failure: Exception) {
                call.reject("failed to append download partial: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun commitDownloadPartial(call: PluginCall) {
        val name = call.getString("name").orEmpty()
        val mimeType = call.getString("mimeType") ?: "application/octet-stream"
        val machineId = call.getString("machineId").orEmpty()
        val remotePath = call.getString("remotePath").orEmpty()
        val totalSize = call.data.optLong("totalSize", 0L)
        val encodedSHA256 = call.getString("sha256Base64").orEmpty()
        ioScope.launch {
            try {
                val expectedSHA256 = Base64.decode(encodedSHA256, Base64.DEFAULT)
                val saved = downloadStore.commitPartial(
                    name, mimeType, machineId, remotePath, totalSize, expectedSHA256,
                )
                call.resolve(savedDownload(saved))
            } catch (failure: Exception) {
                call.reject("failed to commit download partial: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun discardDownloadPartial(call: PluginCall) {
        val machineId = call.getString("machineId").orEmpty()
        val remotePath = call.getString("remotePath").orEmpty()
        val totalSize = call.data.optLong("totalSize", 0L)
        ioScope.launch {
            try {
                val discarded = downloadStore.discardPartial(machineId, remotePath, totalSize)
                call.resolve(JSObject().put("discarded", discarded))
            } catch (failure: Exception) {
                call.reject("failed to discard download partial: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun openUploadSource(call: PluginCall) {
        val contentUri = call.getString("contentUri").orEmpty()
        val requestedOffset = call.data.optLong("offset", 0L)
        val totalSize = call.data.optLong("totalSize", -1L)
        ioScope.launch {
            var input: InputStream? = null
            try {
                require(requestedOffset >= 0 && totalSize >= 0 && requestedOffset <= totalSize) { "upload source offset is invalid" }
                input = context.contentResolver.openInputStream(Uri.parse(contentUri))
                    ?: throw IllegalStateException("upload source is unavailable")
                val digest = MessageDigest.getInstance("SHA-256")
                var offset = 0L
                val buffer = ByteArray(256 * 1024)
                while (offset < requestedOffset) {
                    val count = input.read(buffer, 0, minOf(buffer.size.toLong(), requestedOffset - offset).toInt())
                    if (count <= 0) throw IllegalStateException("upload source is shorter than the resume offset")
                    digest.update(buffer, 0, count)
                    offset += count
                }
                val handle = UUID.randomUUID().toString()
                uploadSources[handle] = UploadSource(input, digest, totalSize, offset)
                input = null
                call.resolve(JSObject().put("handle", handle).put("offset", offset))
            } catch (failure: Exception) {
                runCatching { input?.close() }
                call.reject("failed to open upload source: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun readUploadSource(call: PluginCall) {
        val handle = call.getString("handle").orEmpty()
        val requested = call.data.optInt("length", 0)
        ioScope.launch {
            try {
                require(requested in 1..MAX_TRANSFER_CHUNK_BYTES) { "upload source chunk length is invalid" }
                val source = uploadSources[handle] ?: throw IllegalStateException("upload source handle is unavailable")
                val result = synchronized(source) {
                    val remaining = source.totalSize - source.offset
                    val bytes = ByteArray(minOf(requested.toLong(), remaining).toInt())
                    var count = 0
                    while (count < bytes.size) {
                        val read = source.input.read(bytes, count, bytes.size - count)
                        if (read < 0) break
                        check(read > 0) { "upload source returned no data" }
                        count += read
                    }
                    if (count > 0) {
                        source.digest.update(bytes, 0, count)
                        source.offset += count
                    }
                    val payload = if (count == bytes.size) bytes else bytes.copyOf(count)
                    JSObject()
                        .put("dataBase64", Base64.encodeToString(payload, Base64.NO_WRAP))
                        .put("offset", source.offset)
                        .put("eof", source.offset == source.totalSize)
                }
                call.resolve(result)
            } catch (failure: Exception) {
                call.reject("failed to read upload source: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun finishUploadSource(call: PluginCall) {
        val handle = call.getString("handle").orEmpty()
        ioScope.launch {
            try {
                val source = uploadSources.remove(handle) ?: throw IllegalStateException("upload source handle is unavailable")
                val digest = synchronized(source) {
                    source.input.close()
                    require(source.offset == source.totalSize) { "upload source was not fully read" }
                    source.digest.digest()
                }
                call.resolve(JSObject().put("sha256Base64", Base64.encodeToString(digest, Base64.NO_WRAP)))
            } catch (failure: Exception) {
                call.reject("failed to finish upload source: ${failure.message}", failure)
            }
        }
    }

    @PluginMethod
    fun closeUploadSource(call: PluginCall) {
        val handle = call.getString("handle").orEmpty()
        ioScope.launch {
            val source = uploadSources.remove(handle)
            if (source != null) synchronized(source) { runCatching { source.input.close() } }
            call.resolve()
        }
    }

    private fun savedDownload(saved: AndroidDownloadStore.SavedDownload): JSObject = JSObject().apply {
        put("uri", saved.uri.toString())
        put("path", saved.path)
        put("bytes", saved.bytes)
        put("sha256", saved.sha256)
    }

    @ActivityCallback
    private fun pickFilesResult(call: PluginCall, result: ActivityResult) {
        if (result.resultCode != Activity.RESULT_OK) {
            val ret = JSObject()
            ret.put("files", JSArray())
            call.resolve(ret)
            return
        }

        val data = result.data
        val uris = mutableListOf<Uri>()

        if (data != null) {
            val clipData = data.clipData
            if (clipData != null) {
                for (i in 0 until clipData.itemCount) {
                    clipData.getItemAt(i).uri?.let { uris.add(it) }
                }
            } else {
                data.data?.let { uris.add(it) }
            }
        }

        val files = JSArray()
        for (uri in uris) {
            try {
                context.contentResolver.takePersistableUriPermission(
                    uri,
                    Intent.FLAG_GRANT_READ_URI_PERMISSION,
                )
            } catch (_: Exception) {
                AnyTTYDebugLog.event(AnyTTYDebugEvent.PERSISTABLE_PERMISSION_FAILED)
            }
            val file = queryFileInfo(uri)
            if (file != null) {
                files.put(file)
            }
        }

        val ret = JSObject()
        ret.put("files", files)
        call.resolve(ret)
    }

    private fun queryFileInfo(uri: Uri): JSObject? {
        var name = "unknown"
        var size = 0L
        var mimeType = "application/octet-stream"

        try {
            val cursor: Cursor? = context.contentResolver.query(uri, null, null, null, null)
            cursor?.use {
                if (it.moveToFirst()) {
                    val nameIdx = it.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                    if (nameIdx >= 0) {
                        name = it.getString(nameIdx) ?: name
                    }
                    val sizeIdx = it.getColumnIndex(OpenableColumns.SIZE)
                    if (sizeIdx >= 0) {
                        size = it.getLong(sizeIdx)
                    }
                }
            }
            context.contentResolver.getType(uri)?.let { mimeType = it }
        } catch (_: Exception) {
            AnyTTYDebugLog.event(AnyTTYDebugEvent.FILE_INFO_FAILED)
        }

        return JSObject().apply {
            put("uri", uri.toString())
            put("name", name)
            put("size", size)
            put("mimeType", mimeType)
        }
    }
}
