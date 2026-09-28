package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHomeListsADriver(t *testing.T) {
	srv := httptest.NewServer(newPage(t.Context()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "Open triangle") {
		t.Fatal("missing button")
	}
	if !strings.Contains(string(body), "<h2>") {
		t.Fatal("missing driver section")
	}
	if strings.Contains(string(body), "location.href") || strings.Contains(string(body), "/triangle/view") {
		t.Fatal("triangle replaces the drivers page")
	}
}
