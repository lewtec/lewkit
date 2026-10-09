//go:build linux && !android

package webkitgtk

import (
	"testing"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/stretchr/testify/require"
)

func TestWebKitDirsUseTheSharedChain(t *testing.T) {
	t.Setenv("WEBKITGTK_LIB", "/tmp/webkit-a:/tmp/webkit-b")
	dirs := native.SearchDirs(webkitDirs()...)
	require.Equal(t, []string{"/tmp/webkit-a", "/tmp/webkit-b"}, dirs[:2])
	require.Contains(t, dirs, "/run/current-system/sw/lib")
}

func TestPreferDarkPropertyFollowsTheGTKMajor(t *testing.T) {
	require.Equal(t, "gtk-application-prefer-dark-theme", preferDarkProperty(true))
	require.Equal(t, "gtk-application-prefer-dark", preferDarkProperty(false))
}

func TestPaintInSoftwareAfterAFailedSmoke(t *testing.T) {
	require.True(t, paintInSoftware(false))
	require.False(t, paintInSoftware(true))
	require.Equal(t, int32(2), softwarePolicy(true))
	require.Equal(t, int32(1), softwarePolicy(false))
}

func stubAccel(t *testing.T, ok bool) {
	prev := accelAvailable
	accelAvailable = func() bool { return ok }
	t.Cleanup(func() { accelAvailable = prev })
}

func TestGTK3WebViewKeepsAccelerationWhenTheSmokePasses(t *testing.T) {
	stubAccel(t, true)
	symbols := &Symbols{
		gtk3: true,
		settingsNew: func() uintptr {
			t.Fatal("software settings")
			return 0
		},
		newWebView: func() uintptr { return 8 },
	}
	require.Equal(t, uintptr(8), symbols.WebViewNew())
}

func TestGTK3WebViewDisablesHardwareAcceleration(t *testing.T) {
	stubAccel(t, false)
	var policy int32
	var released []uintptr
	symbols := &Symbols{
		gtk3:        true,
		settingsNew: func() uintptr { return 4 },
		setHardwarePolicy: func(settings uintptr, value int32) {
			require.Equal(t, uintptr(4), settings)
			policy = value
		},
		webViewType: func() uintptr { return 5 },
		objectNew: func(objectType uintptr, property *byte, value, terminator uintptr) uintptr {
			require.Equal(t, uintptr(5), objectType)
			require.Equal(t, "settings", native.GoString(uintptr(unsafe.Pointer(property))))
			require.Equal(t, uintptr(4), value)
			require.Equal(t, uintptr(0), terminator)
			return 9
		},
		newWebView: func() uintptr {
			t.Fatal("accelerated web view")
			return 0
		},
		unref: func(obj uintptr) { released = append(released, obj) },
	}
	require.Equal(t, uintptr(9), symbols.WebViewNew())
	require.Equal(t, hardwareAccelerationNever, policy)
	require.Equal(t, []uintptr{4}, released)
}

func TestGTK4WebViewDisablesHardwareAcceleration(t *testing.T) {
	stubAccel(t, false)
	var policy int32
	symbols := &Symbols{
		settingsNew: func() uintptr { return 4 },
		setHardwarePolicy: func(settings uintptr, value int32) {
			require.Equal(t, uintptr(4), settings)
			policy = value
		},
		webViewType: func() uintptr { return 5 },
		objectNew: func(objectType uintptr, property *byte, value, terminator uintptr) uintptr {
			require.Equal(t, uintptr(5), objectType)
			require.Equal(t, "settings", native.GoString(uintptr(unsafe.Pointer(property))))
			require.Equal(t, uintptr(4), value)
			require.Equal(t, uintptr(0), terminator)
			return 9
		},
		newWebView: func() uintptr {
			t.Fatal("accelerated web view")
			return 0
		},
		unref: func(uintptr) {},
	}
	require.Equal(t, uintptr(9), symbols.WebViewNew())
	require.Equal(t, int32(1), policy)
}

func TestGTK3ProfileViewPassesTheWebContext(t *testing.T) {
	stubAccel(t, false)
	var policy int32
	symbols := &Symbols{gtk3: true}
	symbols.settingsNew = func() uintptr { return 4 }
	symbols.setHardwarePolicy = func(settings uintptr, value int32) {
		require.Equal(t, uintptr(4), settings)
		policy = value
	}
	symbols.webViewType = func() uintptr { return 5 }
	symbols.objectNewPair = func(objectType uintptr, name1 *byte, value1 uintptr, name2 *byte, value2, end uintptr) uintptr {
		require.Equal(t, uintptr(5), objectType)
		require.Equal(t, "web-context", native.GoString(uintptr(unsafe.Pointer(name1))))
		require.Equal(t, uintptr(22), value1)
		require.Equal(t, "settings", native.GoString(uintptr(unsafe.Pointer(name2))))
		require.Equal(t, uintptr(4), value2)
		require.Equal(t, uintptr(0), end)
		return 33
	}
	symbols.dataManagerNew = func(name1, value1, name2, value2 *byte, end uintptr) uintptr { return 11 }
	symbols.contextWithManager = func(manager uintptr) uintptr { return 22 }
	symbols.viewWithContext = func(context uintptr) uintptr {
		t.Fatal("accelerated view")
		return 0
	}
	symbols.unref = func(uintptr) {}
	view, err := symbols.WebViewWithProfile("/data", "/cache")
	require.NoError(t, err)
	require.Equal(t, uintptr(33), view)
	require.Equal(t, hardwareAccelerationNever, policy)
}

func TestProfileBackendPrefersNetworkSession(t *testing.T) {
	require.Equal(t, "network-session", profileBackend(true, true))
	require.Equal(t, "network-session", profileBackend(true, false))
	require.Equal(t, "website-data-manager", profileBackend(false, true))
	require.Empty(t, profileBackend(false, false))
}

func TestWebViewWithProfileNeedsAConstructor(t *testing.T) {
	_, err := (&Symbols{}).WebViewWithProfile("/data", "/cache")
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorContains(t, err, "network session")
}

func TestWebKit6ProfileUsesNetworkSession(t *testing.T) {
	symbols := &Symbols{
		networkSession: func(data, cache *byte) uintptr {
			require.Equal(t, "/data", native.GoString(uintptr(unsafe.Pointer(data))))
			require.Equal(t, "/cache", native.GoString(uintptr(unsafe.Pointer(cache))))
			return 7
		},
		objectNew: func(objectType uintptr, property *byte, value, terminator uintptr) uintptr {
			require.Equal(t, uintptr(5), objectType)
			require.Equal(t, "network-session", native.GoString(uintptr(unsafe.Pointer(property))))
			require.Equal(t, uintptr(7), value)
			require.Equal(t, uintptr(0), terminator)
			return 9
		},
		webViewType: func() uintptr { return 5 },
		dataManagerNew: func(name1, value1, name2, value2 *byte, end uintptr) uintptr {
			t.Fatal("website data manager used beside a network session")
			return 0
		},
		contextWithManager: func(manager uintptr) uintptr { return 1 },
		viewWithContext:    func(context uintptr) uintptr { return 1 },
	}
	view, err := symbols.WebViewWithProfile("/data", "/cache")
	require.NoError(t, err)
	require.Equal(t, uintptr(9), view)
}

func TestWebKit41ProfileBuildsAWebsiteDataManager(t *testing.T) {
	var unrefs []uintptr
	symbols := &Symbols{
		dataManagerNew: func(name1, value1, name2, value2 *byte, end uintptr) uintptr {
			require.Equal(t, uintptr(0), end)
			require.Equal(t, "base-data-directory", native.GoString(uintptr(unsafe.Pointer(name1))))
			require.Equal(t, "/data", native.GoString(uintptr(unsafe.Pointer(value1))))
			require.Equal(t, "base-cache-directory", native.GoString(uintptr(unsafe.Pointer(name2))))
			require.Equal(t, "/cache", native.GoString(uintptr(unsafe.Pointer(value2))))
			return 11
		},
		contextWithManager: func(manager uintptr) uintptr {
			require.Equal(t, uintptr(11), manager)
			return 22
		},
		viewWithContext: func(context uintptr) uintptr {
			require.Equal(t, uintptr(22), context)
			return 33
		},
		unref: func(obj uintptr) { unrefs = append(unrefs, obj) },
	}
	view, err := symbols.WebViewWithProfile("/data", "/cache")
	require.NoError(t, err)
	require.Equal(t, uintptr(33), view)
	require.Equal(t, []uintptr{22, 11}, unrefs)
}

func TestWebsiteDataManagerFailureReleasesTheManager(t *testing.T) {
	var unrefs []uintptr
	symbols := &Symbols{
		dataManagerNew:     func(name1, value1, name2, value2 *byte, end uintptr) uintptr { return 11 },
		contextWithManager: func(manager uintptr) uintptr { return 0 },
		viewWithContext: func(context uintptr) uintptr {
			t.Fatal("view constructed without a context")
			return 0
		},
		unref: func(obj uintptr) { unrefs = append(unrefs, obj) },
	}
	_, err := symbols.WebViewWithProfile("/data", "/cache")
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorContains(t, err, "website data manager")
	require.Equal(t, []uintptr{11}, unrefs)
}
