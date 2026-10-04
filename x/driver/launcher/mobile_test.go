package launcher

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
)

func TestChoiceLinesFlattensNewlines(t *testing.T) {
	got := ChoiceLines([]Item{{Label: "a\nb"}, {Value: "c"}})
	if got != "a b\nc" {
		t.Fatalf("lines %q", got)
	}
}

func TestConfirmAnswer(t *testing.T) {
	ok, err := ConfirmAnswer(askwire.Format(askwire.StatusYes, ""))
	if err != nil || !ok {
		t.Fatalf("yes: %v %v", ok, err)
	}
	ok, err = ConfirmAnswer(askwire.Format(askwire.StatusNo, ""))
	if err != nil || ok {
		t.Fatalf("no: %v %v", ok, err)
	}
	_, err = ConfirmAnswer(askwire.Format(askwire.StatusNoActivity, ""))
	if !errors.Is(err, driver.ErrUnavailable) {
		t.Fatal(err)
	}
}

func TestPromptAnswer(t *testing.T) {
	got, err := PromptAnswer(askwire.Format(askwire.StatusOK, "ada"))
	if err != nil || got != "ada" {
		t.Fatalf("prompt %q %v", got, err)
	}
	_, err = PromptAnswer(askwire.Format(askwire.StatusCanceled, ""))
	if !errors.Is(err, ErrCanceled) {
		t.Fatal(err)
	}
}

func TestChooseAnswer(t *testing.T) {
	items := []Item{{Label: "One", Value: "1"}, {Label: "Two", Value: "2"}}
	got, err := ChooseAnswer(items, askwire.Format(askwire.StatusOK, "Two"))
	if err != nil || got == nil || got.Value != "2" {
		t.Fatalf("choose %+v %v", got, err)
	}
	got, err = ChooseAnswer(items, askwire.Format(askwire.StatusCanceled, ""))
	if err != nil || got != nil {
		t.Fatalf("cancel %+v %v", got, err)
	}
}
