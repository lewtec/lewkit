/* Subset of CoreFoundation used by package coreaudio. */
uintptr_t CFStringCreateWithCString(uintptr_t alloc, uintptr_t cstr, uint32_t encoding);
uint8_t CFStringGetCString(uintptr_t str, uintptr_t buf, int64_t size, uint32_t encoding);
int64_t CFStringGetLength(uintptr_t str);
void CFRelease(uintptr_t str);
