package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/ffmpeg"
)

// errFFmpeg is the lewkit wrapper exit when ffmpeg returns a non-zero status.
var errFFmpeg = errs.Wrap(errs.ErrUnknown, "ffmpeg")

type ffmpegCmd struct{}

func (ffmpegCmd) Description() string {
	return "read, filter, and write media with the ffmpeg CLI"
}

func (*ffmpegCmd) Run(ctx context.Context) error {
	code := ffmpeg.Run(ctx, nil, os.Stdin, os.Stdout, os.Stderr)
	if code != 0 {
		return fmt.Errorf("%w: exit %d", errFFmpeg, code)
	}
	return nil
}
