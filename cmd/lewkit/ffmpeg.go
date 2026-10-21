package main

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/ffmpeg"
)

// errFFmpeg is the lewkit wrapper exit when ffmpeg returns a non-zero status.
var errFFmpeg = errs.Wrap(errs.ErrUnknown, "ffmpeg")

type ffmpegCmd struct{}

func (ffmpegCmd) Description() string {
	return "read, filter, and write media with the ffmpeg CLI"
}

func (*ffmpegCmd) Run(context.Context) error {
	code := ffmpeg.Execute(nil)
	if code != 0 {
		return fmt.Errorf("%w: exit %d", errFFmpeg, code)
	}
	return nil
}
