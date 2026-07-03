package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchWithRetriesDoesNotRetryPermanent4xx(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	t.Cleanup(server.Close)

	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := NewNetworkClient(time.Second)
	resp, err := client.FetchWithRetries(req, nil)
	if resp != nil {
		t.Fatalf("response = %#v, want nil", resp)
	}
	if err == nil {
		t.Fatal("FetchWithRetries returned nil error")
	}
	if !IsPermanent(err) {
		t.Fatalf("error is not permanent: %v", err)
	}
	if calls != 1 {
		t.Fatalf("server calls = %d, want 1", calls)
	}
}
