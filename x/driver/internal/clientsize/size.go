// Package clientsize is the 640×480 fallback shared by window and webview.
package clientsize

// Default returns width and height. A negative dimension is invalid.
// Zero becomes 640 or 480.
func Default(width, height int, invalid error) (int, int, error) {
	if width < 0 || height < 0 {
		return 0, 0, invalid
	}
	if width == 0 {
		width = 640
	}
	if height == 0 {
		height = 480
	}
	return width, height, nil
}
