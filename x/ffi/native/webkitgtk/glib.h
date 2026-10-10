/* Required libglib entry points for WebKitGTK. */
int32_t g_main_context_iteration(uintptr_t ctx, int32_t block);
void g_main_context_wakeup(uintptr_t ctx);
int32_t g_main_context_pending(uintptr_t ctx);
void g_free(uintptr_t p);
uintptr_t g_error_new_literal(uint32_t domain, int32_t code, char *message);
void g_error_free(uintptr_t gError);
