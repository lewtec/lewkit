package compose

import (
	"bytes"
	"fmt"
	"io"
	iofs "io/fs"
	"maps"
	"slices"
)

// Encode writes the combined bytes of one file.
// base opens [Ref] slots. A nil base rejects those slots.
func Encode(file File, base iofs.FS) ([]byte, error) {
	normalized, err := file.normalized()
	if err != nil {
		return nil, err
	}
	if err := normalized.check(); err != nil {
		return nil, err
	}
	if normalized.Type.structured() {
		return encodeStructured(normalized.Type, normalized.Data)
	}
	return encodeSlots(normalized, base)
}

func encodeSlots(file File, base iofs.FS) ([]byte, error) {
	keys := slices.Sorted(maps.Keys(file.Values))
	var buffer bytes.Buffer
	for index, key := range keys {
		if index > 0 {
			if err := buffer.WriteByte('\n'); err != nil {
				return nil, err
			}
		}
		if err := file.Values[key].writeTo(&buffer, base); err != nil {
			return nil, fmt.Errorf("values.%s: %w", key, err)
		}
	}
	return buffer.Bytes(), nil
}

func (slot Slot) writeTo(writer io.Writer, base iofs.FS) error {
	switch slot.kind {
	case slotText:
		_, err := io.WriteString(writer, slot.body)
		return err
	case slotLink:
		_, err := io.WriteString(writer, slot.body)
		return err
	case slotRef:
		if base == nil {
			return fmt.Errorf("%w: nil base", ErrRef)
		}
		opened, err := base.Open(slot.body)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrRef, err)
		}
		_, copyErr := io.Copy(writer, opened)
		closeErr := opened.Close()
		if copyErr != nil {
			return fmt.Errorf("%w: %w", ErrRef, copyErr)
		}
		if closeErr != nil {
			return fmt.Errorf("%w: %w", ErrRef, closeErr)
		}
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrSlot, slot.kind)
	}
}
