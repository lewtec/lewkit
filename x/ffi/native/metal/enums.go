package metal

// Metal enum values used by the draw. Triangle strip is 4; 2 is a line strip,
// which rasters the three edges of a quad and leaves a Z.
const (
	pixelBGRA     = 80 // MTLPixelFormatBGRA8Unorm
	loadDontCare  = 0  // MTLLoadActionDontCare
	storeStore    = 1  // MTLStoreActionStore
	primStrip     = 4  // MTLPrimitiveTypeTriangleStrip
	blendOne      = 1  // MTLBlendFactorOne
	blendOneMinus = 5  // MTLBlendFactorOneMinusSourceAlpha
	// channelSwap stays 0. BGRA8Unorm stores shader output as logical RGBA.
	// Swapping red and blue paints the blue lockup amber.
	channelSwap = 0
)
