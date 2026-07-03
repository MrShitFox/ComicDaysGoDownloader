package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const comicDaysHost = "comic-days.com"

func normalizeComicDaysURL(raw string) (string, error) {
	return normalizeTrustedHTTPSURL(raw, "URL")
}

func normalizeComicDaysAssetURL(raw string) (string, error) {
	return normalizeTrustedHTTPSURL(raw, "page image URL")
}

func normalizeTrustedHTTPSURL(raw, label string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("%s is empty", label)
	}
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	} else if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid %s: %w", label, err)
	}
	if u.Scheme != "https" {
		return "", fmt.Errorf("%s must use https", label)
	}
	if !isComicDaysHost(u.Hostname()) {
		return "", fmt.Errorf("%s host must be %s or its subdomain", label, comicDaysHost)
	}
	if u.Path == "" || u.Path == "/" {
		return "", fmt.Errorf("%s must include a path", label)
	}
	u.Fragment = ""
	return u.String(), nil
}

func isComicDaysHost(host string) bool {
	host = strings.TrimSuffix(strings.ToLower(host), ".")
	return host == comicDaysHost || strings.HasSuffix(host, "."+comicDaysHost)
}

func addCookies(req *http.Request, cookies []Cookie) {
	if req == nil || req.URL == nil || !isComicDaysHost(req.URL.Hostname()) {
		return
	}
	for _, cookie := range cookies {
		if cookie.Name == "" {
			continue
		}
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}
}
