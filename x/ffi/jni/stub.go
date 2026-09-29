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

func classObject(name string) (*Ref, error) {
	_ = name
	return nil, ErrUnavailable
}

func staticField(className, name string) (any, error) {
	_, _ = className, name
	return nil, ErrUnavailable
}

func instanceField(r *Ref, name string) (any, error) {
	_, _ = r, name
	return nil, ErrUnavailable
}

func makeProxy(iface string, invoke func(string, []any) (any, error)) (*Ref, error) {
	_, _ = iface, invoke
	return nil, ErrUnavailable
}

func release(r *Ref) bool {
	_ = r
	return true
}
