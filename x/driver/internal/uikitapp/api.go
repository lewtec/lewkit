package uikitapp

// PickRequest is one document picker.
type PickRequest struct {
	Save      bool
	Folder    bool
	Multiple  bool
	Name      string
	Directory string
	Exts      []string
}
