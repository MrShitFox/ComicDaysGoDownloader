package main

import (
	"fmt"
	"image"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/disintegration/imaging"
)

const retryDelay = 10 * time.Second

type Page struct {
	Src    string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func NewPage(src string, width, height int) Page {
	return Page{
		Src:    src,
		Width:  width,
		Height: height,
	}
}

// Process downloads, deobfuscates and saves a single page. It retries transient
// failures indefinitely but gives up on permanent errors (for example a page
// that requires a purchase) instead of looping forever. It returns an error
// only when the page could not be produced.
func (p Page) Process(networkClient *NetworkClient, cookies []Cookie, outDir string, pageNum int) error {
	var img image.Image
	var err error

	for attempt := 1; ; attempt++ {
		fmt.Printf("Downloading page %d...\n", pageNum)
		img, err = p.downloadAttempt(networkClient, cookies, pageNum)
		if err == nil {
			break
		}

		if IsPermanent(err) {
			log.Printf("Giving up on page %d: %v", pageNum, err)
			return fmt.Errorf("page %d: %w", pageNum, err)
		}

		log.Printf("Failed to download page %d (attempt %d): %v", pageNum, attempt, err)
		log.Printf("Will retry page %d in %v...", pageNum, retryDelay)
		time.Sleep(retryDelay)
	}

	fmt.Printf("Deobfuscating page %d...\n", pageNum)
	if err := p.deobfuscateAndSave(img, outDir, pageNum); err != nil {
		log.Printf("Warning: Could not save page %d: %v", pageNum, err)
		return fmt.Errorf("page %d: %w", pageNum, err)
	}
	return nil
}

func (p Page) downloadAttempt(networkClient *NetworkClient, cookies []Cookie, pageNum int) (image.Image, error) {
	req, err := http.NewRequest("GET", p.Src, nil)
	if err != nil {
		return nil, &PermanentError{Err: fmt.Errorf("error creating request for page %d: %v", pageNum, err)}
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", "https://comic-days.com/")
	req.Header.Set("Origin", "https://comic-days.com")

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	resp, err := networkClient.FetchWithRetries(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download page %d: %w", pageNum, err)
	}
	defer resp.Body.Close()

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding image for page %d: %v", pageNum, err)
	}

	if img == nil {
		return nil, fmt.Errorf("downloaded image for page %d is empty", pageNum)
	}

	fmt.Printf("Page %d downloaded successfully.\n", pageNum)
	return img, nil
}

func (p Page) deobfuscateAndSave(img image.Image, outDir string, pageNum int) error {
	filePath := filepath.Join(outDir, fmt.Sprintf("%03d.png", pageNum))
	imageCtx := NewImageContext(img)
	imageCtx.Deobfuscate(p.Width, p.Height)

	if err := imageCtx.SaveImage(filePath); err != nil {
		return fmt.Errorf("error creating file for page %d: %v", pageNum, err)
	}

	fmt.Printf("Page %d deobfuscated and saved.\n", pageNum)
	return nil
}
