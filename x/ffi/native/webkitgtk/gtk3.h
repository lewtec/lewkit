/* GTK 3 entry points. gtk_window_new(GtkWindowType) stays hand-registered: one C name, two Go signatures. */
void gtk_container_add(uintptr_t parent, uintptr_t child);
void gtk_widget_show_all(uintptr_t widget);
void gtk_widget_destroy(uintptr_t widget);
