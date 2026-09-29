// Package android sets the foreground window brightness.
// DisplayManager.setBrightness needs a signature permission. The driver
// writes WindowManager.LayoutParams.screenBrightness, which Android applies
// while that window is in front. A window still following the system setting
// is read from Settings.System.SCREEN_BRIGHTNESS.
package android
