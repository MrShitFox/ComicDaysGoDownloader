package main

import "fmt"

func main() {
	printBanner()

	printStage(1, "Initialization", "Reading cookies, asking for a chapter URL and fetching + parsing its page data.")
	session, err := NewComicSession("cookie.json")
	if err != nil {
		fatal(err)
	}

	printStage(2, "Download & Deobfuscation", "Downloading each page and reversing Comic Days' grid-transpose scrambling.")
	if len(session.Pages) == 0 {
		fatal(fmt.Errorf("no pages were found for this chapter — it may be unavailable or require a valid cookie"))
	}

	printDeobfuscationLegend()

	pl := StartPipeline(len(session.Pages))
	for i, page := range session.Pages {
		pageNum := i + 1
		// Process already reports success/failure for this page through pl,
		// so the returned error only decides the exit code of the loop body,
		// not whether anything more needs to be printed here.
		_ = page.Process(session.NetworkClient, session.Cookies, session.OutDir, pageNum, pl)
	}
	stats := pl.Finish(session.OutDir)

	printStage(3, "Summary", "Here's how the run went.")
	printFinalSummary(stats)
}
