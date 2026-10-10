/* libGL entry points used to open a GLX context. glXCreateContextAttribsARB is getproc, not dlsym. */
uintptr_t glXChooseFBConfig(uintptr_t dpy, int32_t screen, int32_t *attrib, int32_t *n);
int32_t glXGetFBConfigAttrib(uintptr_t dpy, uintptr_t config, int32_t attrib, int32_t *value);
uintptr_t glXGetProcAddress(char *name);
int32_t glXMakeCurrent(uintptr_t dpy, uintptr_t drawable, uintptr_t ctx);
void glXSwapBuffers(uintptr_t dpy, uintptr_t drawable);
void glXDestroyContext(uintptr_t dpy, uintptr_t ctx);
