package share

import (
	"errors"
	"testing"
)

func TestValidate_Empty(t *testing.T) {
	err := Item{}.validate()
	if !errors.Is(err, ErrEmptyItem) {
		t.Fatal(err)
	}
}

func TestValidate_RelativePath(t *testing.T) {
	err := Item{Paths: []string{"rel"}}.validate()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_Text(t *testing.T) {
	if err := (Item{Text: "hi"}).validate(); err != nil {
		t.Fatal(err)
	}
}
