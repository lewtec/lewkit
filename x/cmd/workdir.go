package cmd

// WorkDirArg is a working directory for -C. It must already exist.
// Default is the process cwd. It does not create the path.
type WorkDirArg struct {
	Container[string]
}

func (WorkDirArg) ArgDefault() string { return "." }

func (w *WorkDirArg) Parse(arg string) error {
	if err := existingDir(arg); err != nil {
		return err
	}
	w.value = arg
	return nil
}

var (
	_ Parser       = (*WorkDirArg)(nil)
	_ Arg[string]  = (*WorkDirArg)(nil)
	_ ArgDefaulter = WorkDirArg{}
)
