//go:build linux && !android

package webkitgtk

import (
	"fmt"
	"strings"
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
			require.FailNow(t, fmt.Sprint("software settings"))
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
			require.FailNow(t, fmt.Sprint("accelerated web view"))
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
			require.FailNow(t, fmt.Sprint("accelerated web view"))
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
		require.FailNow(t, fmt.Sprint("accelerated view"))
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
			require.FailNow(t, fmt.Sprint("website data manager used beside a network session"))
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
			require.FailNow(t, fmt.Sprint("view constructed without a context"))
			return 0
		},
		unref: func(obj uintptr) { unrefs = append(unrefs, obj) },
	}
	_, err := symbols.WebViewWithProfile("/data", "/cache")
	require.ErrorIs(t, err, ErrUnavailable)
	require.ErrorContains(t, err, "website data manager")
	require.Equal(t, []uintptr{11}, unrefs)
}

func pinnedLive() int {
	n := 0
	pinnedBlobs.Range(func(any, any) bool {
		n++
		return true
	})
	return n
}

func TestFinishRequestKeepsThePinUntilDestroy(t *testing.T) {
	body := []byte("hello")
	var data uintptr
	var unrefs []uintptr
	symbols := &Symbols{
		memoryStream: func(pointer uintptr, length int64, destroy uintptr) uintptr {
			data = pointer
			require.NotEqual(t, uintptr(unsafe.Pointer(&body[0])), pointer)
			require.Equal(t, int64(len(body)), length)
			require.Equal(t, destroyPinnedBytes, destroy)
			require.Equal(t, body, unsafe.Slice((*byte)(unsafe.Pointer(pointer)), len(body)))
			return 42
		},
		finishRequest: func(request, stream uintptr, length int64, mime *byte) {
			require.Equal(t, uintptr(1), request)
			require.Equal(t, uintptr(42), stream)
			require.Equal(t, int64(len(body)), length)
			require.Equal(t, "text/plain", native.GoString(uintptr(unsafe.Pointer(mime))))
		},
		unref: func(obj uintptr) { unrefs = append(unrefs, obj) },
	}
	symbols.FinishRequest(1, body, "text/plain")
	require.Equal(t, []uintptr{42}, unrefs)
	require.Equal(t, 1, pinnedLive())
	dropPinned(data)
	require.Zero(t, pinnedLive())
	dropPinned(data)
	require.Zero(t, pinnedLive())
}

func TestFinishRequestDropsThePinWhenTheStreamIsMissing(t *testing.T) {
	symbols := &Symbols{
		memoryStream: func(uintptr, int64, uintptr) uintptr { return 0 },
		finishRequest: func(request, stream uintptr, length int64, mime *byte) {
			require.Equal(t, uintptr(0), stream)
			require.Equal(t, int64(4), length)
		},
		unref: func(uintptr) { require.FailNow(t, "unref of a missing stream") },
	}
	symbols.FinishRequest(1, []byte("body"), "text/plain")
	require.Zero(t, pinnedLive())
}

func TestFinishResponseReleasesTheStreamAndResponse(t *testing.T) {
	var unrefs []uintptr
	symbols := &Symbols{
		unixInputStream: func(fd, closeFD int32) uintptr {
			require.Equal(t, int32(1), closeFD)
			return 10
		},
		responseNew: func(stream uintptr, length int64) uintptr {
			require.Equal(t, uintptr(10), stream)
			require.Equal(t, int64(-1), length)
			return 20
		},
		responseStatus:      func(uintptr, uint32, *byte) {},
		responseContentType: func(uintptr, *byte) {},
		headersNew: func(kind int32) uintptr {
			require.Equal(t, int32(1), kind)
			return 30
		},
		headersAppend: func(uintptr, *byte, *byte) {},
		responseHeaders: func(response, headers uintptr) {
			require.Equal(t, uintptr(20), response)
			require.Equal(t, uintptr(30), headers)
		},
		finishWithResponse: func(request, response uintptr) {
			require.Equal(t, uintptr(7), request)
			require.Equal(t, uintptr(20), response)
		},
		unref: func(obj uintptr) { unrefs = append(unrefs, obj) },
	}
	symbols.FinishResponse(7, 201, "text/plain", []Header{{Name: "X-A", Value: "b"}}, strings.NewReader("hi"))
	require.Equal(t, []uintptr{20, 10}, unrefs)
}

func TestFinishResponseDropsTheStreamWhenResponseNewFails(t *testing.T) {
	var unrefs []uintptr
	symbols := &Symbols{
		unixInputStream: func(int32, int32) uintptr { return 10 },
		responseNew:     func(uintptr, int64) uintptr { return 0 },
		responseStatus:  func(uintptr, uint32, *byte) {},
		finishWithResponse: func(uintptr, uintptr) {
			require.FailNow(t, "finish with a missing response")
		},
		memoryStream: func(data uintptr, length int64, destroy uintptr) uintptr {
			require.Zero(t, data)
			require.Zero(t, length)
			require.Zero(t, destroy)
			return 50
		},
		finishRequest: func(request, stream uintptr, length int64, mime *byte) {
			require.Equal(t, uintptr(7), request)
			require.Equal(t, uintptr(50), stream)
			require.Zero(t, length)
			require.Equal(t, "text/plain", native.GoString(uintptr(unsafe.Pointer(mime))))
		},
		unref: func(obj uintptr) { unrefs = append(unrefs, obj) },
	}
	symbols.FinishResponse(7, 200, "text/plain", nil, strings.NewReader("x"))
	require.Equal(t, []uintptr{10, 50}, unrefs)
	require.Zero(t, pinnedLive())
}

func TestFinishNotFoundFreesTheError(t *testing.T) {
	var freed []uintptr
	symbols := &Symbols{
		newError: func(domain uint32, code int32, message *byte) uintptr {
			require.Equal(t, uint32(4), domain)
			require.Equal(t, int32(1), code)
			require.Equal(t, "not found", native.GoString(uintptr(unsafe.Pointer(message))))
			return 9
		},
		ioErrorQuark: func() uint32 { return 4 },
		finishRequestError: func(request, gError uintptr) {
			require.Equal(t, uintptr(3), request)
			require.Equal(t, uintptr(9), gError)
		},
		freeError: func(gError uintptr) { freed = append(freed, gError) },
	}
	symbols.FinishNotFound(3)
	require.Equal(t, []uintptr{9}, freed)
}

func TestInputStreamReadPinsTheBuffer(t *testing.T) {
	symbols := &Symbols{
		streamRead: func(stream, buffer uintptr, count, cancellable, errorOut uintptr) int64 {
			require.Equal(t, uintptr(1), stream)
			require.Equal(t, uintptr(2), count)
			*(*byte)(unsafe.Pointer(buffer)) = 'Z'
			*(*uintptr)(unsafe.Pointer(errorOut)) = 0
			return 1
		},
	}
	reader := &inputStream{symbols: symbols, stream: 1}
	buf := []byte{'a', 'b'}
	n, err := reader.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	require.Equal(t, []byte{'Z', 'b'}, buf)
}
