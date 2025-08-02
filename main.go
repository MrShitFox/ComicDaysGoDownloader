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

	for i, page := range session.Pages {
		pageNum := i + 1
		fmt.Printf("\nProcessing page %d of %d\n", pageNum, len(session.Pages))
		page.Process(session.NetworkClient, session.Cookies, session.OutDir, pageNum)
	}

	fmt.Println("\nStage 3: Completion")
	fmt.Println("- All pages have been processed and saved.")
	fmt.Printf("- You can find the downloaded manga in the directory: %s\n", session.OutDir)
}