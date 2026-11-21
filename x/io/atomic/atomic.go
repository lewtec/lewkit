package atomic

import (
	"errors"
	"fmt"
	"io"
	"os"
	"uuid"

	"github.com/lewtec/lewkit/report"
)

type Operation struct {
	finalPath string
	tempPath  string
	replace   bool
}

func (o *Operation) StagingPath() string {
	return o.tempPath
}

func (o *Operation) Commit() error {
	displaced := ""
	if o.replace {
		displaced = tempOf(o.finalPath)
		err := os.Rename(o.finalPath, displaced)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			displaced = ""
		}
	}
	err := os.Rename(o.tempPath, o.finalPath)
	if err != nil {
		restoreErr := error(nil)
		if displaced != "" {
			restoreErr = os.Rename(displaced, o.finalPath)
		}
		return errors.Join(err, os.Rename(o.finalPath, o.tempPath), restoreErr)
	}
	if displaced == "" {
		return nil
	}
	return os.RemoveAll(displaced)
}

func (o *Operation) Rollback() error {
	return os.RemoveAll(o.tempPath)
}

func NewOperation(location string, replace bool) Operation {
	return Operation{
		finalPath: location,
		tempPath:  tempOf(location),
		replace:   replace,
	}
}

func tempOf(path string) string {
	return fmt.Sprintf("%s.%s", path, uuid.NewV7())
}

func Create(path string, mode os.FileMode) error {
	op := NewOperation(path, true)
	defer op.Rollback()
	f, err := os.Create(op.StagingPath())
	defer report.Report(f.Close())
	if err != nil {
		return err
	}
	if err := os.Chmod(op.StagingPath(), mode); err != nil {
		return err
	}
	return op.Commit()
}

func WriteFileFunction(path string, f func(w io.Writer) error) error {
	op := NewOperation(path, true)
	defer op.Rollback()
	w, err := os.Create(op.StagingPath())
	if err != nil {
		return err
	}
	if err := f(w); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return op.Commit()
}

func WriteString(path string, data string) error {
	return WriteFileFunction(path, func(w io.Writer) error {
		_, err := io.WriteString(w, data)
		return err
	})
}
