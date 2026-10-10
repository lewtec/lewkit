/* GTK 4 entry points. gtk_window_new here takes no arguments. */
uintptr_t gtk_window_new(void);
void gtk_window_set_child(uintptr_t window, uintptr_t child);
void gtk_window_destroy(uintptr_t window);
