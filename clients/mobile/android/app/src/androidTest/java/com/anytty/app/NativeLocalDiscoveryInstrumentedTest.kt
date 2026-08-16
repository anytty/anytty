package com.anytty.app

import android.content.Context
import android.net.nsd.NsdManager
import android.net.nsd.NsdServiceInfo
import androidx.test.core.app.ApplicationProvider
import androidx.test.ext.junit.runners.AndroidJUnit4
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test
import org.junit.runner.RunWith
import java.net.ServerSocket
import java.util.concurrent.CountDownLatch
import java.util.concurrent.TimeUnit
import java.util.concurrent.atomic.AtomicReference

@RunWith(AndroidJUnit4::class)
class NativeLocalDiscoveryInstrumentedTest {
    @Test
    fun discoversAndResolvesAnAnyTTYServiceInsideAndroid() {
        val context = ApplicationProvider.getApplicationContext<Context>()
        val manager = context.getSystemService(Context.NSD_SERVICE) as NsdManager
        val deviceId = "device-android-nsd"
        val fingerprint = "ed25519-sha256:android-nsd"
        val registered = CountDownLatch(1)
        val registrationFailure = AtomicReference<String>()
        val listener = object : NsdManager.RegistrationListener {
            override fun onServiceRegistered(serviceInfo: NsdServiceInfo) = registered.countDown()
            override fun onRegistrationFailed(serviceInfo: NsdServiceInfo, errorCode: Int) {
                registrationFailure.set("registration failed: $errorCode")
                registered.countDown()
            }
            override fun onServiceUnregistered(serviceInfo: NsdServiceInfo) = Unit
            override fun onUnregistrationFailed(serviceInfo: NsdServiceInfo, errorCode: Int) = Unit
        }

        ServerSocket(0).use { socket ->
            val service = NsdServiceInfo().apply {
                serviceName = "anytty-instrumented-${System.nanoTime()}"
                serviceType = "_anytty._tcp."
                port = socket.localPort
                setAttribute("v", "1")
                setAttribute("p", "7")
                setAttribute("k", nativeLocalDiscoveryKey(deviceId, fingerprint))
            }
            manager.registerService(service, NsdManager.PROTOCOL_DNS_SD, listener)
            try {
                assertTrue("NSD registration timed out", registered.await(10, TimeUnit.SECONDS))
                assertEquals(null, registrationFailure.get())

                NativeLocalDiscovery(context).use { discovery ->
                    discovery.restart(true)
                    val deadline = System.nanoTime() + TimeUnit.SECONDS.toNanos(20)
                    var candidates = emptyList<NativeLocalDiscoveryCandidate>()
                    while (System.nanoTime() < deadline && candidates.isEmpty()) {
                        Thread.sleep(100)
                        candidates = NativeLocalDiscoveryCache.snapshot(deviceId, fingerprint)
                    }
                    assertTrue("registered AnyTTY service was not discovered", candidates.isNotEmpty())
                    assertTrue(candidates.any { it.port == socket.localPort && it.protocolVersion == 7 })
                }
            } finally {
                runCatching { manager.unregisterService(listener) }
                NativeLocalDiscoveryCache.clear()
            }
        }
    }
}
