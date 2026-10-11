/* Subset of CoreAudio used by package coreaudio. */
int32_t AudioObjectGetPropertyDataSize(uint32_t id, uintptr_t addr, uint32_t qualSize, uintptr_t qual, uint32_t *out);
int32_t AudioObjectGetPropertyData(uint32_t id, uintptr_t addr, uint32_t qualSize, uintptr_t qual, uint32_t *size, uintptr_t out);
