package vulkan

import "testing"

func TestVendorSlug(t *testing.T) {
	tests := []struct {
		vendorID   uint32
		deviceType uint32
		name       string
		want       string
	}{
		{vendorIDAMD, 1, "AMD Radeon Graphics (RADV RENOIR)", "amd"},
		{vendorIDNVIDIA, 2, "NVIDIA GeForce RTX 3060", "nvidia"},
		{vendorIDMesa, physicalDeviceTypeCPU, "llvmpipe (LLVM 21.1.8, 256 bits)", "llvmpipe"},
		{vendorIDMesa, physicalDeviceTypeCPU, "lavapipe", "cpu"},
		{vendorIDApple, 2, "Apple M5", "apple"},
		{0, 2, "Apple M5", "apple"},
		{0, 2, "Some GPU", "unknown"},
		{vendorIDIntel, 1, "Intel(R) UHD Graphics", "intel"},
	}
	for _, tt := range tests {
		got := vendorSlug(tt.vendorID, tt.deviceType, tt.name)
		if got != tt.want {
			t.Errorf("vendorSlug(%#x, %d, %q) = %q, want %q", tt.vendorID, tt.deviceType, tt.name, got, tt.want)
		}
	}
}
