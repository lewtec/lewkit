package main

import (
	"bytes"
	"context"
	"net/http"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/http/asset"
	"github.com/lewtec/lewkit/x/http/middleware"
	"github.com/lewtec/lewkit/x/ui/web"
)

func spaHandler(ctx context.Context) (http.Handler, error) {
	var body bytes.Buffer
	if err := web.Page().Render(ctx, &body); err != nil {
		return nil, err
	}
	files := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: body.Bytes()},
	}
	return asset.Mount(middleware.SPA(files, nil)), nil
}
