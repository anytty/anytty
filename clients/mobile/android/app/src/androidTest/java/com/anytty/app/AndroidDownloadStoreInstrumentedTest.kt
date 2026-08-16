package com.anytty.app

import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertArrayEquals
import org.junit.Assert.assertEquals
import org.junit.Test
import org.junit.runner.RunWith
import java.security.MessageDigest

@RunWith(AndroidJUnit4::class)
class AndroidDownloadStoreInstrumentedTest {
    @Test
    fun savePersistsExactContentAndReportsDigest() {
        val context = ApplicationProvider.getApplicationContext<android.content.Context>()
        val content = ByteArray(128 * 1024) { index -> (index % 251).toByte() }
        val saved = AndroidDownloadStore(context).save("rtc010-download-store.bin", "application/octet-stream", content)
        val persisted = context.contentResolver.openInputStream(saved.uri)!!.use { it.readBytes() }
        val digest = MessageDigest.getInstance("SHA-256").digest(content).joinToString("") { "%02x".format(it) }

        assertArrayEquals(content, persisted)
        assertEquals(content.size.toLong(), saved.bytes)
        assertEquals(digest, saved.sha256)
        context.contentResolver.delete(saved.uri, null, null)
    }

    @Test
    fun partialDownloadSurvivesStoreRecreationAndCommitsExactContent() {
        val context = ApplicationProvider.getApplicationContext<android.content.Context>()
        val content = ByteArray(320 * 1024) { index -> (index % 239).toByte() }
        val split = 128 * 1024
        val machineId = "instrumented-machine"
        val remotePath = "/tmp/rtc010-resume.bin"
        val first = AndroidDownloadStore(context)
        first.discardPartial(machineId, remotePath, content.size.toLong())
        assertEquals(split.toLong(), first.appendPartial(machineId, remotePath, content.size.toLong(), 0, content.copyOfRange(0, split)))

        val restored = AndroidDownloadStore(context)
        assertEquals(split.toLong(), restored.resumeOffset(machineId, remotePath, content.size.toLong()))
        assertEquals(content.size.toLong(), restored.appendPartial(machineId, remotePath, content.size.toLong(), split.toLong(), content.copyOfRange(split, content.size)))
        val digest = MessageDigest.getInstance("SHA-256").digest(content)
        val saved = restored.commitPartial(
            "rtc010-resumed-download.bin", "application/octet-stream", machineId, remotePath, content.size.toLong(), digest,
        )
        val persisted = context.contentResolver.openInputStream(saved.uri)!!.use { it.readBytes() }

        assertArrayEquals(content, persisted)
        assertEquals(0L, restored.resumeOffset(machineId, remotePath, content.size.toLong()))
        context.contentResolver.delete(saved.uri, null, null)
    }
}
