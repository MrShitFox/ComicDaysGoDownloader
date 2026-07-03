package main

import (
	"net/http"
	"testing"
)

func TestNormalizeComicDaysURLAcceptsSchemelessURL(t *testing.T) {
	got, err := normalizeComicDaysURL("comic-days.com/episode/123#ignored")
	if err != nil {
		t.Fatalf("normalizeComicDaysURL returned error: %v", err)
	}
	want := "https://comic-days.com/episode/123"
	if got != want {
		t.Fatalf("normalizeComicDaysURL() = %q, want %q", got, want)
	}
}

func TestNormalizeComicDaysURLRejectsUntrustedHost(t *testing.T) {
	if _, err := normalizeComicDaysURL("https://example.com/episode/123"); err == nil {
		t.Fatal("normalizeComicDaysURL accepted an untrusted host")
	}
}

func TestAddCookiesOnlyForComicDaysHosts(t *testing.T) {
	cookies := []Cookie{{Name: "session", Value: "secret"}}

	trusted, err := http.NewRequest("GET", "https://comic-days.com/episode/1", nil)
	if err != nil {
		t.Fatal(err)
	}
	addCookies(trusted, cookies)
	if trusted.Header.Get("Cookie") == "" {
		t.Fatal("expected cookies on trusted Comic Days host")
	}

	untrusted, err := http.NewRequest("GET", "https://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}
	addCookies(untrusted, cookies)
	if got := untrusted.Header.Get("Cookie"); got != "" {
		t.Fatalf("unexpected cookies on untrusted host: %q", got)
	}
}
