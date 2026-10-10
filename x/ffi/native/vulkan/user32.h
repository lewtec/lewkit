/* user32.dll window procs for a Vulkan present window. */
uint16_t RegisterClassW(void *wc);
uintptr_t CreateWindowExW(uint32_t ex, uint16_t *class, uint16_t *title, uint32_t style, int32_t x, int32_t y, int32_t w, int32_t h, uintptr_t parent, uintptr_t menu, uintptr_t inst, uintptr_t param);
int32_t ShowWindow(uintptr_t hwnd, int32_t cmd);
int32_t DestroyWindow(uintptr_t hwnd);
bool PeekMessageW(void *msg, uintptr_t hwnd, uint32_t min, uint32_t max, uint32_t remove);
bool TranslateMessage(void *msg);
uintptr_t DispatchMessageW(void *msg);
