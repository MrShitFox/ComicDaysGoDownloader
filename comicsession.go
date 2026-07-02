package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// defaultUserAgent is sent with every request so the site does not reject the
// default Go HTTP client user agent.
const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

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
		if IsPermanent(err) {
			return nil, fmt.Errorf("could not load the page: %w", err)
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
	line, err := reader.ReadString('\n')
	url := strings.TrimSpace(line)
	// ReadString returns io.EOF together with the data when the input has no
	// trailing newline (e.g. when the URL is piped in). Only treat it as a
	// failure when nothing was read.
	if err != nil && !(errors.Is(err, io.EOF) && url != "") {
		return "", err
	}
	if url == "" {
		return "", fmt.Errorf("no URL was provided")
	}
	return url, nil
}

func fetchComicHTML(url string, cookies []Cookie, networkClient *NetworkClient) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, &PermanentError{Err: fmt.Errorf("error creating request: %v", err)}
	}

	req.Header.Set("User-Agent", defaultUserAgent)

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	resp, err := networkClient.FetchWithRetries(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the page: %w", err)
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

	// The pages array is already in reading order; keep it as-is. Sorting by the
	// CDN src would reorder pages if the opaque image IDs are not monotonic.
	return validPages, nil
}

func createOutputDir() (string, error) {
	dir := filepath.Join(".", time.Now().Format("2006-01-02-15-04-05"))
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}
	return dir, nil
}
