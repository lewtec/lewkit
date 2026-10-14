package launcher

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/stretchr/testify/require"
)

func TestChoiceLinesFlattensNewlines(t *testing.T) {
	got := ChoiceLines([]Item{{Label: "a\nb"}, {Value: "c"}})
	if got != "a b\nc" {
		require.FailNowf(t, "", "lines %q", got)
	}
}

func TestConfirmAnswer(t *testing.T) {
	ok, err := ConfirmAnswer(askwire.Format(askwire.StatusYes, ""))
	if err != nil || !ok {
		require.FailNowf(t, "", "yes: %v %v", ok, err)
	}
	ok, err = ConfirmAnswer(askwire.Format(askwire.StatusNo, ""))
	if err != nil || ok {
		require.FailNowf(t, "", "no: %v %v", ok, err)
	}
	_, err = ConfirmAnswer(askwire.Format(askwire.StatusNoActivity, ""))
	require.ErrorIs(t, err, driver.ErrUnavailable)
}

func TestPromptAnswer(t *testing.T) {
	got, err := PromptAnswer(askwire.Format(askwire.StatusOK, "ada"))
	if err != nil || got != "ada" {
		require.FailNowf(t, "", "prompt %q %v", got, err)
	}
	_, err = PromptAnswer(askwire.Format(askwire.StatusCanceled, ""))
	require.ErrorIs(t, err, ErrCanceled)
}

func TestChooseAnswer(t *testing.T) {
	items := []Item{{Label: "One", Value: "1"}, {Label: "Two", Value: "2"}}
	got, err := ChooseAnswer(items, askwire.Format(askwire.StatusOK, "Two"))
	if err != nil || got == nil || got.Value != "2" {
		require.FailNowf(t, "", "choose %+v %v", got, err)
	}
	got, err = ChooseAnswer(items, askwire.Format(askwire.StatusCanceled, ""))
	if err != nil || got != nil {
		require.FailNowf(t, "", "cancel %+v %v", got, err)
	}
}
