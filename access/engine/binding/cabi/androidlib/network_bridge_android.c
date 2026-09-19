#include <jni.h>

extern void anytty_android_network_snapshot(char *data);

JNIEXPORT void JNICALL
Java_com_anytty_app_AndroidNetworkBridge_updateNetworks(JNIEnv *env, jobject self, jstring snapshot) {
    (void)self;
    const char *data = (*env)->GetStringUTFChars(env, snapshot, 0);
    if (data == NULL) return;
    anytty_android_network_snapshot((char *)data);
    (*env)->ReleaseStringUTFChars(env, snapshot, data);
}
