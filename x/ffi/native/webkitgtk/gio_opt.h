/* Optional libgio entry points. Missing symbols stay nil. */
int64_t g_input_stream_read(uintptr_t stream, uintptr_t buffer, uintptr_t count, uintptr_t cancellable, uintptr_t errorOut);
uintptr_t g_unix_input_stream_new(int32_t fd, int32_t closeFD);
