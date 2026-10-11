/* Subset of CoreGraphics used to upload a window image. */
uintptr_t CGColorSpaceCreateDeviceRGB(void);
uintptr_t CGDataProviderCreateWithCFData(uintptr_t data);
uintptr_t CGImageCreate(uintptr_t width, uintptr_t height, uintptr_t bpc, uintptr_t bpp, uintptr_t bpr, uintptr_t space, uint32_t bitmapInfo, uintptr_t provider, uintptr_t decode, bool interpolate, int32_t intent);
void CGColorSpaceRelease(uintptr_t space);
void CGDataProviderRelease(uintptr_t provider);
void CGImageRelease(uintptr_t image);
