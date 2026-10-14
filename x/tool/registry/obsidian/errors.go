package pkg

import (
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/tool/registry/internal/install"
)

var (
	ErrNoPlatformArtifact = install.ErrNoPlatformArtifact
	ErrDecodeRelease      = errs.Wrap(errs.ErrInvalid, "decode release")
)
