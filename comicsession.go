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
	Cookies []Cookie
	Client  *http.Client
	URL     string
	Doc     *goquery.Document
	Pages   []Page
	OutDir  string
}

func (s *ComicSession) Init() error {
	var err error
	s.Cookies, err = NewFileCookieLoader("cookie.json").Load()
	if err != nil {
		// Proceed without cookies
		log.Printf("Warning: %v", err)
	}

	s.URL, err = readComicDaysURL()
	if err != nil {
		return err
	}

	s.Client = &http.Client{}
	s.Doc, err = fetchComicHTML(s.URL, s.Cookies, s.Client)
	if err != nil {
		return err
	}

	jsonData, err := extractEpisodeJSON(s.Doc)
	if err != nil {
		return err
	}

	s.Pages, err = parsePages(jsonData)
	if err != nil {
		return err
	}

	s.OutDir, err = createOutputDir()
	if err != nil {
		return err
	}

	return nil
}

func readComicDaysURL() (string, error) {
	fmt.Print("Please enter a manga link from comic-days website: ")
	reader := bufio.NewReader(os.Stdin)
	url, err := reader.ReadString('\n')
	return strings.TrimSpace(url), err
}

func fetchComicHTML(url string, cookies []Cookie, client *http.Client) (*goquery.Document, error) {
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching the webpage: %v", err)
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
