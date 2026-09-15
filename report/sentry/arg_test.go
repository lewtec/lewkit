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
	if a.rep == nil {
		t.Fatal("Parse did not hold a reporter")
	}
}

func TestArgParseEmpty(t *testing.T) {
	t.Parallel()
	var a Arg
	if err := a.Parse(""); err != nil {
		t.Fatal(err)
	}
	if err := a.Setup(); err != nil {
		t.Fatal(err)
	}
}

func TestArgParseBadDSN(t *testing.T) {
	t.Parallel()
	var a Arg
	if err := a.Parse("not-a-dsn"); err == nil {
		t.Fatal("Parse(not-a-dsn) = nil error")
	}
}
