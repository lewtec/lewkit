package sentry

import "testing"

func TestArgParse(t *testing.T) {
	t.Parallel()
	var a Arg
	if err := a.Parse("https://public@example.com/1"); err != nil {
		t.Fatal(err)
	}
	if got := a.Value(); got != "https://public@example.com/1" {
		t.Fatalf("Value() = %q", got)
	}
}

func TestArgSetupEmpty(t *testing.T) {
	t.Parallel()
	var a Arg
	if err := a.Setup(); err != nil {
		t.Fatal(err)
	}
}

func TestArgSetupBadDSN(t *testing.T) {
	t.Parallel()
	var a Arg
	if err := a.Parse("not-a-dsn"); err != nil {
		t.Fatal(err)
	}
	if err := a.Setup(); err == nil {
		t.Fatal("Setup(not-a-dsn) = nil error")
	}
}
