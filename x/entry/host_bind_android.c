#include <jni.h>
#include <stdlib.h>
#include <string.h>

static jclass g_host;
static JavaVM *g_vm;

void lewkit_bind_host(JNIEnv *env) {
	jclass local;
	if (g_vm == NULL) {
		(*env)->GetJavaVM(env, &g_vm);
	}
	if (g_host != NULL) {
		return;
	}
	local = (*env)->FindClass(env, "lewkit/Host");
	if (local == NULL) {
		(*env)->ExceptionClear(env);
		return;
	}
	g_host = (*env)->NewGlobalRef(env, local);
	(*env)->DeleteLocalRef(env, local);
}

jclass lewkit_host_class(void) {
	return g_host;
}

JNIEnv *lewkit_attach(void) {
	JNIEnv *env = NULL;
	if (g_vm == NULL) {
		return NULL;
	}
	if ((*g_vm)->GetEnv(g_vm, (void **)&env, JNI_VERSION_1_6) == JNI_OK) {
		return env;
	}
	if ((*g_vm)->AttachCurrentThread(g_vm, &env, NULL) != JNI_OK) {
		return NULL;
	}
	return env;
}

char *lewkit_go_string(JNIEnv *env, jstring s) {
	const char *utf;
	char *out;
	if (s == NULL) {
		return NULL;
	}
	utf = (*env)->GetStringUTFChars(env, s, NULL);
	if (utf == NULL) {
		return NULL;
	}
	out = strdup(utf);
	(*env)->ReleaseStringUTFChars(env, s, utf);
	return out;
}
