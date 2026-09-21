package tool

import "context"

type noCacheKey struct{}
type dryRunKey struct{}

// WithNoCache marks ctx so Ensure and Install ignore a warm version directory.
func WithNoCache(ctx context.Context) context.Context {
	return context.WithValue(ctx, noCacheKey{}, true)
}

// NoCache reports whether WithNoCache is set on ctx.
func NoCache(ctx context.Context) bool {
	return contextFlag(ctx, noCacheKey{})
}

// WithDryRun marks ctx so a no-cache Ensure leaves an existing tree in place.
func WithDryRun(ctx context.Context) context.Context {
	return context.WithValue(ctx, dryRunKey{}, true)
}

// DryRun reports whether WithDryRun is set on ctx.
func DryRun(ctx context.Context) bool {
	return contextFlag(ctx, dryRunKey{})
}

func contextFlag(ctx context.Context, key any) bool {
	value, ok := ctx.Value(key).(bool)
	return ok && value
}
