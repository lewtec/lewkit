// Package gui is a bubbletea-shaped loop whose View is a pixel tensor.
//
// [Model] is Init, Update, View. View returns a fused (h, w, 4) uint8
// tensor. [Run] drives a caller-supplied [window.Window]; it does not
// call [window.Open]. Host events are Resize, Expose, and Close.
// Pointer and key events are not on the bus.
package gui
