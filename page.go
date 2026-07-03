package main

import (
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/disintegration/imaging"
)

const (
	retryDelay              = 10 * time.Second
	maxPageDownloadAttempts = 3
	maxPagePixels           = 100_000_000
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
// step through pl. It retries transient failures a bounded number of times but
// gives up on permanent errors (for example a page that requires a purchase)
// immediately. It returns an error only when the page could not be
// produced; pl has already reported success or failure by the time it does.
func (p Page) Process(networkClient HTTPFetcher, cookies []Cookie, outDir string, pageNum int, pl *Pipeline) error {
	start := time.Now()

	var img image.Image
	var downloadedBytes int64
	var err error

	for attempt := 1; attempt <= maxPageDownloadAttempts; attempt++ {
		pl.Status(pageNum, "downloading...")
		img, downloadedBytes, err = p.downloadAttempt(networkClient, cookies, pageNum, pl)
		if err == nil {
			break
		}

		if IsPermanent(err) {
			pl.PageFailed(pageNum, err)
			return fmt.Errorf("page %d: %w", pageNum, err)
		}
		if attempt == maxPageDownloadAttempts {
			finalErr := fmt.Errorf("download failed after %d attempts: %w", attempt, err)
			pl.PageFailed(pageNum, finalErr)
			return fmt.Errorf("page %d: %w", pageNum, finalErr)
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

func (p Page) downloadAttempt(networkClient HTTPFetcher, cookies []Cookie, pageNum int, pl *Pipeline) (image.Image, int64, error) {
	src, err := normalizeComicDaysAssetURL(p.Src)
	if err != nil {
		return nil, 0, &PermanentError{Err: fmt.Errorf("invalid page %d src: %w", pageNum, err)}
	}
	req, err := http.NewRequest("GET", src, nil)
	if err != nil {
		return nil, 0, &PermanentError{Err: fmt.Errorf("error creating request for page %d: %v", pageNum, err)}
	}

	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Referer", "https://comic-days.com/")
	req.Header.Set("Origin", "https://comic-days.com")
	addCookies(req, cookies)

	var onRetry RetryObserver
	if pl != nil {
		onRetry = pl.RetryObserver(pageNum, "download")
	}
	resp, err := networkClient.FetchWithRetries(req, onRetry)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to download page %d: %w", pageNum, err)
	}
	if resp == nil {
		return nil, 0, fmt.Errorf("failed to download page %d: empty response", pageNum)
	}
	defer resp.Body.Close()
	if err := validateImageContentType(resp, pageNum); err != nil {
		return nil, 0, err
	}

	counting := &countingReader{r: resp.Body}
	img, err := imaging.Decode(counting)
	if err != nil {
		return nil, counting.count, fmt.Errorf("error decoding image for page %d: %v", pageNum, err)
	}

	if img == nil {
		return nil, counting.count, fmt.Errorf("downloaded image for page %d is empty", pageNum)
	}
	if err := p.validateImageBounds(img); err != nil {
		return nil, counting.count, err
	}

	return img, counting.count, nil
}

func validateImageContentType(resp *http.Response, pageNum int) error {
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		return nil
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = contentType
	}
	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if !strings.HasPrefix(mediaType, "image/") {
		return &PermanentError{Err: fmt.Errorf("page %d returned %q instead of an image", pageNum, contentType)}
	}
	return nil
}

func validatePageDimensions(width, height int) error {
	if width <= 0 || height <= 0 {
		return fmt.Errorf("dimensions must be positive, got %dx%d", width, height)
	}
	if width > maxPagePixels/height {
		return fmt.Errorf("dimensions %dx%d exceed the %d pixel safety limit", width, height, maxPagePixels)
	}
	return nil
}

func (p Page) validateImageBounds(img image.Image) error {
	if err := validatePageDimensions(p.Width, p.Height); err != nil {
		return &PermanentError{Err: err}
	}
	bounds := img.Bounds()
	if bounds.Dx() != p.Width || bounds.Dy() != p.Height {
		return &PermanentError{Err: fmt.Errorf(
			"decoded image dimensions %dx%d do not match metadata %dx%d",
			bounds.Dx(), bounds.Dy(), p.Width, p.Height,
		)}
	}
	return nil
}

// deobfuscateAndSave reverses the grid scrambling and writes the PNG to
// disk, returning the size of the saved file for reporting.
func (p Page) deobfuscateAndSave(img image.Image, outDir string, pageNum int) (int64, error) {
	if err := p.validateImageBounds(img); err != nil {
		return 0, err
	}
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
