package cocoa

// alertFirstButton is NSAlertFirstButtonReturn.
const alertFirstButton = 1000

// buttonIndex maps an NSAlert response onto a choice button.
// The cancel button is past count, so it is not a choice.
func buttonIndex(response, count int) (int, bool) {
	index := response - alertFirstButton
	if count <= 0 || index < 0 || index >= count {
		return 0, false
	}
	return index, true
}
