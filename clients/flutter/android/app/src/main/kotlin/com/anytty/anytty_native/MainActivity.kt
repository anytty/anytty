package com.anytty.app

import android.app.Activity
import android.content.Context
import android.content.ClipData
import android.content.Intent
import android.os.Bundle
import android.net.Uri
import android.webkit.MimeTypeMap
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.embedding.engine.FlutterEngineCache
import io.flutter.embedding.engine.dart.DartExecutor
import io.flutter.embedding.android.FlutterActivity
import io.flutter.plugin.common.MethodCall
import io.flutter.plugin.common.MethodChannel
import java.io.File
import java.io.FileInputStream

class MainActivity : FlutterActivity() {
    private data class PendingExport(
        val source: File,
        val result: MethodChannel.Result,
    )

    private var pendingExport: PendingExport? = null
    private var fileChannel: MethodChannel? = null
    private var externalUriChannel: MethodChannel? = null
    private var sshCredentialChannel: MethodChannel? = null
    private var localDiscoveryChannel: MethodChannel? = null
    private var imeInsetsBridge: AndroidImeInsetsBridge? = null
    private var terminalInputBridge: AndroidTerminalInputBridge? = null
    private val sshCredentials = AndroidSSHCredentialStore()
    private val localDiscovery by lazy { AndroidLocalDiscoveryBridge(applicationContext) }

    override fun provideFlutterEngine(context: Context): FlutterEngine =
        retainedFlutterEngine(context)

    override fun shouldDestroyEngineWithHost(): Boolean = false

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AnyttyBackgroundBridge.attachActivity(this)
        AnyttyBackgroundBridge.deliverRoute(intent.getStringExtra(AnyttyBackgroundBridge.EXTRA_ROUTE))
    }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        fileChannel = MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            FILE_CHANNEL,
        ).also { channel -> channel.setMethodCallHandler(
            ::handleFileMethod,
        ) }
        externalUriChannel = MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            EXTERNAL_URI_CHANNEL,
        ).also { channel -> channel.setMethodCallHandler(
            ::handleExternalUriMethod,
        ) }
        sshCredentialChannel = MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            SSH_CREDENTIAL_CHANNEL,
        ).also { channel -> channel.setMethodCallHandler(sshCredentials::handle) }
        localDiscoveryChannel = MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            LOCAL_DISCOVERY_CHANNEL,
        ).also { channel -> channel.setMethodCallHandler(localDiscovery::handle) }
        imeInsetsBridge?.close()
        imeInsetsBridge = AndroidImeInsetsBridge(
            window.decorView,
            flutterEngine.dartExecutor.binaryMessenger,
        )
        terminalInputBridge?.close()
        terminalInputBridge = AndroidTerminalInputBridge(
            this,
            flutterEngine.dartExecutor.binaryMessenger,
        )
        AnyttyBackgroundBridge.attach(applicationContext, flutterEngine.dartExecutor.binaryMessenger)
        AnyttyBackgroundBridge.attachActivity(this)
    }

    override fun cleanUpFlutterEngine(flutterEngine: FlutterEngine) {
        fileChannel?.setMethodCallHandler(null)
        fileChannel = null
        externalUriChannel?.setMethodCallHandler(null)
        externalUriChannel = null
        sshCredentialChannel?.setMethodCallHandler(null)
        sshCredentialChannel = null
        localDiscoveryChannel?.setMethodCallHandler(null)
        localDiscoveryChannel = null
        imeInsetsBridge?.close()
        imeInsetsBridge = null
        terminalInputBridge?.close()
        terminalInputBridge = null
        localDiscovery.close()
        AnyttyBackgroundBridge.detachActivity(this)
        super.cleanUpFlutterEngine(flutterEngine)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        AnyttyBackgroundBridge.deliverRoute(intent.getStringExtra(AnyttyBackgroundBridge.EXTRA_ROUTE))
    }

    override fun onRequestPermissionsResult(
        requestCode: Int,
        permissions: Array<out String>,
        grantResults: IntArray,
    ) {
        super.onRequestPermissionsResult(requestCode, permissions, grantResults)
        AnyttyBackgroundBridge.onRequestPermissionsResult(requestCode, permissions, grantResults)
    }

    private fun handleFileMethod(call: MethodCall, result: MethodChannel.Result) {
        if (call.method == "openFile") {
            openFile(call, result)
            return
        }
        if (call.method != "exportFile") {
            result.notImplemented()
            return
        }
        if (pendingExport != null) {
            result.error("export_busy", "Another file export is active", null)
            return
        }
        val sourcePath = call.argument<String>("sourcePath")
        val fileName = call.argument<String>("fileName")
        val mimeType = call.argument<String>("mimeType") ?: DEFAULT_MIME_TYPE
        if (sourcePath.isNullOrBlank() || fileName.isNullOrBlank()) {
            result.error("invalid_export", "Missing export source or file name", null)
            return
        }
        val source = try {
            File(sourcePath).canonicalFile
        } catch (error: Exception) {
            result.error("invalid_export", "Invalid export source", error.message)
            return
        }
        if (!source.isFile || !isAppTemporaryFile(source)) {
            result.error("invalid_export", "Export source is outside app storage", null)
            return
        }

        pendingExport = PendingExport(source, result)
        val intent = Intent(Intent.ACTION_CREATE_DOCUMENT).apply {
            addCategory(Intent.CATEGORY_OPENABLE)
            type = mimeType
            putExtra(Intent.EXTRA_TITLE, fileName)
        }
        try {
            startActivityForResult(intent, EXPORT_REQUEST_CODE)
        } catch (error: Exception) {
            pendingExport = null
            result.error("export_failed", "Unable to open the save dialog", error.message)
        }
    }

    private fun handleExternalUriMethod(call: MethodCall, result: MethodChannel.Result) {
        if (call.method != "openExternalUri") {
            result.notImplemented()
            return
        }
        try {
            val uri = Uri.parse(call.argument<String>("uri").orEmpty())
            val scheme = uri.scheme?.lowercase().orEmpty()
            require(scheme in EXTERNAL_URI_SCHEMES) { "External URI scheme is not allowed" }
            val intent = Intent(Intent.ACTION_VIEW, uri).apply {
                addCategory(Intent.CATEGORY_BROWSABLE)
            }
            startActivity(intent)
            result.success(null)
        } catch (error: Exception) {
            result.error("open_external_failed", "Unable to open the external URI", error.message)
        }
    }

    @Deprecated("Uses the Activity result callback supported by FlutterActivity")
    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        super.onActivityResult(requestCode, resultCode, data)
        if (requestCode != EXPORT_REQUEST_CODE) return
        val pending = pendingExport ?: return
        val destination = data?.data
        if (resultCode != Activity.RESULT_OK || destination == null) {
            pendingExport = null
            pending.result.success(null)
            return
        }
        copyExport(pending, destination)
    }

    private fun openFile(call: MethodCall, result: MethodChannel.Result) {
        try {
            val rawUri = call.argument<String>("uri").orEmpty()
            val fileName = call.argument<String>("fileName").orEmpty()
            val requestedMimeType = call.argument<String>("mimeType").orEmpty()
            val uri = Uri.parse(rawUri)
            require(uri.scheme == "content") { "Download URI is invalid" }
            val extension = MimeTypeMap.getFileExtensionFromUrl(fileName).lowercase()
            val mimeType = contentResolver.getType(uri)
                ?: requestedMimeType.takeIf { it.isNotBlank() }
                ?: MimeTypeMap.getSingleton().getMimeTypeFromExtension(extension)
                ?: DEFAULT_MIME_TYPE
            val viewIntent = Intent(Intent.ACTION_VIEW).apply {
                setDataAndType(uri, mimeType)
                clipData = ClipData.newRawUri(fileName.ifBlank { "download" }, uri)
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
            val chooser = Intent.createChooser(viewIntent, null).apply {
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
            startActivity(chooser)
            result.success(null)
        } catch (error: Exception) {
            result.error("open_failed", "Unable to open the file", error.message)
        }
    }

    private fun copyExport(pending: PendingExport, destination: Uri) {
        Thread {
            try {
                FileInputStream(pending.source).use { input ->
                    val output = contentResolver.openOutputStream(destination, "w")
                        ?: throw IllegalStateException("Unable to open the selected destination")
                    output.use { input.copyTo(it) }
                }
                runOnUiThread {
                    pendingExport = null
                    pending.result.success(destination.toString())
                }
            } catch (error: Exception) {
                runOnUiThread {
                    pendingExport = null
                    pending.result.error("export_failed", "Unable to save the file", error.message)
                }
            }
        }.start()
    }

    private fun isAppTemporaryFile(source: File): Boolean {
        val roots = listOfNotNull(cacheDir, filesDir, externalCacheDir, getExternalFilesDir(null))
        return roots.any { root ->
            val rootPath = root.canonicalFile.path + File.separator
            source.path.startsWith(rootPath)
        }
    }

    companion object {
        private const val FILE_CHANNEL = "com.anytty.app/files"
        private const val EXTERNAL_URI_CHANNEL = "com.anytty.app/external-uri"
        private const val SSH_CREDENTIAL_CHANNEL = "com.anytty.app/ssh-credentials"
        private const val LOCAL_DISCOVERY_CHANNEL = "com.anytty.app/local-discovery"
        private const val ENGINE_ID = "anytty-retained-engine"
        private const val DEFAULT_MIME_TYPE = "application/octet-stream"
        private const val EXPORT_REQUEST_CODE = 7001
        private val EXTERNAL_URI_SCHEMES = setOf("http", "https", "mailto", "tel")

        @Synchronized
        private fun retainedFlutterEngine(context: Context): FlutterEngine {
            FlutterEngineCache.getInstance().get(ENGINE_ID)?.let { return it }
            return FlutterEngine(context.applicationContext).also { engine ->
                AnyttyBackgroundBridge.attach(
                    context.applicationContext,
                    engine.dartExecutor.binaryMessenger,
                )
                engine.navigationChannel.setInitialRoute("/")
                engine.dartExecutor.executeDartEntrypoint(
                    DartExecutor.DartEntrypoint.createDefault(),
                )
                FlutterEngineCache.getInstance().put(ENGINE_ID, engine)
            }
        }
    }
}
