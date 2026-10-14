// Package clientsize is the 640×480 fallback shared by window and webview.
package clientsize

import "github.com/lewtec/lewkit/x/errs"

// ErrSize means a dimension is negative.
var ErrSize = errs.Wrap(errs.ErrInvalid, "invalid size")

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
