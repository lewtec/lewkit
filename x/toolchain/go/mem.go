package gotool

import "os"

// memFile is a file the compiler opens by path.
// Linux uses a memfd. Darwin and Windows use a temporary file whose
// name does not end in .go, so this package builds there.
type memFile struct {
	f    *os.File
	path string
}

// Path is the name a compiler or linker process opens.
func (m *memFile) Path() string { return m.path }

// Write appends bytes to the file.
func (m *memFile) Write(p []byte) (int, error) { return m.f.Write(p) }

// Close releases the descriptor.
func (m *memFile) Close() error { return m.f.Close() }

// File is the open descriptor. RunSource passes the Linux binary to exec.
func (m *memFile) File() *os.File { return m.f }
