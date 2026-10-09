//go:build linux && !android

package native

import (
	"os"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

const (
	eglNone                          = 0x3038
	eglHeight                        = 0x3056
	eglWidth                         = 0x3057
	eglExtensions                    = 0x3055
	eglPlatformSurfaceless           = 0x31DD
	eglOpenGLESAPI                   = 0x30A0
	eglRenderableType                = 0x3040
	eglOpenGLES2Bit                  = 0x0004
	eglContextClientVersion          = 0x3098
	eglGLTexture2D                   = 0x30B1
	eglLinuxDMABuf                   = 0x3270
	eglLinuxDRMFourCC                = 0x3271
	eglDMABufPlane0FD                = 0x3272
	eglDMABufPlane0Offset            = 0x3273
	eglDMABufPlane0Pitch             = 0x3274
	eglDMABufPlane0ModifierLo        = 0x3443
	eglDMABufPlane0ModifierHi        = 0x3444
	glTexture2D               uint32 = 0x0DE1
	glRGBA                    int32  = 0x1908
	glUnsignedByte            uint32 = 0x1401
	drmFormatARGB8888                = 0x34325241
	drmFormatXRGB8888                = 0x34325258
	gbmUseRendering           uint32 = 1 << 2
	gbmUseLinear              uint32 = 1 << 4
)

// accelSmoke is one try of the EGL display and, when the driver
// advertises DMA-BUF import, one eglCreateImage. WebKit aborts the web
// process when eglInitialize fails, and a failed import logs
// "Failed to create EGL image from DMABuf" on every frame.
type accelSmoke struct {
	display   bool
	importExt bool
	image     bool
}

// useHardwareAccel keeps the GPU when the display comes up and a DMA-BUF
// import is either unsupported or the one smoke image succeeded.
// A failure is software rendering.
func useHardwareAccel(smoke accelSmoke) bool {
	if !smoke.display {
		return false
	}
	if !smoke.importExt {
		return true
	}
	return smoke.image
}

var (
	dmaOnce sync.Once
	dmaOK   bool
)

// DMABufSmoke runs the EGL probe once. A failure selects software
// rendering and, when the caller has not chosen a WebKit renderer,
// shared-memory buffers. The probe returns on failure. It does not abort.
func DMABufSmoke() bool {
	dmaOnce.Do(func() {
		var smoke accelSmoke
		func() {
			defer func() { _ = recover() }()
			smoke = probeAccel()
		}()
		dmaOK = useHardwareAccel(smoke)
		if dmaOK {
			return
		}
		if key, value, ok := sharedMemoryRendererEnv(os.Getenv("WEBKIT_DISABLE_DMABUF_RENDERER"), os.Getenv("WEBKIT_DMABUF_RENDERER_FORCE_SHM")); ok {
			_ = os.Setenv(key, value)
		}
	})
	return dmaOK
}

func probeAccel() accelSmoke {
	lib, err := Open("libEGL.so.1", Now)
	if err != nil || lib == 0 {
		return accelSmoke{}
	}
	var queryString func(display uintptr, name int32) uintptr
	var initialize func(display uintptr, major, minor *int32) uint32
	var getDisplay func(native uintptr) uintptr
	var getPlatform func(platform uint32, native, attribs uintptr) uintptr
	var createImage func(display, context uintptr, target uint32, buffer uintptr, attribs *uintptr) uintptr
	var destroyImage func(display, image uintptr) uint32
	if !bindSymbol(lib, "eglQueryString", &queryString) || !bindSymbol(lib, "eglInitialize", &initialize) || !bindSymbol(lib, "eglGetDisplay", &getDisplay) {
		return accelSmoke{}
	}
	_ = bindSymbol(lib, "eglGetPlatformDisplay", &getPlatform)
	_ = bindSymbol(lib, "eglCreateImage", &createImage)
	_ = bindSymbol(lib, "eglDestroyImage", &destroyImage)

	clientExt := GoString(queryString(0, eglExtensions))
	display := uintptr(0)
	if getPlatform != nil && strings.Contains(clientExt, "EGL_MESA_platform_surfaceless") {
		display = getPlatform(eglPlatformSurfaceless, 0, 0)
	}
	if display == 0 {
		display = getDisplay(0)
	}
	if display == 0 {
		return accelSmoke{}
	}
	var major, minor int32
	if initialize(display, &major, &minor) == 0 {
		return accelSmoke{}
	}
	// Leave this display up. eglTerminate on some drivers drops every
	// EGL display in the process, including the one WebKit uses next.
	smoke := accelSmoke{display: true}
	if !strings.Contains(GoString(queryString(display, eglExtensions)), "EGL_EXT_image_dma_buf_import") {
		return smoke
	}
	smoke.importExt = true
	if createImage == nil {
		return smoke
	}
	// conda-forge WebKit is built without GBM, so it exports a GL texture
	// and the UI imports that DMA-BUF. Prefer that round trip. A GBM
	// buffer is the path a WebKit built with GBM uses.
	dispExt := GoString(queryString(display, eglExtensions))
	if strings.Contains(dispExt, "EGL_MESA_image_dma_buf_export") && strings.Contains(dispExt, "EGL_KHR_surfaceless_context") {
		smoke.image = exportRoundTrip(lib, display, dispExt, createImage, destroyImage)
		return smoke
	}
	smoke.image = importOneDMABuf(display, createImage, destroyImage)
	return smoke
}

func exportRoundTrip(lib, display uintptr, dispExt string, createImage func(uintptr, uintptr, uint32, uintptr, *uintptr) uintptr, destroyImage func(uintptr, uintptr) uint32) bool {
	var bindAPI func(api uint32) uint32
	var chooseConfig func(display uintptr, attribs *int32, configs *uintptr, count int32, num *int32) uint32
	var createContext func(display, config, share uintptr, attribs *int32) uintptr
	var makeCurrent func(display, draw, read, context uintptr) uint32
	var destroyContext func(display, context uintptr) uint32
	if !bindSymbol(lib, "eglBindAPI", &bindAPI) || !bindSymbol(lib, "eglChooseConfig", &chooseConfig) || !bindSymbol(lib, "eglCreateContext", &createContext) || !bindSymbol(lib, "eglMakeCurrent", &makeCurrent) || !bindSymbol(lib, "eglDestroyContext", &destroyContext) {
		return false
	}
	var queryAPI func() uint32
	previous := uint32(0)
	if bindSymbol(lib, "eglQueryAPI", &queryAPI) {
		previous = queryAPI()
	}
	if bindAPI(eglOpenGLESAPI) == 0 {
		return false
	}
	if previous != 0 {
		defer bindAPI(previous)
	}
	cfgAttribs := []int32{eglRenderableType, eglOpenGLES2Bit, eglNone}
	var config uintptr
	var ncfg int32
	if chooseConfig(display, &cfgAttribs[0], &config, 1, &ncfg) == 0 || ncfg < 1 || config == 0 {
		return false
	}
	ctxAttribs := []int32{eglContextClientVersion, 2, eglNone}
	ctx := createContext(display, config, 0, &ctxAttribs[0])
	if ctx == 0 {
		return false
	}
	defer destroyContext(display, ctx)
	if makeCurrent(display, 0, 0, ctx) == 0 {
		return false
	}
	defer makeCurrent(display, 0, 0, 0)
	gles, err := Open("libGLESv2.so.2", Now)
	if err != nil || gles == 0 {
		return false
	}
	var genTextures func(n int32, textures *uint32)
	var bindTexture func(target uint32, texture uint32)
	var texImage func(target uint32, level, internal, width, height, border int32, format, kind uint32, pixels *byte)
	var deleteTextures func(n int32, textures *uint32)
	if !bindSymbol(gles, "glGenTextures", &genTextures) || !bindSymbol(gles, "glBindTexture", &bindTexture) || !bindSymbol(gles, "glTexImage2D", &texImage) || !bindSymbol(gles, "glDeleteTextures", &deleteTextures) {
		return false
	}
	var tex uint32
	genTextures(1, &tex)
	if tex == 0 {
		return false
	}
	defer deleteTextures(1, &tex)
	bindTexture(glTexture2D, tex)
	texImage(glTexture2D, 0, glRGBA, 64, 64, 0, uint32(glRGBA), glUnsignedByte, nil)
	none := []uintptr{eglNone}
	texImageEGL := createImage(display, ctx, eglGLTexture2D, uintptr(tex), &none[0])
	if texImageEGL == 0 {
		return false
	}
	defer func() {
		if destroyImage != nil {
			destroyImage(display, texImageEGL)
		}
	}()
	query, export := mesaExport(lib)
	if query == nil || export == nil {
		return false
	}
	var fourcc, planes int32
	var modifier uint64
	if query(display, texImageEGL, &fourcc, &planes, &modifier) == 0 || planes < 1 || planes > 4 || fourcc == 0 {
		return false
	}
	fds := make([]int32, planes)
	for i := range fds {
		fds[i] = -1
	}
	strides := make([]int32, planes)
	offsets := make([]int32, planes)
	defer func() {
		for _, fd := range fds {
			if fd >= 0 {
				unix.Close(int(fd))
			}
		}
	}()
	if export(display, texImageEGL, &fds[0], &strides[0], &offsets[0]) == 0 || fds[0] < 0 || strides[0] <= 0 {
		return false
	}
	attribs := []uintptr{
		eglWidth, 64,
		eglHeight, 64,
		eglLinuxDRMFourCC, uintptr(fourcc),
		eglDMABufPlane0FD, uintptr(fds[0]),
		eglDMABufPlane0Offset, uintptr(offsets[0]),
		eglDMABufPlane0Pitch, uintptr(strides[0]),
	}
	if strings.Contains(dispExt, "EGL_EXT_image_dma_buf_import_modifiers") && modifier != 0x00ffffffffffffff {
		attribs = append(attribs,
			eglDMABufPlane0ModifierLo, uintptr(uint32(modifier)),
			eglDMABufPlane0ModifierHi, uintptr(uint32(modifier>>32)),
		)
	}
	attribs = append(attribs, eglNone)
	imported := createImage(display, 0, eglLinuxDMABuf, 0, &attribs[0])
	if imported == 0 {
		return false
	}
	if destroyImage != nil {
		destroyImage(display, imported)
	}
	return true
}

func mesaExport(lib uintptr) (query func(display, image uintptr, fourcc, planes *int32, modifier *uint64) uint32, export func(display, image uintptr, fds, strides, offsets *int32) uint32) {
	queryAddr := lookupProc(lib, "eglExportDMABUFImageQueryMESA")
	exportAddr := lookupProc(lib, "eglExportDMABUFImageMESA")
	if queryAddr == 0 || exportAddr == 0 {
		return nil, nil
	}
	Register(&query, queryAddr)
	Register(&export, exportAddr)
	return query, export
}

func lookupProc(lib uintptr, name string) uintptr {
	var getProc func(*byte) uintptr
	if !bindSymbol(lib, "eglGetProcAddress", &getProc) {
		return 0
	}
	cname := CString(name)
	addr := getProc(&cname[0])
	runtime.KeepAlive(cname)
	return addr
}

func bindSymbol(lib uintptr, name string, fnptr any) bool {
	addr, err := Symbol(lib, name)
	if err != nil || addr == 0 {
		return false
	}
	Register(fnptr, addr)
	return true
}

func importOneDMABuf(display uintptr, createImage func(uintptr, uintptr, uint32, uintptr, *uintptr) uintptr, destroyImage func(uintptr, uintptr) uint32) bool {
	fd, ok := renderNode()
	if !ok {
		return false
	}
	defer unix.Close(fd)
	gbm, err := Open("libgbm.so.1", Now)
	if err != nil || gbm == 0 {
		return false
	}
	var createDevice func(fd int32) uintptr
	var destroyDevice func(device uintptr)
	var boCreate func(device uintptr, width, height, format, flags uint32) uintptr
	var boDestroy func(bo uintptr)
	var boFD func(bo uintptr) int32
	var boStride func(bo uintptr) uint32
	if !bindSymbol(gbm, "gbm_create_device", &createDevice) || !bindSymbol(gbm, "gbm_device_destroy", &destroyDevice) || !bindSymbol(gbm, "gbm_bo_create", &boCreate) || !bindSymbol(gbm, "gbm_bo_destroy", &boDestroy) || !bindSymbol(gbm, "gbm_bo_get_fd", &boFD) || !bindSymbol(gbm, "gbm_bo_get_stride", &boStride) {
		return false
	}
	device := createDevice(int32(fd))
	if device == 0 {
		return false
	}
	defer destroyDevice(device)
	bo, format := createLinearBO(boCreate, device)
	if bo == 0 {
		return false
	}
	defer boDestroy(bo)
	buf := boFD(bo)
	if buf < 0 {
		return false
	}
	defer unix.Close(int(buf))
	stride := boStride(bo)
	if stride == 0 {
		return false
	}
	attribs := []uintptr{
		eglWidth, 64,
		eglHeight, 64,
		eglLinuxDRMFourCC, uintptr(format),
		eglDMABufPlane0FD, uintptr(buf),
		eglDMABufPlane0Offset, 0,
		eglDMABufPlane0Pitch, uintptr(stride),
		eglNone,
	}
	image := createImage(display, 0, eglLinuxDMABuf, 0, &attribs[0])
	if image == 0 {
		return false
	}
	if destroyImage != nil {
		destroyImage(display, image)
	}
	return true
}

func createLinearBO(boCreate func(uintptr, uint32, uint32, uint32, uint32) uintptr, device uintptr) (uintptr, uint32) {
	flags := gbmUseRendering | gbmUseLinear
	for _, format := range []uint32{drmFormatARGB8888, drmFormatXRGB8888} {
		if bo := boCreate(device, 64, 64, format, flags); bo != 0 {
			return bo, format
		}
	}
	return 0, 0
}

func renderNode() (int, bool) {
	entries, err := os.ReadDir("/dev/dri")
	if err != nil {
		return 0, false
	}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "renderD") {
			continue
		}
		fd, err := unix.Open("/dev/dri/"+name, unix.O_RDWR|unix.O_CLOEXEC, 0)
		if err != nil {
			continue
		}
		return fd, true
	}
	return 0, false
}
