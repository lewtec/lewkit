package cmd

// CwdArg is a working directory for -C. It must already exist.
// Default is the process cwd. It does not create the path.
type CwdArg struct {
	Container[string]
}

func (CwdArg) ArgDefault() string { return "." }

func (c *CwdArg) Parse(arg string) error {
	if err := existingDir(arg); err != nil {
		return err
	}
	c.value = arg
	return nil
}

var (
	_ Parser       = (*CwdArg)(nil)
	_ Arg[string]  = (*CwdArg)(nil)
	_ ArgDefaulter = CwdArg{}
)
