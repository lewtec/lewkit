package path

// StripTopLevelDirectory moves children up when root contains exactly one directory.
// A root with any other shape is left unchanged.
func StripTopLevelDirectory(root *Root) error {
	entries, err := New(".").ReadDir(root)
	if err != nil {
		return err
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		return nil
	}
	child := New(entries[0].Name())
	children, err := child.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range children {
		from := child.Join(entry.Name())
		if err := from.Rename(root, New(entry.Name())); err != nil {
			return err
		}
	}
	return child.RemoveAll(root)
}
