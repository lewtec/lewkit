/* libX11 entry points used to choose a GLX framebuffer. */
int32_t XDefaultScreen(uintptr_t dpy);
int32_t XGetWindowAttributes(uintptr_t dpy, uintptr_t win, void *attr);
uintptr_t XVisualIDFromVisual(uintptr_t visual);
int32_t XFree(uintptr_t p);
