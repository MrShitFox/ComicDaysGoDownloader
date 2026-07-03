package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	maxRetries = 5
	baseDelay  = 1 * time.Second
)

// RetryObserver is notified before FetchWithRetries sleeps and retries a
// request, so callers can surface progress in their own UI (a spinner, a log
// line, ...) instead of the network layer printing directly. It is also
// called, with delay 0, when a client-side timeout forces an immediate,
// non-retrying failure. A nil observer simply disables notifications.
type RetryObserver func(attempt, maxAttempts int, err error, delay time.Duration)

// PermanentError marks a failure that will not be resolved by retrying (for
// example an HTTP 4xx response or a malformed request). Callers can detect it
// with errors.As to stop retrying instead of looping forever.
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string { return e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// IsPermanent reports whether err (or anything it wraps) is a PermanentError.
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

type NetworkClient struct {
	client *http.Client
}

func NewNetworkClient(timeout time.Duration) *NetworkClient {
	return &NetworkClient{
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// FetchWithRetries performs the request, retrying transient failures with
// exponential backoff. A successful call returns a response with a 2xx status
// whose Body the caller must close. Persistent 4xx responses (except 429) are
// returned as a *PermanentError so callers can stop retrying. onRetry may be
// nil.
func (nc *NetworkClient) FetchWithRetries(req *http.Request, onRetry RetryObserver) (*http.Response, error) {
	var lastErr error
	lastPermanent := false

	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := nc.client.Do(req)
		switch {
		case err != nil:
			if isTimeout(err) {
				// The overall client timeout elapsed. Retrying within the same
				// short-lived client is unlikely to help, so fail fast and let
				// the higher level decide whether to try again.
				if onRetry != nil {
					onRetry(attempt+1, maxRetries, err, 0)
				}
				return nil, err
			}
			lastErr = err
			lastPermanent = false
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			return resp, nil
		default:
			// Non-2xx response: drain and close the body so the connection can
			// be reused, then classify the status code.
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			status := resp.StatusCode
			lastErr = fmt.Errorf("server returned HTTP %d %s", status, http.StatusText(status))
			lastPermanent = status >= 400 && status < 500 && status != http.StatusTooManyRequests
		}

		if attempt < maxRetries-1 {
			delay := baseDelay * time.Duration(1<<attempt)
			if onRetry != nil {
				onRetry(attempt+1, maxRetries, lastErr, delay)
			}
			time.Sleep(delay)
		}
	}

	err := fmt.Errorf("failed to execute request after %d attempts: %w", maxRetries, lastErr)
	if lastPermanent {
		return nil, &PermanentError{Err: err}
	}
	return nil, err
}

// isTimeout reports whether err was caused by a timeout, covering both the
// http.Client.Timeout deadline and lower level network timeouts.
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}
