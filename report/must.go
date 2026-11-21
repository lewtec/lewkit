package report

func Must[T any](f func() (v T, err error)) T {
	v, err := f()
	if err != nil {
		Report(err)
		panic(err)
	}
	return v
}
