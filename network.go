package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	maxRetries = 5
	baseDelay  = 1 * time.Second
)

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

func (nc *NetworkClient) FetchWithRetries(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, err = nc.client.Do(req)
		if err == nil {
			return resp, nil
		}

		if strings.Contains(err.Error(), "context deadline exceeded") {
			log.Println("Timeout error detected in network layer. Failing fast.")
			return nil, err
		}

		delay := baseDelay * time.Duration(1<<i)
		log.Printf("Request failed (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, delay)
		time.Sleep(delay)
	}

	return nil, fmt.Errorf("failed to execute request after %d attempts: %v", maxRetries, err)
}