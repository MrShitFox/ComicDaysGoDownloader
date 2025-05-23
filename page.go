package main

import (
	"fmt"
	"image"
	"net/http"
	"path/filepath"

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

func (p *Page) Download(cookies []Cookie, pageNum int) (image.Image, error) {
	fmt.Printf("Downloading page %d...\n", pageNum)

	client := &http.Client{}
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

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading image for page %d: %v", pageNum, err)
	}
	defer resp.Body.Close()

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error decoding image for page %d: %v", pageNum, err)
	}

	if img == nil {
		return nil, fmt.Errorf("skipping page %d due to download error", pageNum)
	}

	fmt.Printf("Page %d downloaded successfully.\n", pageNum)
	return img, nil
}

func (p *Page) DeobfuscateAndSave(img image.Image, outDir string, pageNum int) error {
	fmt.Printf("Deobfuscating page %d...\n", pageNum)

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
