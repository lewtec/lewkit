package vulkan

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
		got := VendorFrom(tt.vendorID, tt.name)
		if got != tt.want {
			t.Errorf("VendorFrom(%#x, %q) = %s, want %s", tt.vendorID, tt.name, got, tt.want)
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
	require.Equal(t, "dedicated", DeviceTypeDedicated.String())
	require.Equal(t, 0, DeviceTypeSoftware.Weight())
	require.Equal(t, 70, DeviceTypeDedicated.Weight())
	require.Equal(t, uint32(2), DeviceTypeDedicated.Uint32())
}

func TestParseDeviceType(t *testing.T) {
	got, err := ParseDeviceType("dedicated")
	require.NoError(t, err)
	require.Equal(t, DeviceTypeDedicated, got)
	_, err = ParseDeviceType("discrete")
	require.Error(t, err)
}
