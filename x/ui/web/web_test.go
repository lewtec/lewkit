package web

import (
	"bytes"
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/http/asset/htmx"
	"github.com/lewtec/lewkit/x/http/asset/jquery"
	"github.com/lewtec/lewkit/x/http/asset/sakuracss"
	"github.com/lewtec/lewkit/x/http/asset/tailwindcss"
	"github.com/stretchr/testify/require"
)

func TestTemplates(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		render func(context.Context, *bytes.Buffer) error
		needle string
	}{
		{name: "htmx", render: func(ctx context.Context, buf *bytes.Buffer) error { return HTMX().Render(ctx, buf) }, needle: htmx.Path},
		{name: "jquery", render: func(ctx context.Context, buf *bytes.Buffer) error { return JQuery().Render(ctx, buf) }, needle: jquery.Path},
		{name: "tailwindcss", render: func(ctx context.Context, buf *bytes.Buffer) error { return TailwindCSS().Render(ctx, buf) }, needle: tailwindcss.Path},
		{name: "sakuracss", render: func(ctx context.Context, buf *bytes.Buffer) error { return SakuraCSS().Render(ctx, buf) }, needle: sakuracss.Path},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			require.NoError(t, tt.render(t.Context(), &buf))
			require.Contains(t, buf.String(), tt.needle)
		})
	}
}
