package taskgroup

import "bytes"

// TakeLines appends p to pending and calls emit for each newline-terminated record.
func TakeLines(pending *[]byte, p []byte, emit func(string)) {
	*pending = append(*pending, p...)
	for {
		i := bytes.IndexByte(*pending, '\n')
		if i < 0 {
			return
		}
		line := bytes.TrimSuffix((*pending)[:i], []byte{'\r'})
		*pending = (*pending)[i+1:]
		if emit != nil {
			emit(string(line))
		}
	}
}
