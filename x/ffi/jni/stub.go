//go:build !android || !cgo

package jni

// SetCurrentEnv records nothing on this build.
func SetCurrentEnv(fn func() uintptr) { _ = fn }

func bind(env uintptr, anchor string) error {
	_, _ = env, anchor
	return ErrUnavailable
}

func callStatic(className, method string, args ...any) (any, error) {
	_, _, _ = className, method, args
	return nil, ErrUnavailable
}

func callNew(className string, args ...any) (*Ref, error) {
	_, _ = className, args
	return nil, ErrUnavailable
}

func callRef(r *Ref, method string, args ...any) (any, error) {
	_, _, _ = r, method, args
	return nil, ErrUnavailable
}

func release(r *Ref) bool {
	_ = r
	return true
}
