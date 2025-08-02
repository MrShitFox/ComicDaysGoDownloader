package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type ComicSession struct {
	Cookies       []Cookie
	NetworkClient *NetworkClient
	URL           string
	Doc           *goquery.Document
	Pages         []Page
	OutDir        string
}

func NewComicSession(cookieFile string) (*ComicSession, error) {
	cookies, err := NewFileCookieLoader(cookieFile).Load()
	if err != nil {
		log.Printf("Warning: %v", err)
		// Optional: You can continue without cookies, but you can also choose to stop.
	}

	url, err := readComicDaysURL()
	if err != nil {
		return nil, err
	}

	networkClient := NewNetworkClient(15 * time.Second)

	var doc *goquery.Document
	for {
		doc, err = fetchComicHTML(url, cookies, networkClient)
		if err == nil {
			break
		}
		log.Printf("Error during initial fetch: %v", err)
		log.Println("Retrying initial fetch in 10 seconds...")
		time.Sleep(10 * time.Second)
	}

	jsonData, err := extractEpisodeJSON(doc)
	if err != nil {
		return nil, err
	}

	pages, err := parsePages(jsonData)
	if err != nil {
		return nil, err
	}

	outDir, err := createOutputDir()
	if err != nil {
		return nil, err
	}

	return &ComicSession{
		Cookies:       cookies,
		NetworkClient: networkClient,
		URL:           url,
		Doc:           doc,
		Pages:         pages,
		OutDir:        outDir,
	}, nil
}

func readComicDaysURL() (string, error) {
	fmt.Print("Please enter a manga link from the comic-days website: ")
	reader := bufio.NewReader(os.Stdin)
	url, err := reader.ReadString('\n')
	return strings.TrimSpace(url), err
}

func fetchComicHTML(url string, cookies []Cookie, networkClient *NetworkClient) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	resp, err := networkClient.FetchWithRetries(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request after 5 attempts: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing the webpage: %v", err)
	}

	return doc, nil
}

func extractEpisodeJSON(doc *goquery.Document) (string, error) {
	jsonData, exists := doc.Find("#episode-json").Attr("data-value")
	if !exists {
		return "", fmt.Errorf("could not find episode data on the page")
	}
	jsonData = html.UnescapeString(jsonData)
	if jsonData == "" {
		return "", fmt.Errorf("episode data is empty")
	}
	return jsonData, nil
}

func parsePages(jsonData string) ([]Page, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return nil, fmt.Errorf("error parsing JSON data: %v", err)
	}

	readableProduct, ok := data["readableProduct"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid JSON structure: missing readableProduct")
	}
	pageStructure, ok := readableProduct["pageStructure"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid JSON structure: missing pageStructure")
	}
	pages, ok := pageStructure["pages"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid JSON structure: missing pages")
	}

	var validPages []Page
	for _, p := range pages {
		page, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		src, ok := page["src"].(string)
		width, okW := page["width"].(float64)
		height, okH := page["height"].(float64)
		if ok && okW && okH && src != "" {
			validPages = append(validPages, NewPage(
				src,
				int(width),
				int(height),
			))
		}
	}

	sort.Slice(validPages, func(i, j int) bool {
		return validPages[i].Src < validPages[j].Src
	})

	return validPages, nil
}

func createOutputDir() (string, error) {
	dir := filepath.Join(".", time.Now().Format("2006-01-02-15-04-05"))
	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}
	return dir, nil
}