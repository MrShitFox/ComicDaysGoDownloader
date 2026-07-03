package main

import "testing"

func TestParsePagesValidatesAndKeepsOrder(t *testing.T) {
	jsonData := `{
		"readableProduct": {
			"pageStructure": {
				"pages": [
					{"src": "https://cdn.comic-days.com/images/2.png", "width": 2, "height": 3},
					{"src": "//img.comic-days.com/images/1.png", "type": "main", "width": 4, "height": 5},
					{"type": "link"},
					{"type": "other"},
					{"type": "backMatter"}
				]
			}
		}
	}`

	pages, err := parsePages(jsonData)
	if err != nil {
		t.Fatalf("parsePages returned error: %v", err)
	}
	if len(pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(pages))
	}
	if pages[0].Src != "https://cdn.comic-days.com/images/2.png" {
		t.Fatalf("first page src = %q", pages[0].Src)
	}
	if pages[1].Src != "https://img.comic-days.com/images/1.png" {
		t.Fatalf("second page src = %q", pages[1].Src)
	}
}

func TestParsePagesRejectsUntrustedImageHost(t *testing.T) {
	jsonData := `{
		"readableProduct": {
			"pageStructure": {
				"pages": [
					{"src": "https://example.com/images/1.png", "width": 2, "height": 3}
				]
			}
		}
	}`

	if _, err := parsePages(jsonData); err == nil {
		t.Fatal("parsePages accepted an untrusted image host")
	}
}

func TestParsePagesRejectsInvalidDimensions(t *testing.T) {
	jsonData := `{
		"readableProduct": {
			"pageStructure": {
				"pages": [
					{"src": "https://cdn.comic-days.com/images/1.png", "width": 0, "height": 3}
				]
			}
		}
	}`

	if _, err := parsePages(jsonData); err == nil {
		t.Fatal("parsePages accepted invalid dimensions")
	}
}
