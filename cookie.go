package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

type Cookie struct {
	Domain         string  `json:"domain"`
	ExpirationDate float64 `json:"expirationDate"`
	HostOnly       bool    `json:"hostOnly"`
	HTTPOnly       bool    `json:"httpOnly"`
	Name           string  `json:"name"`
	Path           string  `json:"path"`
	SameSite       string  `json:"sameSite"`
	Secure         bool    `json:"secure"`
	Session        bool    `json:"session"`
	StoreID        string  `json:"storeId"`
	Value          string  `json:"value"`
}

type CookieLoader interface {
	Load() ([]Cookie, error)
}

type FileCookieLoader struct {
	Filename string
}

func NewFileCookieLoader(filename string) FileCookieLoader {
	return FileCookieLoader{Filename: filename}
}

func (f FileCookieLoader) Load() ([]Cookie, error) {
	file, err := os.Open(f.Filename)
	if err != nil {
		return nil, fmt.Errorf("could not open cookie file: %v", err)
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("could not read cookie file: %v", err)
	}

	var cookies []Cookie
	if err := json.Unmarshal(bytes, &cookies); err != nil {
		return nil, fmt.Errorf("could not parse cookie file: %v", err)
	}

	return cookies, nil
}
