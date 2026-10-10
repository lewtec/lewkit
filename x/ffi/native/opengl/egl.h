/* Required EGL entry points. eglGetPlatformDisplay is optional and lives in egl_platform.h. */
uintptr_t eglGetDisplay(uintptr_t display);
uint32_t eglInitialize(uintptr_t dpy, int32_t *major, int32_t *minor);
uint32_t eglBindAPI(uint32_t api);
uint32_t eglChooseConfig(uintptr_t dpy, int32_t *attrib, uintptr_t *configs, int32_t n, int32_t *num);
uintptr_t eglCreateContext(uintptr_t dpy, uintptr_t config, uintptr_t share, int32_t *attrib);
uintptr_t eglCreateWindowSurface(uintptr_t dpy, uintptr_t config, uintptr_t win, int32_t *attrib);
uintptr_t eglCreatePbufferSurface(uintptr_t dpy, uintptr_t config, int32_t *attrib);
uint32_t eglMakeCurrent(uintptr_t dpy, uintptr_t draw, uintptr_t read, uintptr_t ctx);
uint32_t eglSwapBuffers(uintptr_t dpy, uintptr_t surface);
uint32_t eglDestroyContext(uintptr_t dpy, uintptr_t ctx);
uint32_t eglDestroySurface(uintptr_t dpy, uintptr_t surface);
uintptr_t eglGetProcAddress(char *name);
int32_t eglGetError(void);
