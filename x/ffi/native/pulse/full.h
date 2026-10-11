/* Subset of libpulse used by package pulse. */
uintptr_t pa_strerror(int32_t code);
uintptr_t pa_mainloop_new(void);
void pa_mainloop_free(uintptr_t m);
uintptr_t pa_mainloop_get_api(uintptr_t m);
int32_t pa_mainloop_prepare(uintptr_t m, int32_t timeout);
int32_t pa_mainloop_poll(uintptr_t m);
int32_t pa_mainloop_dispatch(uintptr_t m);
uintptr_t pa_context_new(uintptr_t api, uintptr_t name);
void pa_context_unref(uintptr_t c);
int32_t pa_context_connect(uintptr_t c, uintptr_t server, int32_t flags, uintptr_t api);
void pa_context_disconnect(uintptr_t c);
int32_t pa_context_get_state(uintptr_t c);
int32_t pa_context_errno(uintptr_t c);
uintptr_t pa_context_get_sink_info_list(uintptr_t c, uintptr_t cb, uintptr_t userdata);
void pa_operation_unref(uintptr_t o);
