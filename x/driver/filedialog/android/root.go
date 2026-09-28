// Package android shows the system document picker.
//
// Open and folder results are filesystem paths. A shared-storage document
// the process can already read is returned as that path. Otherwise the
// bytes are copied into the app cache. Save returns a shared-storage path
// when the document id names one.
package android
