package main

import (
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
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

// countingReader tallies how many bytes have passed through it, which lets
// the UI report real download sizes even though the image is decoded
// straight from the streaming HTTP response body.
type countingReader struct {
	r     io.Reader
	count int64
}

func (c *countingReader) Read(buf []byte) (int, error) {
	n, err := c.r.Read(buf)
	c.count += int64(n)
	return n, err
}

// Process downloads, deobfuscates and saves a single page, narrating every
// step through pl. It retries transient failures indefinitely but gives up
// on permanent errors (for example a page that requires a purchase) instead
// of looping forever. It returns an error only when the page could not be
// produced; pl has already reported success or failure by the time it does.
func (p Page) Process(networkClient *NetworkClient, cookies []Cookie, outDir string, pageNum int, pl *Pipeline) error {
	start := time.Now()

	var img image.Image
	var downloadedBytes int64
	var err error

	for attempt := 1; ; attempt++ {
		pl.Status(pageNum, "downloading...")
		img, downloadedBytes, err = p.downloadAttempt(networkClient, cookies, pageNum, pl)
		if err == nil {
			break
		}

		if IsPermanent(err) {
			pl.PageFailed(pageNum, err)
			return fmt.Errorf("page %d: %w", pageNum, err)
		}

		pl.Status(pageNum, "download failed (attempt %d): %v — retrying in %v...", attempt, err, retryDelay)
		time.Sleep(retryDelay)
	}

	pl.Status(pageNum, "reversing %dx%d grid transpose...", divideNum, divideNum)
	savedBytes, err := p.deobfuscateAndSave(img, outDir, pageNum)
	if err != nil {
		pl.PageFailed(pageNum, err)
		return fmt.Errorf("page %d: %w", pageNum, err)
	}

	pl.PageSucceeded(pageResult{
		pageNum:       pageNum,
		width:         p.Width,
		height:        p.Height,
		downloadBytes: downloadedBytes,
		savedBytes:    savedBytes,
		elapsed:       time.Since(start),
	})
	return nil
}

func (p Page) downloadAttempt(networkClient *NetworkClient, cookies []Cookie, pageNum int, pl *Pipeline) (image.Image, int64, error) {
	req, err := http.NewRequest("GET", p.Src, nil)
	if err != nil {
		return nil, 0, &PermanentError{Err: fmt.Errorf("error creating request for page %d: %v", pageNum, err)}
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

	resp, err := networkClient.FetchWithRetries(req, pl.RetryObserver(pageNum, "download"))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to download page %d: %w", pageNum, err)
	}
	defer resp.Body.Close()

	counting := &countingReader{r: resp.Body}
	img, err := imaging.Decode(counting)
	if err != nil {
		return nil, counting.count, fmt.Errorf("error decoding image for page %d: %v", pageNum, err)
	}

	if img == nil {
		return nil, counting.count, fmt.Errorf("downloaded image for page %d is empty", pageNum)
	}

	return img, counting.count, nil
}

// deobfuscateAndSave reverses the grid scrambling and writes the PNG to
// disk, returning the size of the saved file for reporting.
func (p Page) deobfuscateAndSave(img image.Image, outDir string, pageNum int) (int64, error) {
	filePath := filepath.Join(outDir, fmt.Sprintf("%03d.png", pageNum))
	imageCtx := NewImageContext(img)
	imageCtx.Deobfuscate(p.Width, p.Height)

	if err := imageCtx.SaveImage(filePath); err != nil {
		return 0, fmt.Errorf("error creating file for page %d: %v", pageNum, err)
	}

	if info, err := os.Stat(filePath); err == nil {
		return info.Size(), nil
	}
	// The file was saved successfully; not knowing its exact size is only
	// cosmetic, so this is not treated as an error.
	return 0, nil
}
