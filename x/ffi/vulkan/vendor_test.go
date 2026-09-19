package vulkan

import "testing"

func TestVendorFrom(t *testing.T) {
	tests := []struct {
		vendorID uint32
		name     string
		want     Vendor
	}{
		{VendorAMD.Uint32(), "AMD Radeon Graphics (RADV RENOIR)", VendorAMD},
		{VendorNVIDIA.Uint32(), "NVIDIA GeForce RTX 3060", VendorNVIDIA},
		{VendorMesa.Uint32(), "llvmpipe (LLVM 21.1.8, 256 bits)", VendorMesa},
		{VendorMesa.Uint32(), "lavapipe", VendorMesa},
		{VendorApple.Uint32(), "Apple M5", VendorApple},
		{0, "Apple M5", VendorApple},
		{0, "Some GPU", VendorUnknown},
		{VendorIntel.Uint32(), "Intel(R) UHD Graphics", VendorIntel},
	}
	for _, tt := range tests {
		got := vendorFrom(tt.vendorID, tt.name)
		if got != tt.want {
			t.Errorf("vendorFrom(%#x, %q) = %s, want %s", tt.vendorID, tt.name, got, tt.want)
		}
	}
}

func TestParseVendor(t *testing.T) {
	got, err := ParseVendor("nvidia")
	if err != nil || got != VendorNVIDIA {
		t.Fatalf("ParseVendor(nvidia) = %v, %v", got, err)
	}
	if got.Uint32() != 0x10de {
		t.Fatalf("VendorNVIDIA.Uint32() = %#x", got.Uint32())
	}
	_, err = ParseVendor("nouveau")
	if err == nil {
		t.Fatal("ParseVendor(nouveau) succeeded")
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
	if DeviceTypeDedicated.Uint32() != 2 {
		t.Fatalf("DeviceTypeDedicated.Uint32() = %d", DeviceTypeDedicated.Uint32())
	}
}

func TestParseDeviceType(t *testing.T) {
	got, err := ParseDeviceType("dedicated")
	if err != nil || got != DeviceTypeDedicated {
		t.Fatalf("ParseDeviceType(dedicated) = %v, %v", got, err)
	}
	_, err = ParseDeviceType("discrete")
	if err == nil {
		t.Fatal("ParseDeviceType(discrete) succeeded")
	}
}
