/* Optional X11 atoms. A missing symbol leaves the Go func nil. */
uint64_t XInternAtom(uintptr_t dpy, char *name, int32_t onlyIfExists);
int32_t XChangeProperty(uintptr_t dpy, uint64_t win, uint64_t property, uint64_t typ, int32_t format, int32_t mode, void *data, int32_t nelements);
