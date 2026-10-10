/* libX11 entry points for a Vulkan present window. */
uintptr_t XOpenDisplay(char *name);
int32_t XCloseDisplay(uintptr_t dpy);
int32_t XDefaultScreen(uintptr_t dpy);
uint64_t XRootWindow(uintptr_t dpy, int32_t screen);
uint64_t XBlackPixel(uintptr_t dpy, int32_t screen);
uint64_t XWhitePixel(uintptr_t dpy, int32_t screen);
uint64_t XCreateSimpleWindow(uintptr_t dpy, uint64_t parent, int32_t x, int32_t y, uint32_t w, uint32_t h, uint32_t border, uint64_t borderC, uint64_t bg);
int32_t XStoreName(uintptr_t dpy, uint64_t win, char *name);
int32_t XMapWindow(uintptr_t dpy, uint64_t win);
int32_t XFlush(uintptr_t dpy);
int32_t XDestroyWindow(uintptr_t dpy, uint64_t win);
int32_t XSelectInput(uintptr_t dpy, uint64_t win, int64_t mask);
int32_t XPending(uintptr_t dpy);
int32_t XNextEvent(uintptr_t dpy, void *ev);
