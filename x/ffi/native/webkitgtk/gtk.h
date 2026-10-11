/* GTK entry points shared by GTK 3 and GTK 4. */
void gtk_init(void);
uintptr_t gtk_settings_get_default(void);
void gtk_window_set_title(uintptr_t window, char *title);
void gtk_window_set_default_size(uintptr_t window, int32_t width, int32_t height);
void gtk_window_present(uintptr_t window);
