package main

import (
	"fmt"
	"image"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

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

func (p Page) Process(networkClient *NetworkClient, cookies []Cookie, outDir string, pageNum int) {
	var img image.Image
	var err error

	localNetworkClient := networkClient

	for {
		fmt.Printf("Downloading page %d...\n", pageNum)
		img, err = p.downloadAttempt(localNetworkClient, cookies, pageNum)
		if err == nil {
			break
		}

		log.Printf("Failed to download page %d: %v", pageNum, err)

		if strings.Contains(err.Error(), "context deadline exceeded") {
			log.Println("Critical timeout detected. Resetting the network client for the next attempt.")
			localNetworkClient = NewNetworkClient(15 * time.Second)
		}

		log.Printf("Will retry page %d in 10 seconds...", pageNum)
		time.Sleep(10 * time.Second)
	}

	fmt.Printf("Deobfuscating page %d...\n", pageNum)
	err = p.deobfuscateAndSave(img, outDir, pageNum)
	if err != nil {
		log.Printf("Warning: Could not save page %d: %v", pageNum, err)
	}
}

func (p Page) downloadAttempt(networkClient *NetworkClient, cookies []Cookie, pageNum int) (image.Image, error) {
	req, err := http.NewRequest("GET", p.Src, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request for page %d: %v", pageNum, err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	req.Header.Set("Referer", "https://comic-days.com/")

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	resp, err := networkClient.FetchWithRetries(req)
	if err != nil {
		return nil, fmt.Errorf("all retry attempts failed: %v", err)
	}
	defer resp.Body.Close()

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding image: %v", err)
	}

	if img == nil {
		return nil, fmt.Errorf("downloaded image is empty")
	}

	fmt.Printf("Page %d downloaded successfully.\n", pageNum)
	return img, nil
}

func (p Page) deobfuscateAndSave(img image.Image, outDir string, pageNum int) error {
	filePath := filepath.Join(outDir, fmt.Sprintf("%03d.png", pageNum))
	imageCtx := NewImageContext(img)
	imageCtx.Deobfuscate(p.Width, p.Height)
	rightTransparentWidth := imageCtx.DetectTransparentStripWidth()
	fmt.Printf("Detected transparent right strip width for page %d: %d pixels\n", pageNum, rightTransparentWidth)

	imageCtx.RestoreRightTransparentStrip(p.Width, p.Height, rightTransparentWidth)

	err := imageCtx.SaveImage(filePath)
	if err != nil {
		return fmt.Errorf("error creating file for page %d: %v", pageNum, err)
	}

	fmt.Printf("Page %d deobfuscated and saved.\n", pageNum)
	return nil
}