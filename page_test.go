package main

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fakeFetcher struct {
	resp  *http.Response
	err   error
	req   *http.Request
	calls int
}

func (f *fakeFetcher) FetchWithRetries(req *http.Request, onRetry RetryObserver) (*http.Response, error) {
	f.calls++
	f.req = req
	return f.resp, f.err
}

func TestDownloadAttemptRejectsUntrustedPageSource(t *testing.T) {
	fetcher := &fakeFetcher{}
	page := NewPage("https://example.com/image.png", 1, 1)

	_, _, err := page.downloadAttempt(fetcher, nil, 1, nil)
	if err == nil {
		t.Fatal("downloadAttempt accepted an untrusted page source")
	}
	if !IsPermanent(err) {
		t.Fatalf("error is not permanent: %v", err)
	}
	if fetcher.calls != 0 {
		t.Fatalf("fetcher calls = %d, want 0", fetcher.calls)
	}
}

func TestDownloadAttemptRejectsNonImageContentType(t *testing.T) {
	fetcher := &fakeFetcher{
		resp: testResponse("text/html; charset=utf-8", strings.NewReader("<html></html>")),
	}
	page := NewPage("https://cdn.comic-days.com/image.png", 1, 1)

	_, _, err := page.downloadAttempt(fetcher, nil, 1, nil)
	if err == nil {
		t.Fatal("downloadAttempt accepted a non-image response")
	}
	if !IsPermanent(err) {
		t.Fatalf("error is not permanent: %v", err)
	}
}

func TestDownloadAttemptRejectsDimensionMismatch(t *testing.T) {
	fetcher := &fakeFetcher{
		resp: testResponse("image/png", bytes.NewReader(testPNG(t, 2, 2))),
	}
	page := NewPage("https://cdn.comic-days.com/image.png", 1, 1)

	_, _, err := page.downloadAttempt(fetcher, nil, 1, nil)
	if err == nil {
		t.Fatal("downloadAttempt accepted mismatched dimensions")
	}
	if !IsPermanent(err) {
		t.Fatalf("error is not permanent: %v", err)
	}
}

func testResponse(contentType string, body io.Reader) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(body),
	}
}

func testPNG(t *testing.T, width, height int) []byte {
	t.Helper()
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
