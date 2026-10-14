package win32

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDialogTemplateAlignsControls(t *testing.T) {
	body := promptDialog("Name", "Ada").bytes()
	if len(body) < 18 {
		require.FailNowf(t, "", "short template %d", len(body))
	}
	count := binary.LittleEndian.Uint16(body[8:10])
	if count != 4 {
		require.FailNowf(t, "", "controls %d", count)
	}
	width := int16(binary.LittleEndian.Uint16(body[14:16]))
	if width != 220 {
		require.FailNowf(t, "", "width %d", width)
	}
	// The first control starts on a 4-byte boundary and carries the edit id.
	found := false
	for i := 18; i+18 <= len(body); i++ {
		if i%4 != 0 {
			continue
		}
		id := binary.LittleEndian.Uint16(body[i+16 : i+18])
		if id == idField {
			found = true
			break
		}
	}
	if !found {
		require.FailNow(t, fmt.Sprint("edit control missing"))
	}
}

func TestChooseDialogNamesTheList(t *testing.T) {
	body := chooseDialog("Open").bytes()
	count := binary.LittleEndian.Uint16(body[8:10])
	if count != 4 {
		require.FailNowf(t, "", "controls %d", count)
	}
}
