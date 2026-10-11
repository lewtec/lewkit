/* Subset of AudioToolbox used by package coreaudio. */
int32_t AudioQueueNewOutput(uintptr_t format, uintptr_t cb, uintptr_t user, uintptr_t runLoop, uintptr_t mode, uint32_t flags, uintptr_t *out);
int32_t AudioQueueAllocateBuffer(uintptr_t q, uint32_t size, uintptr_t *out);
int32_t AudioQueueEnqueueBuffer(uintptr_t q, uintptr_t buf, uint32_t packets, uintptr_t descs);
int32_t AudioQueueStart(uintptr_t q, uintptr_t when);
int32_t AudioQueueStop(uintptr_t q, uint8_t immediate);
int32_t AudioQueueDispose(uintptr_t q, uint8_t immediate);
int32_t AudioQueueSetProperty(uintptr_t q, uint32_t prop, uintptr_t data, uint32_t size);
