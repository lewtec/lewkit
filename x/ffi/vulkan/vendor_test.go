package vulkan

import "testing"

func TestVendorSlug(t *testing.T) {
	tests := []struct {
		vendorID   uint32
		deviceType uint32
		name       string
		want       string
	}{
		{vendorIDAMD, uint32(DeviceTypeIntegrated), "AMD Radeon Graphics (RADV RENOIR)", "amd"},
		{vendorIDNVIDIA, uint32(DeviceTypeDedicated), "NVIDIA GeForce RTX 3060", "nvidia"},
		{vendorIDMesa, uint32(DeviceTypeSoftware), "llvmpipe (LLVM 21.1.8, 256 bits)", "llvmpipe"},
		{vendorIDMesa, uint32(DeviceTypeSoftware), "lavapipe", "cpu"},
		{vendorIDApple, uint32(DeviceTypeDedicated), "Apple M5", "apple"},
		{0, uint32(DeviceTypeDedicated), "Apple M5", "apple"},
		{0, uint32(DeviceTypeDedicated), "Some GPU", "unknown"},
		{vendorIDIntel, uint32(DeviceTypeIntegrated), "Intel(R) UHD Graphics", "intel"},
	}
	for _, tt := range tests {
		got := vendorSlug(tt.vendorID, tt.deviceType, tt.name)
		if got != tt.want {
			t.Errorf("vendorSlug(%#x, %d, %q) = %q, want %q", tt.vendorID, tt.deviceType, tt.name, got, tt.want)
		}
	}
}

func TestDeviceTypeFrom(t *testing.T) {
	tests := []struct {
		deviceType uint32
		name       string
		want       DeviceType
	}{
		{uint32(DeviceTypeIntegrated), "AMD Radeon Graphics (RADV RENOIR)", DeviceTypeIntegrated},
		{uint32(DeviceTypeDedicated), "NVIDIA GeForce RTX 3060", DeviceTypeDedicated},
		{uint32(DeviceTypeSoftware), "llvmpipe (LLVM 21.1.8, 256 bits)", DeviceTypeSoftware},
		{uint32(DeviceTypeSoftware), "lavapipe", DeviceTypeSoftware},
		{uint32(DeviceTypeVirtual), "VirtIO", DeviceTypeVirtual},
		{uint32(DeviceTypeOther), "mystery", DeviceTypeOther},
		{uint32(DeviceTypeDedicated), "llvmpipe", DeviceTypeSoftware},
	}
	for _, tt := range tests {
		got := deviceTypeFrom(tt.deviceType, tt.name)
		if got != tt.want {
			t.Errorf("deviceTypeFrom(%d, %q) = %v, want %v", tt.deviceType, tt.name, got, tt.want)
		}
	}
}

func TestDeviceTypeString(t *testing.T) {
	if DeviceTypeDedicated.String() != "dedicated" {
		t.Fatalf("DeviceTypeDedicated.String() = %q", DeviceTypeDedicated.String())
	}
	if DeviceTypeSoftware.Weight() != 0 || DeviceTypeDedicated.Weight() != 70 {
		t.Fatalf("weights software=%d dedicated=%d", DeviceTypeSoftware.Weight(), DeviceTypeDedicated.Weight())
	}
}
