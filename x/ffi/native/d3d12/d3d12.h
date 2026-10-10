/* C exports from d3d12.dll. COM vtables stay hand-written. */
uint32_t D3D12CreateDevice(uintptr_t adapter, uint32_t level, uintptr_t riid, uintptr_t *dev);
uint32_t D3D12SerializeRootSignature(uintptr_t desc, uint32_t version, uintptr_t *blob, uintptr_t *errBlob);
