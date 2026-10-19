// Package uikit opens a UIView on iOS.
//
// The view is the host surface. Metal presents into it. When no host window
// exists, this process opens a UIWindow and a surface view. That view reports
// touch, hardware keys, size, and the safe area through the entry package.
// A Swift host can still hand over its own view with openSurface.
package uikit
