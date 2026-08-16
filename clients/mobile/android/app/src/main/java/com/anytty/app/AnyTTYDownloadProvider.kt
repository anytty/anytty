package com.anytty.app

import android.content.ContentProvider
import android.content.ContentResolver
import android.content.ContentValues
import android.content.Context
import android.database.Cursor
import android.database.MatrixCursor
import android.net.Uri
import android.os.Environment
import android.os.ParcelFileDescriptor
import android.provider.OpenableColumns
import android.webkit.MimeTypeMap
import java.io.File
import java.io.FileNotFoundException

/** Grants read-only access to completed legacy downloads and nothing else. */
class AnyTTYDownloadProvider : ContentProvider() {
    override fun onCreate(): Boolean = true

    override fun getType(uri: Uri): String {
        val extension = MimeTypeMap.getFileExtensionFromUrl(resolve(uri).name).lowercase()
        return MimeTypeMap.getSingleton().getMimeTypeFromExtension(extension) ?: "application/octet-stream"
    }

    override fun query(
        uri: Uri,
        projection: Array<out String>?,
        selection: String?,
        selectionArgs: Array<out String>?,
        sortOrder: String?,
    ): Cursor {
        val file = resolve(uri)
        val columns = projection ?: arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE)
        return MatrixCursor(columns).apply {
            addRow(columns.map { column ->
                when (column) {
                    OpenableColumns.DISPLAY_NAME -> file.name
                    OpenableColumns.SIZE -> file.length()
                    else -> null
                }
            })
        }
    }

    override fun openFile(uri: Uri, mode: String): ParcelFileDescriptor {
        if (mode != "r") throw FileNotFoundException("download is read-only")
        return ParcelFileDescriptor.open(resolve(uri), ParcelFileDescriptor.MODE_READ_ONLY)
    }

    override fun insert(uri: Uri, values: ContentValues?): Uri? = throw UnsupportedOperationException("read-only provider")
    override fun update(uri: Uri, values: ContentValues?, selection: String?, selectionArgs: Array<out String>?): Int = 0
    override fun delete(uri: Uri, selection: String?, selectionArgs: Array<out String>?): Int = 0

    private fun resolve(uri: Uri): File {
        val currentContext = context ?: throw FileNotFoundException("download provider is unavailable")
        if (uri.scheme != ContentResolver.SCHEME_CONTENT || uri.authority != authority(currentContext)) {
            throw FileNotFoundException("download URI is invalid")
        }
        val segments = uri.pathSegments
        if (segments.size != 2 || segments[0] != "file") throw FileNotFoundException("download URI is invalid")
        return validatedFile(currentContext, File(downloadRoot(currentContext), segments[1]))
    }

    companion object {
        fun uriForFile(context: Context, file: File): Uri {
            val validated = validatedFile(context, file)
            return Uri.Builder()
                .scheme(ContentResolver.SCHEME_CONTENT)
                .authority(authority(context))
                .appendPath("file")
                .appendPath(validated.name)
                .build()
        }

        private fun authority(context: Context): String = "${context.packageName}.downloads"

        private fun downloadRoot(context: Context): File =
            File(context.getExternalFilesDir(Environment.DIRECTORY_DOWNLOADS), "AnyTTY").canonicalFile

        private fun validatedFile(context: Context, file: File): File {
            val root = downloadRoot(context)
            val candidate = file.canonicalFile
            if (!candidate.path.startsWith(root.path + File.separator) || !candidate.isFile) {
                throw FileNotFoundException("download is outside the app download directory")
            }
            return candidate
        }
    }
}
