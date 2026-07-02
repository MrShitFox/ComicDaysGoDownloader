package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("Comic Days Manga Downloader and Deobfuscator")
	fmt.Println("============================================")

	fmt.Println("\nStage 1: Initialization")
	fmt.Println("- This stage prepares the environment and retrieves manga information.")

	session, err := NewComicSession("cookie.json")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nStage 2: Downloading and Deobfuscating Pages")
	fmt.Println("- This stage downloads, deobfuscates, and saves each page of the manga.")

	if len(session.Pages) == 0 {
		log.Fatal("No pages were found for this chapter. It may be unavailable or require a valid cookie.")
	}

	failed := 0
	for i, page := range session.Pages {
		pageNum := i + 1
		fmt.Printf("\nProcessing page %d of %d\n", pageNum, len(session.Pages))
		if err := page.Process(session.NetworkClient, session.Cookies, session.OutDir, pageNum); err != nil {
			failed++
			log.Printf("Page %d could not be downloaded: %v", pageNum, err)
		}
	}

	fmt.Println("\nStage 3: Completion")
	if failed == 0 {
		fmt.Println("- All pages have been processed and saved.")
	} else {
		fmt.Printf("- Done, but %d of %d pages could not be downloaded.\n", failed, len(session.Pages))
	}
	fmt.Printf("- You can find the downloaded manga in the directory: %s\n", session.OutDir)
}
