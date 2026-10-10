/* Subset of winmm.dll used by package winmm. */
uint32_t waveOutGetNumDevs(void);
uint32_t waveOutGetDevCapsW(uintptr_t id, uintptr_t caps, uint32_t size);
uint32_t waveOutOpen(uintptr_t *out, uint32_t id, uintptr_t format, uintptr_t callback, uintptr_t instance, uint32_t flags);
uint32_t waveOutPrepareHeader(uintptr_t h, uintptr_t hdr, uint32_t size);
uint32_t waveOutUnprepareHeader(uintptr_t h, uintptr_t hdr, uint32_t size);
uint32_t waveOutWrite(uintptr_t h, uintptr_t hdr, uint32_t size);
uint32_t waveOutReset(uintptr_t h);
uint32_t waveOutClose(uintptr_t h);
