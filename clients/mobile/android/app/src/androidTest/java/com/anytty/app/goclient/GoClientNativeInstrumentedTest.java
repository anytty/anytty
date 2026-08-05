package com.anytty.app.goclient;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.assertTrue;

import androidx.test.core.app.ApplicationProvider;
import androidx.test.ext.junit.runners.AndroidJUnit4;

import com.anytty.app.goclient.GoClientNative;

import org.junit.Test;
import org.junit.runner.RunWith;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;

import anytty.client.binding.v1.ClientBinding;
import anytty.remote.auth.v1.RemoteAuth;

/** GoClientNativeInstrumentedTest 在 APK 进程中验证生产 Go Client ABI、平台请求与持久化行为。 */
@RunWith(AndroidJUnit4.class)
public final class GoClientNativeInstrumentedTest {
    @Test
    public void acceptsCallsFromIndependentJvmThreads() throws Exception {
        List<Callable<Integer>> calls = new ArrayList<>();
        for (int i = 0; i < 16; i++) {
            calls.add(() -> GoClientNative.INSTANCE.abiVersion());
        }
        ExecutorService executor = Executors.newFixedThreadPool(4);
        try {
            for (Future<Integer> result : executor.invokeAll(calls)) {
                assertEquals(3, result.get().intValue());
            }
        } finally {
            executor.shutdownNow();
        }
    }

    @Test
    public void productionEngineExchangesPlatformProtoRequests() throws Exception {
        long engine = GoClientNative.INSTANCE.create();
        try {
            long operation = GoClientNative.INSTANCE.engineCommand(engine,
                    ClientBinding.EngineCommand.newBuilder()
                            .setDeleteCredential(ClientBinding.DeleteCredentialRequest.newBuilder()
                                    .setRequestId("delete-platform")
                                    .setCredentialRef("credential:studio"))
                            .build().toByteArray());
            ClientBinding.PlatformRequest request = ClientBinding.PlatformRequest.parseFrom(
                    GoClientNative.INSTANCE.nextPlatformRequest(engine, 5_000));
            assertTrue(request.hasCredentialDelete());
            assertEquals("credential:studio", request.getCredentialDelete().getCredentialRef());
            GoClientNative.INSTANCE.completePlatformRequest(engine,
                    ClientBinding.PlatformResponse.newBuilder()
                            .setRequestId(request.getRequestId())
                            .build().toByteArray());
            ClientBinding.EventEnvelope event = next(engine);
            assertTrue(event.hasDeleteCredential());
            assertEquals(operation, event.getDeleteCredential().getOperationHandle());
            GoClientNative.INSTANCE.release(engine, operation);
        } finally {
            GoClientNative.INSTANCE.close(engine);
        }
    }

    @Test
    public void endpointRegistrySurvivesProductionEngineRecreation() throws Exception {
        String endpointId = "android-registry-test";
        AndroidGoClientEngine first = new AndroidGoClientEngine(ApplicationProvider.getApplicationContext());
        try {
            long operation = GoClientNative.INSTANCE.engineCommand(first.getHandle(),
                    ClientBinding.EngineCommand.newBuilder()
                            .setEndpointUpsert(ClientBinding.EndpointUpsertRequest.newBuilder()
                                    .setRequestId("registry-upsert")
                                    .setEndpoint(testEndpoint(endpointId)))
                            .build().toByteArray());
            ClientBinding.EventEnvelope event = next(first.getHandle());
            assertTrue(event.toString(), event.hasEndpointUpsert());
            assertEquals(operation, event.getEndpointUpsert().getOperationHandle());
            assertEquals(endpointId, event.getEndpointUpsert().getEndpoint().getEndpointId());
            GoClientNative.INSTANCE.release(first.getHandle(), operation);
        } finally {
            first.close();
        }

        AndroidGoClientEngine second = new AndroidGoClientEngine(ApplicationProvider.getApplicationContext());
        try {
            long operation = GoClientNative.INSTANCE.engineCommand(second.getHandle(),
                    ClientBinding.EngineCommand.newBuilder()
                            .setEndpointRegistryGet(ClientBinding.EndpointRegistryGetRequest.newBuilder()
                                    .setRequestId("registry-get"))
                            .build().toByteArray());
            ClientBinding.EventEnvelope event = next(second.getHandle());
            assertTrue(event.toString(), event.hasEndpointRegistryGet());
            assertTrue(event.getEndpointRegistryGet().getRegistry().getEndpointsList().stream()
                    .anyMatch(endpoint -> endpointId.equals(endpoint.getEndpointId())));
            GoClientNative.INSTANCE.release(second.getHandle(), operation);

            long deleteOperation = GoClientNative.INSTANCE.engineCommand(second.getHandle(),
                    ClientBinding.EngineCommand.newBuilder()
                            .setEndpointDelete(ClientBinding.EndpointDeleteRequest.newBuilder()
                                    .setRequestId("registry-delete")
                                    .setEndpointId(endpointId))
                            .build().toByteArray());
            ClientBinding.EventEnvelope deleted = next(second.getHandle());
            assertTrue(deleted.toString(), deleted.hasEndpointDelete());
            assertEquals(deleteOperation, deleted.getEndpointDelete().getOperationHandle());
            GoClientNative.INSTANCE.release(second.getHandle(), deleteOperation);
        } finally {
            second.close();
        }
    }

    private static RemoteAuth.EndpointConfigV1 testEndpoint(String endpointId) {
        RemoteAuth.EndpointRouteConfigV1 route = RemoteAuth.EndpointRouteConfigV1.newBuilder()
                .setSchemaVersion(1)
                .setRouteId("direct")
                .setEnabled(true)
                .setCredentialRef("android-registry-test")
                .setSource(RemoteAuth.EndpointSource.ENDPOINT_SOURCE_BOOTSTRAP)
                .setPolicySource(RemoteAuth.EndpointSource.ENDPOINT_SOURCE_BOOTSTRAP)
                .setDirectWebrtcTcp(RemoteAuth.DirectWebRTCTCPRouteConfig.newBuilder()
                        .addSignalingAddresses("127.0.0.1:41120")
                        .addIceTcpAddresses("127.0.0.1:41121"))
                .build();
        return RemoteAuth.EndpointConfigV1.newBuilder()
                .setSchemaVersion(1)
                .setEndpointId(endpointId)
                .setLabel("Android registry test")
                .setLabelSource(RemoteAuth.EndpointSource.ENDPOINT_SOURCE_USER)
                .setIdentity(RemoteAuth.EndpointDaemonIdentity.newBuilder()
                        .setDeviceId("daemon-android-registry-test")
                        .setDeviceFingerprint("ed25519-sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"))
                .setConnectMode(RemoteAuth.EndpointConnectMode.ENDPOINT_CONNECT_MODE_ON_DEMAND)
                .setEnabled(true)
                .addRoutes(route)
                .build();
    }

    private static ClientBinding.EventEnvelope next(long engine) throws Exception {
        return ClientBinding.EventEnvelope.parseFrom(GoClientNative.INSTANCE.nextEvent(engine, 15_000));
    }
}
