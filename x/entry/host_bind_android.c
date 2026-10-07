#include <jni.h>
#include <stdlib.h>
#include <string.h>

static JavaVM *g_vm;

void lewkit_note_vm(JavaVM *vm) {
	if (vm != NULL) {
		g_vm = vm;
	}
}

void lewkit_bind_host(JNIEnv *env) {
	if (g_vm == NULL) {
		(*env)->GetJavaVM(env, &g_vm);
	}
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
