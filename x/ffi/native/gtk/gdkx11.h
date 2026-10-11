/* GDK X11 helpers used by package gtk. Loaded from libgtk. */
void gtk_widget_realize(uintptr_t widget);
uintptr_t gtk_native_get_surface(uintptr_t widget);
uintptr_t gdk_x11_surface_get_xid(uintptr_t surface);
