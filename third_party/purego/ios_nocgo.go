// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build !cgo && ios

package purego

// Upstream is_ios.go stops this build. iOS carries the darwin build tag,
// so the darwin fakecgo path is the one this tree keeps. A cgo-free iOS
// app can then link on a host that has no Apple SDK.
