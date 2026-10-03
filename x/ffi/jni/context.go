package jni

import "errors"

var errNoContext = errors.New("application context")

// Context returns the application context stored on lewkit.Host.
// The caller releases the reference.
func Context() (*Ref, error) {
	app, err := AsRef(StaticField("lewkit.Host", "app"))
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errNoContext
	}
	return app, nil
}
