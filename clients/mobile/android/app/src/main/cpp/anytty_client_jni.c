#include <jni.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#include "anytty_client.h"

extern void anytty_debug_log_set_path(char *path);

static int throw_status(JNIEnv *env, anytty_status_v1 status) {
  if (status == ANYTTY_STATUS_OK) {
    return 0;
  }
  jclass exception = (*env)->FindClass(env, "java/lang/IllegalStateException");
  if (exception != NULL) {
    char message[64];
    snprintf(message, sizeof(message), "anytty native status %d", (int)status);
    (*env)->ThrowNew(env, exception, message);
  }
  return -1;
}

static jbyte *borrow_payload(JNIEnv *env, jbyteArray payload, jsize *length) {
  if (payload == NULL) {
    throw_status(env, ANYTTY_STATUS_INVALID_ARGUMENT);
    return NULL;
  }
  *length = (*env)->GetArrayLength(env, payload);
  return (*env)->GetByteArrayElements(env, payload, NULL);
}

JNIEXPORT jint JNICALL
Java_com_anytty_app_goclient_GoClientNative_abiVersion(JNIEnv *env, jobject self) {
  (void)env;
  (void)self;
  return (jint)anytty_client_abi_version();
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_setDebugLogPath(JNIEnv *env, jobject self, jstring path) {
  (void)self;
  const char *value = path == NULL ? "" : (*env)->GetStringUTFChars(env, path, NULL);
  if (value == NULL) return;
  anytty_debug_log_set_path((char *)value);
  if (path != NULL) (*env)->ReleaseStringUTFChars(env, path, value);
}

JNIEXPORT jlong JNICALL
Java_com_anytty_app_goclient_GoClientNative_create(JNIEnv *env, jobject self) {
  (void)self;
  anytty_handle_t engine = 0;
  if (throw_status(env, anytty_engine_create(&engine)) != 0) {
    return 0;
  }
  return (jlong)engine;
}

JNIEXPORT jint JNICALL
Java_com_anytty_app_goclient_GoClientNative_startBridge(JNIEnv *env, jobject self, jlong engine, jstring token) {
  (void)self;
  if (token == NULL) {
    throw_status(env, ANYTTY_STATUS_INVALID_ARGUMENT);
    return 0;
  }
  const char *value = (*env)->GetStringUTFChars(env, token, NULL);
  if (value == NULL) {
    return 0;
  }
  jsize length = (*env)->GetStringUTFLength(env, token);
  uint16_t port = 0;
  anytty_status_v1 status = anytty_bridge_start(
      (anytty_handle_t)engine, (const uint8_t *)value, (size_t)length, &port);
  (*env)->ReleaseStringUTFChars(env, token, value);
  return throw_status(env, status) == 0 ? (jint)port : 0;
}

JNIEXPORT jboolean JNICALL
Java_com_anytty_app_goclient_GoClientNative_localProbe(JNIEnv *env, jobject self, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return JNI_FALSE;
  }
  uint8_t reachable = 0;
  anytty_status_v1 status = anytty_local_probe(
      (const uint8_t *)bytes, (size_t)length, &reachable);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  return throw_status(env, status) == 0 && reachable != 0 ? JNI_TRUE : JNI_FALSE;
}

JNIEXPORT jboolean JNICALL
Java_com_anytty_app_goclient_GoClientNative_directProbe(JNIEnv *env, jobject self, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return JNI_FALSE;
  }
  uint8_t reachable = 0;
  anytty_status_v1 status = anytty_direct_probe(
      (const uint8_t *)bytes, (size_t)length, &reachable);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  return throw_status(env, status) == 0 && reachable != 0 ? JNI_TRUE : JNI_FALSE;
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_replaceSupervisorDemand(JNIEnv *env, jobject self, jlong engine, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) return;
  anytty_status_v1 status = anytty_supervisor_replace_demand(
      (anytty_handle_t)engine, (const uint8_t *)bytes, (size_t)length);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  throw_status(env, status);
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_signalSupervisor(JNIEnv *env, jobject self, jlong engine, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) return;
  anytty_status_v1 status = anytty_supervisor_signal(
      (anytty_handle_t)engine, (const uint8_t *)bytes, (size_t)length);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  throw_status(env, status);
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_repairSupervisorEndpoint(JNIEnv *env, jobject self, jlong engine, jbyteArray endpoint_id) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, endpoint_id, &length);
  if (bytes == NULL) return;
  anytty_status_v1 status = anytty_supervisor_repair(
      (anytty_handle_t)engine, (const uint8_t *)bytes, (size_t)length);
  (*env)->ReleaseByteArrayElements(env, endpoint_id, bytes, JNI_ABORT);
  throw_status(env, status);
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_awaitSupervisorReady(JNIEnv *env, jobject self, jlong engine, jint timeout_millis) {
  (void)self;
  throw_status(env, anytty_supervisor_wait_ready((anytty_handle_t)engine, (uint32_t)timeout_millis));
}

JNIEXPORT jbyteArray JNICALL
Java_com_anytty_app_goclient_GoClientNative_supervisorSnapshot(JNIEnv *env, jobject self, jlong engine) {
  (void)self;
  anytty_buffer_v1 snapshot = {0};
  if (throw_status(env, anytty_supervisor_snapshot((anytty_handle_t)engine, &snapshot)) != 0) {
    return NULL;
  }
  jbyteArray result = (*env)->NewByteArray(env, (jsize)snapshot.length);
  if (result != NULL && snapshot.length > 0) {
    (*env)->SetByteArrayRegion(env, result, 0, (jsize)snapshot.length, (const jbyte *)snapshot.data);
  }
  anytty_status_v1 free_status = anytty_buffer_free(snapshot.buffer_handle);
  if (result != NULL && throw_status(env, free_status) != 0) {
    return NULL;
  }
  return result;
}

JNIEXPORT jlong JNICALL
Java_com_anytty_app_goclient_GoClientNative_openSession(JNIEnv *env, jobject self, jlong engine, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return 0;
  }
  anytty_handle_t operation = 0;
  anytty_status_v1 status = anytty_engine_open_session(
      (anytty_handle_t)engine, (const uint8_t *)bytes, (size_t)length, &operation);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  return throw_status(env, status) == 0 ? (jlong)operation : 0;
}

JNIEXPORT jlong JNICALL
Java_com_anytty_app_goclient_GoClientNative_execute(JNIEnv *env, jobject self, jlong engine, jlong session, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return 0;
  }
  anytty_handle_t operation = 0;
  anytty_status_v1 status = anytty_engine_execute(
      (anytty_handle_t)engine, (anytty_handle_t)session,
      (const uint8_t *)bytes, (size_t)length, &operation);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  return throw_status(env, status) == 0 ? (jlong)operation : 0;
}

JNIEXPORT jlong JNICALL
Java_com_anytty_app_goclient_GoClientNative_openResourceStream(JNIEnv *env, jobject self, jlong engine, jlong session, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return 0;
  }
  anytty_handle_t stream = 0;
  anytty_status_v1 status = anytty_engine_open_resource_stream(
      (anytty_handle_t)engine, (anytty_handle_t)session,
      (const uint8_t *)bytes, (size_t)length, &stream);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  return throw_status(env, status) == 0 ? (jlong)stream : 0;
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_sendResourceStreamFrame(JNIEnv *env, jobject self, jlong engine, jlong stream, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return;
  }
  anytty_status_v1 status = anytty_engine_send_resource_stream_frame(
      (anytty_handle_t)engine, (anytty_handle_t)stream,
      (const uint8_t *)bytes, (size_t)length);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  throw_status(env, status);
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_closeResourceStream(JNIEnv *env, jobject self, jlong engine, jlong stream) {
  (void)self;
  throw_status(env, anytty_engine_close_resource_stream((anytty_handle_t)engine, (anytty_handle_t)stream));
}

JNIEXPORT jlong JNICALL
Java_com_anytty_app_goclient_GoClientNative_engineCommand(JNIEnv *env, jobject self, jlong engine, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return 0;
  }
  anytty_handle_t operation = 0;
  anytty_status_v1 status = anytty_engine_command(
      (anytty_handle_t)engine, (const uint8_t *)bytes, (size_t)length, &operation);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  return throw_status(env, status) == 0 ? (jlong)operation : 0;
}

JNIEXPORT jbyteArray JNICALL
Java_com_anytty_app_goclient_GoClientNative_nextEvent(JNIEnv *env, jobject self, jlong engine, jint timeout_millis) {
  (void)self;
  anytty_buffer_v1 event = {0};
  if (throw_status(env, anytty_engine_next_event((anytty_handle_t)engine, (uint32_t)timeout_millis, &event)) != 0) {
    return NULL;
  }
  jbyteArray result = (*env)->NewByteArray(env, (jsize)event.length);
  if (result != NULL && event.length > 0) {
    (*env)->SetByteArrayRegion(env, result, 0, (jsize)event.length, (const jbyte *)event.data);
  }
  anytty_status_v1 free_status = anytty_buffer_free(event.buffer_handle);
  if (result != NULL && throw_status(env, free_status) != 0) {
    return NULL;
  }
  return result;
}

JNIEXPORT jbyteArray JNICALL
Java_com_anytty_app_goclient_GoClientNative_nextPlatformRequest(JNIEnv *env, jobject self, jlong engine, jint timeout_millis) {
  (void)self;
  anytty_buffer_v1 request = {0};
  if (throw_status(env, anytty_platform_next_request((anytty_handle_t)engine, (uint32_t)timeout_millis, &request)) != 0) {
    return NULL;
  }
  jbyteArray result = (*env)->NewByteArray(env, (jsize)request.length);
  if (result != NULL && request.length > 0) {
    (*env)->SetByteArrayRegion(env, result, 0, (jsize)request.length, (const jbyte *)request.data);
  }
  anytty_status_v1 free_status = anytty_buffer_free(request.buffer_handle);
  if (result != NULL && throw_status(env, free_status) != 0) {
    return NULL;
  }
  return result;
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_completePlatformRequest(JNIEnv *env, jobject self, jlong engine, jbyteArray payload) {
  (void)self;
  jsize length = 0;
  jbyte *bytes = borrow_payload(env, payload, &length);
  if (bytes == NULL) {
    return;
  }
  anytty_status_v1 status = anytty_platform_complete(
      (anytty_handle_t)engine, (const uint8_t *)bytes, (size_t)length);
  (*env)->ReleaseByteArrayElements(env, payload, bytes, JNI_ABORT);
  throw_status(env, status);
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_cancel(JNIEnv *env, jobject self, jlong engine, jlong operation) {
  (void)self;
  throw_status(env, anytty_engine_cancel((anytty_handle_t)engine, (anytty_handle_t)operation));
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_closeSession(JNIEnv *env, jobject self, jlong engine, jlong session) {
  (void)self;
  throw_status(env, anytty_engine_close_session((anytty_handle_t)engine, (anytty_handle_t)session));
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_release(JNIEnv *env, jobject self, jlong engine, jlong handle) {
  (void)self;
  throw_status(env, anytty_engine_release((anytty_handle_t)engine, (anytty_handle_t)handle));
}

JNIEXPORT void JNICALL
Java_com_anytty_app_goclient_GoClientNative_close(JNIEnv *env, jobject self, jlong engine) {
  (void)self;
  throw_status(env, anytty_engine_close((anytty_handle_t)engine));
}
