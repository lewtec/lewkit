/* Optional libsoup-3.0 helpers. The library itself may be absent. */
uintptr_t soup_message_headers_new(int32_t kind);
void soup_message_headers_append(uintptr_t headers, char *name, char *value);
