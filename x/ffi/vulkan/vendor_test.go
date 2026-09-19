package vulkan

import "testing"

func TestVendorSlug(t *testing.T) {
	tests := []struct {
		vendorID   uint32
		deviceType uint32
		name       string
		want       string
	}{
		{vendorIDAMD, uint32(KindIntegrated), "AMD Radeon Graphics (RADV RENOIR)", "amd"},
		{vendorIDNVIDIA, uint32(KindDedicated), "NVIDIA GeForce RTX 3060", "nvidia"},
		{vendorIDMesa, uint32(KindSoftware), "llvmpipe (LLVM 21.1.8, 256 bits)", "llvmpipe"},
		{vendorIDMesa, uint32(KindSoftware), "lavapipe", "cpu"},
		{vendorIDApple, uint32(KindDedicated), "Apple M5", "apple"},
		{0, uint32(KindDedicated), "Apple M5", "apple"},
		{0, uint32(KindDedicated), "Some GPU", "unknown"},
		{vendorIDIntel, uint32(KindIntegrated), "Intel(R) UHD Graphics", "intel"},
	}
	for _, tt := range tests {
		got := vendorSlug(tt.vendorID, tt.deviceType, tt.name)
		if got != tt.want {
			t.Errorf("vendorSlug(%#x, %d, %q) = %q, want %q", tt.vendorID, tt.deviceType, tt.name, got, tt.want)
		}
	}
}

func TestKindFrom(t *testing.T) {
	tests := []struct {
		deviceType uint32
		name       string
		want       Kind
	}{
		{uint32(KindIntegrated), "AMD Radeon Graphics (RADV RENOIR)", KindIntegrated},
		{uint32(KindDedicated), "NVIDIA GeForce RTX 3060", KindDedicated},
		{uint32(KindSoftware), "llvmpipe (LLVM 21.1.8, 256 bits)", KindSoftware},
		{uint32(KindSoftware), "lavapipe", KindSoftware},
		{uint32(KindVirtual), "VirtIO", KindVirtual},
		{uint32(KindOther), "mystery", KindOther},
		{uint32(KindDedicated), "llvmpipe", KindSoftware},
	}
	for _, tt := range tests {
		got := kindFrom(tt.deviceType, tt.name)
		if got != tt.want {
			t.Errorf("kindFrom(%d, %q) = %v, want %v", tt.deviceType, tt.name, got, tt.want)
		}
	}
}

func TestKindString(t *testing.T) {
	if KindDedicated.String() != "dedicated" {
		t.Fatalf("KindDedicated.String() = %q", KindDedicated.String())
	}
	if KindSoftware.Weight() != 0 || KindDedicated.Weight() != 70 {
		t.Fatalf("weights software=%d dedicated=%d", KindSoftware.Weight(), KindDedicated.Weight())
	}
}
