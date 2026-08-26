package com.anytty.app

import java.io.File
import java.nio.file.Files
import java.util.zip.ZipFile
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertTrue
import org.junit.Test

class AnyTTYDiagnosticStoreTest {
    private val temporaryDirectories = mutableListOf<File>()

    @After
    fun cleanUp() {
        temporaryDirectories.forEach { it.deleteRecursively() }
    }

    @Test
    fun rollingFilesStayBoundedAndRetainNewestGenerations() {
        val directory = temporaryDirectory()
        val files = RollingDiagnosticFiles(directory, "events.log", maxFileBytes = 8, retainedFiles = 3)

        files.append("aaaa".toByteArray())
        files.append("bbbb".toByteArray())
        files.append("cc".toByteArray())
        files.append("dddddd".toByteArray())
        files.append("x".toByteArray())

        assertEquals("x", File(directory, "events.log").readText())
        assertEquals("ccdddddd", File(directory, "events.log.1").readText())
        assertEquals("aaaabbbb", File(directory, "events.log.2").readText())
        assertFalse(File(directory, "events.log.3").exists())
        assertTrue(directory.listFiles().orEmpty().all { it.length() <= 8L })
    }

    @Test
    fun sanitizerRemovesControlCharactersAndCapsValues() {
        val sanitized = AnyTTYDiagnosticStore.sanitize("phase=ready\nreason:path/changed" + "x".repeat(600))

        assertEquals(512, sanitized.length)
        assertFalse(sanitized.contains('\n'))
        assertFalse(sanitized.contains(':'))
        assertFalse(sanitized.contains('/'))
        assertTrue(sanitized.startsWith("phase=ready_reason_path_changed"))
    }

    @Test
    fun diagnosticZipContainsManifestAndLogEntries() {
        val directory = temporaryDirectory()
        val source = File(directory, "app-events.log").apply { writeText("event=foreground_resume_done\n") }
        val target = File(directory, "diagnostics.zip")

        writeDiagnosticZip(target, "format=anytty-diagnostics-v1\nautomatic_upload=false\n", listOf(source))

        ZipFile(target).use { zip ->
            assertNotNull(zip.getEntry("manifest.txt"))
            assertNotNull(zip.getEntry("logs/app-events.log"))
            val manifest = zip.getInputStream(zip.getEntry("manifest.txt")).bufferedReader().readText()
            assertTrue(manifest.contains("automatic_upload=false"))
        }
    }

    private fun temporaryDirectory(): File =
        Files.createTempDirectory("anytty-diagnostics-test").toFile().also(temporaryDirectories::add)
}
