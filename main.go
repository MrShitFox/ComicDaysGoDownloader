package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"html"

	"github.com/PuerkitoBio/goquery"
	"github.com/disintegration/imaging"
)

type Page struct {
	Src    string `json:"src"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type Cookie struct {
	Domain         string  `json:"domain"`
	ExpirationDate float64 `json:"expirationDate"`
	HostOnly       bool    `json:"hostOnly"`
	HTTPOnly       bool    `json:"httpOnly"`
	Name           string  `json:"name"`
	Path           string  `json:"path"`
	SameSite       string  `json:"sameSite"`
	Secure         bool    `json:"secure"`
	Session        bool    `json:"session"`
	StoreID        string  `json:"storeId"`
	Value          string  `json:"value"`
}

func main() {
	fmt.Println("Comic Days Manga Downloader and Deobfuscator")
	fmt.Println("============================================")

	fmt.Println("\nStage 1: Initialization")
	fmt.Println("- This stage prepares the environment and retrieves manga information.")

	cookies := loadCookies("cookie.json")

	fmt.Print("Please enter a manga link from comic-days website: ")
	reader := bufio.NewReader(os.Stdin)
	url, _ := reader.ReadString('\n')
	url = strings.TrimSpace(url)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal("Error creating request:", err)
	}

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Error fetching the webpage:", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal("Error parsing the webpage:", err)
	}

	jsonData, exists := doc.Find("#episode-json").Attr("data-value")
	if !exists {
		log.Fatal("Could not find episode data on the page")
	}
	jsonData = html.UnescapeString(jsonData)

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		log.Fatal("Error parsing JSON data:", err)
	}

	pages := data["readableProduct"].(map[string]interface{})["pageStructure"].(map[string]interface{})["pages"].([]interface{})

	var validPages []Page
	for _, p := range pages {
		page := p.(map[string]interface{})
		if src, ok := page["src"].(string); ok && src != "" {
			validPages = append(validPages, Page{
				Src:    src,
				Width:  int(page["width"].(float64)),
				Height: int(page["height"].(float64)),
			})
		}
	}

	sort.Slice(validPages, func(i, j int) bool {
		return validPages[i].Src < validPages[j].Src
	})

	fmt.Printf("- Found %d pages\n", len(validPages))

	filesDir := filepath.Join(".", time.Now().Format("2006-01-02-15-04-05"))
	os.MkdirAll(filesDir, os.ModePerm)
	fmt.Printf("- Created directory for saving images: %s\n", filesDir)

	fmt.Println("\nStage 2: Downloading and Deobfuscating Pages")
	fmt.Println("- This stage downloads, deobfuscates, and saves each page of the manga.")

	for i, page := range validPages {
		pageNum := i + 1
		fmt.Printf("\nProcessing page %d of %d\n", pageNum, len(validPages))
		
		img := downloadPage(pageNum, page, cookies)
		if img == nil {
			fmt.Printf("Skipping page %d due to download error\n", pageNum)
			continue
		}
		
		deobfuscateAndSavePage(pageNum, page, img, filesDir)
	}

	fmt.Println("\nStage 3: Completion")
	fmt.Println("- All pages have been processed and saved.")
	fmt.Printf("- You can find the downloaded manga in the directory: %s\n", filesDir)
}

func loadCookies(filename string) []Cookie {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Warning: Could not open cookie file: %v\n", err)
		return nil
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Printf("Warning: Could not read cookie file: %v\n", err)
		return nil
	}

	var cookies []Cookie
	if err := json.Unmarshal(bytes, &cookies); err != nil {
		fmt.Printf("Warning: Could not parse cookie file: %v\n", err)
		return nil
	}

	fmt.Println("Successfully loaded cookies from file.")
	return cookies
}

func downloadPage(pageNum int, page Page, cookies []Cookie) image.Image {
	fmt.Printf("Downloading page %d...\n", pageNum)

	client := &http.Client{}
	req, err := http.NewRequest("GET", page.Src, nil)
	if err != nil {
		log.Printf("Error creating request for page %d: %v", pageNum, err)
		return nil
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Referer", "https://comic-days.com/")

	for _, cookie := range cookies {
		req.AddCookie(&http.Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error downloading image for page %d: %v", pageNum, err)
		return nil
	}
	defer resp.Body.Close()

	img, err := imaging.Decode(resp.Body)
	if err != nil {
		log.Printf("Error decoding image for page %d: %v", pageNum, err)
		return nil
	}

	fmt.Printf("Page %d downloaded successfully.\n", pageNum)
	return img
}

func deobfuscateAndSavePage(pageNum int, page Page, img image.Image, filesDir string) {
	fmt.Printf("Deobfuscating page %d...\n", pageNum)

	filePath := filepath.Join(filesDir, fmt.Sprintf("%03d.png", pageNum))
	spacingWidth := (page.Width / 32) * 8
	spacingHeight := (page.Height / 32) * 8

	newImg := image.NewRGBA(image.Rect(0, 0, page.Width, page.Height))

	for x := 0; x+spacingWidth <= page.Width; x += spacingWidth {
		for y := (x / spacingWidth) * spacingHeight + spacingHeight; y+spacingHeight <= page.Height; y += spacingHeight {
			oldRect := image.Rect(x, y, x+spacingWidth, y+spacingHeight)
			newPosX := (y / spacingHeight) * spacingWidth
			newPosY := (x / spacingWidth) * spacingHeight
			newRect := image.Rect(newPosX, newPosY, newPosX+spacingWidth, newPosY+spacingHeight)

			draw.Draw(newImg, oldRect, img, newRect.Min, draw.Src)
			draw.Draw(newImg, newRect, img, oldRect.Min, draw.Src)
		}
	}

	for i := 0; i < 4; i++ {
		midLineX := i * spacingWidth
		midLineY := i * spacingHeight
		midRect := image.Rect(midLineX, midLineY, midLineX+spacingWidth, midLineY+spacingHeight)
		draw.Draw(newImg, midRect, img, midRect.Min, draw.Src)
	}

	outFile, err := os.Create(filePath)
	if err != nil {
		log.Printf("Error creating file for page %d: %v", pageNum, err)
		return
	}
	defer outFile.Close()

	png.Encode(outFile, newImg)

	fmt.Printf("Page %d deobfuscated and saved.\n", pageNum)
}