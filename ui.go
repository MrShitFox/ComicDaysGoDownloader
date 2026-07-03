package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pterm/pterm"
)

// ui.go is the presentation layer for the whole pipeline: a title screen,
// staged section headers, live spinners/progress bars, a diagram explaining
// the descrambling algorithm and a closing report. Every other file routes
// its output through here (or through plain pterm.Info/Success/Warning/Error
// calls) instead of fmt/log, so nothing ever fights over the terminal: pterm
// automatically clears and redraws any active spinner or progress bar
// whenever a new line is printed.

const totalStages = 3

// spinnerFrames is a smooth 10-frame braille "dots" animation, a common and
// pleasant looking spinner style, used instead of pterm's blockier default.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// stagePalette gives every stage its own accent color so the pipeline reads
// like a small storyboard instead of a wall of uniform text.
var stagePalette = []pterm.Color{pterm.BgBlue, pterm.BgMagenta, pterm.BgGreen}

// ---------------------------------------------------------------------------
// Title screen & stage headers
// ---------------------------------------------------------------------------

// goMark is a small, hand-drawn "GO" logo mark — a nod to the "Go" in
// ComicDaysGoDownloader (this is a Go program) — shaded top-to-bottom from
// light teal to deep blue, echoing Go's own brand color. It deliberately
// stays compact instead of rendering the whole app name in pterm's stock
// block font, which every pterm-based CLI ends up looking like.
var goMark = []string{
	` ██████   ██████  `,
	`██        ██    ██ `,
	`██   ███  ██    ██ `,
	`██    ██  ██    ██ `,
	` ██████    ██████  `,
}

var goMarkShades = []pterm.RGB{
	pterm.NewRGB(94, 234, 212),
	pterm.NewRGB(45, 212, 230),
	pterm.NewRGB(14, 165, 233),
	pterm.NewRGB(2, 132, 199),
	pterm.NewRGB(3, 105, 161),
}

// gradientSegment is a run of text faded between two RGB colors.
type gradientSegment struct {
	text     string
	from, to pterm.RGB
}

// gradientText fades s's characters from `from` to `to`.
func gradientText(s string, from, to pterm.RGB) string {
	runes := []rune(s)
	n := len(runes)
	if n == 0 {
		return ""
	}
	var b strings.Builder
	for i, r := range runes {
		step := float32(0)
		if n > 1 {
			step = float32(i) / float32(n-1)
		}
		b.WriteString(from.Fade(0, 1, step, to).Sprint(string(r)))
	}
	return b.String()
}

// renderSegments concatenates several independently-faded gradientSegments.
func renderSegments(segments ...gradientSegment) string {
	var b strings.Builder
	for _, seg := range segments {
		b.WriteString(gradientText(seg.text, seg.from, seg.to))
	}
	return b.String()
}

// printBanner renders the application title screen: the GO mark as a small
// logo, the full app name as a gradient wordmark (with "Go" called out in
// Go's brand color) framed by manga-style corner brackets, and a tagline.
func printBanner() {
	var rows []string
	for i, l := range goMark {
		rows = append(rows, goMarkShades[i].Sprint(l))
	}
	pterm.DefaultCenter.Println(strings.Join(rows, "\n"))

	name := renderSegments(
		gradientSegment{"Comic Days ", pterm.NewRGB(56, 189, 248), pterm.NewRGB(56, 189, 248)},
		gradientSegment{"Go", pterm.NewRGB(0, 173, 216), pterm.NewRGB(0, 173, 216)},
		gradientSegment{" Downloader", pterm.NewRGB(217, 70, 239), pterm.NewRGB(217, 70, 239)},
	)
	pterm.DefaultCenter.Println(pterm.Gray("「 ") + name + pterm.Gray(" 」"))
	pterm.DefaultCenter.Println(pterm.Gray("manga downloader  ·  drm deobfuscator"))
	pterm.Println()
}

// printStage prints a full-width, colour-coded banner marking the start of a
// pipeline stage, followed by a one-line description of what it does.
func printStage(n int, title, desc string) {
	bg := stagePalette[(n-1)%len(stagePalette)]
	pterm.Println()
	pterm.DefaultHeader.
		WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(bg)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Printfln("STAGE %d/%d   %s", n, totalStages, strings.ToUpper(title))
	if desc != "" {
		pterm.Info.Println(desc)
	}
}

// fatal prints a styled fatal error and exits, playing the role log.Fatal
// used to, but through pterm so it cannot clash with an active spinner.
func fatal(err error) {
	pterm.Error.Println(err)
	os.Exit(1)
}

// ---------------------------------------------------------------------------
// Stage 1 — initialization helpers
// ---------------------------------------------------------------------------

// reportCookieLoad prints whether cookie.json was loaded successfully. A
// missing/broken cookie file is not fatal — the download simply continues
// unauthenticated — so this only ever warns, never fails.
func reportCookieLoad(cookies []Cookie, err error) {
	if err != nil {
		pterm.Warning.Printfln("🍪 Cookies not loaded: %v", err)
		pterm.Warning.Println("   Continuing without authentication — purchased/members-only chapters will fail.")
		return
	}
	pterm.Success.Printfln("🍪 Loaded %d cookie(s) from cookie.json", len(cookies))
}

// printURLPrompt prints a styled prompt on the current line (no newline), so
// the caller's bufio.Reader can keep reading the answer right after it.
func printURLPrompt() {
	pterm.Print(pterm.LightCyan("🔗 Manga URL ") + pterm.Gray("(comic-days.com/episode/...): "))
}

// newSpinner starts a spinner using a smoother animation than pterm's
// default, with its own timer disabled — callers that care about timing
// report it explicitly once an operation completes.
func newSpinner(text string) *pterm.SpinnerPrinter {
	sp, _ := pterm.DefaultSpinner.
		WithSequence(spinnerFrames...).
		WithDelay(90 * time.Millisecond).
		WithShowTimer(false).
		Start(text)
	return sp
}

// spinnerRetryObserver adapts a RetryObserver (see network.go) so retries
// happening deep inside the network layer surface as live updates to an
// already-running spinner instead of printing their own lines.
func spinnerRetryObserver(sp *pterm.SpinnerPrinter, label string) RetryObserver {
	return func(attempt, maxAttempts int, err error, delay time.Duration) {
		if delay <= 0 {
			sp.UpdateText(fmt.Sprintf("%s timed out: %v", label, err))
			return
		}
		sp.UpdateText(fmt.Sprintf("%s retry %d/%d in %v: %v", label, attempt, maxAttempts, delay.Round(time.Millisecond), err))
	}
}

// printSessionSummary renders a small info table once the chapter page has
// been parsed, right before the download pipeline starts.
func printSessionSummary(pageCount int, outDir string, cookieCount int) {
	rows := [][]string{
		{"Property", "Value"},
		{"Pages found", strconv.Itoa(pageCount)},
		{"Cookies loaded", strconv.Itoa(cookieCount)},
		{"Output directory", outDir},
	}
	pterm.DefaultTable.WithHasHeader().WithData(rows).WithBoxed().Render()
}

// ---------------------------------------------------------------------------
// Stage 2 — the descrambling legend
// ---------------------------------------------------------------------------

// printDeobfuscationLegend draws a side-by-side "before / after" diagram of
// Comic Days' grid-transpose scrambling, generated straight from the
// divideNum constant in imageprocessor.go so it can never drift out of sync
// with the real algorithm. Cells that swap places share a color; the
// untouched diagonal is grayed out.
func printDeobfuscationLegend() {
	palette := []pterm.Color{
		pterm.FgLightCyan, pterm.FgLightMagenta, pterm.FgLightYellow,
		pterm.FgLightGreen, pterm.FgLightRed, pterm.FgLightBlue,
	}

	styleFor := func(r, c int) *pterm.Style {
		if r == c {
			return pterm.NewStyle(pterm.FgGray)
		}
		lo, hi := r, c
		if lo > hi {
			lo, hi = hi, lo
		}
		return pterm.NewStyle(palette[(lo*divideNum+hi)%len(palette)], pterm.Bold)
	}

	received := renderGrid(divideNum, func(r, c int) string {
		return fmt.Sprintf("R%dC%d", r, c)
	}, styleFor)

	restored := renderGrid(divideNum, func(r, c int) string {
		return fmt.Sprintf("R%dC%d", c, r) // transpose: (r,c) <- (c,r)
	}, styleFor)

	leftBox := pterm.DefaultBox.WithTitle(pterm.LightRed("① received (scrambled)")).WithTitleTopCenter().Sprint(received)
	rightBox := pterm.DefaultBox.WithTitle(pterm.LightGreen("② deobfuscated")).WithTitleTopCenter().Sprint(restored)

	panels, err := pterm.DefaultPanel.WithPanels(pterm.Panels{
		{{Data: leftBox}, {Data: rightBox}},
	}).Srender()

	explanation := fmt.Sprintf(
		"Comic Days splits every page into a %[1]dx%[1]d grid and swaps cell (row,col)\n"+
			"with cell (col,row) before serving it. Same-colored cells below are swapped\n"+
			"with each other to undo it; gray cells sit on the diagonal and never move.\n",
		divideNum,
	)

	body := explanation
	if err == nil {
		body += "\n" + panels
	} else {
		body += "\n" + received + "\n\n" + restored
	}

	pterm.DefaultBox.
		WithTitle(" 🧩 How the descrambling works ").
		WithTitleTopLeft().
		Println(body)
}

// renderGrid draws a divide x divide box-drawing grid, one cell per (row,
// col), using label() for its text and styleFor() (optional) for its color.
func renderGrid(divide int, label func(r, c int) string, styleFor func(r, c int) *pterm.Style) string {
	cellWidth := 0
	for r := 0; r < divide; r++ {
		for c := 0; c < divide; c++ {
			if l := len([]rune(label(r, c))); l > cellWidth {
				cellWidth = l
			}
		}
	}
	cellWidth += 2

	hBar := strings.Repeat("─", cellWidth)
	top := "┌" + strings.Repeat(hBar+"┬", divide-1) + hBar + "┐"
	mid := "├" + strings.Repeat(hBar+"┼", divide-1) + hBar + "┤"
	bot := "└" + strings.Repeat(hBar+"┴", divide-1) + hBar + "┘"

	var b strings.Builder
	b.WriteString(top)
	for r := 0; r < divide; r++ {
		b.WriteString("\n│")
		for c := 0; c < divide; c++ {
			cell := centerText(label(r, c), cellWidth)
			if styleFor != nil {
				cell = styleFor(r, c).Sprint(cell)
			}
			b.WriteString(cell + "│")
		}
		if r != divide-1 {
			b.WriteString("\n" + mid)
		}
	}
	b.WriteString("\n" + bot)
	return b.String()
}

func centerText(s string, width int) string {
	pad := width - len([]rune(s))
	if pad <= 0 {
		return s
	}
	left := pad / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", pad-left)
}

// ---------------------------------------------------------------------------
// Stage 2 — live per-page pipeline (status spinner with an embedded bar)
// ---------------------------------------------------------------------------

// pageResult carries what a successfully processed page should report.
type pageResult struct {
	pageNum       int
	width, height int
	downloadBytes int64
	savedBytes    int64
	elapsed       time.Duration
}

// Pipeline narrates the whole download loop through a single live spinner
// that carries a hand-drawn progress bar plus whatever the current page is
// doing right now (downloading, unscrambling, saving...). It deliberately
// keeps exactly one live printer active: pterm auto-clears/redraws whatever
// live printers are active whenever a new pterm.Success/Warning/Error line
// is printed, but it does so once *per active live printer* — running a
// spinner and a progress bar at the same time would print every one of
// those per-page lines twice. A single spinner with a bar baked into its
// text gets the same visual result without that pitfall.
type Pipeline struct {
	total   int
	done    int
	spinner *pterm.SpinnerPrinter
	start   time.Time

	okCount, failCount             int
	totalDownloadBytes, totalSaved int64
}

// barWidth is how many characters wide the hand-drawn progress bar is.
const barWidth = 24

// StartPipeline begins tracking `total` pages.
func StartPipeline(total int) *Pipeline {
	pl := &Pipeline{total: total, start: time.Now()}
	pl.spinner = newSpinner(pl.render(0, "warming up..."))
	return pl
}

// progressBar renders a filled/empty block bar for current out of total.
func progressBar(current, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if current > total {
		current = total
	}
	filled := current * width / total
	return pterm.LightCyan(strings.Repeat("█", filled)) + pterm.Gray(strings.Repeat("░", width-filled))
}

// render composes the bar, percentage, page counter and status text into the
// single line shown by the spinner.
func (pl *Pipeline) render(pageNum int, status string) string {
	pct := 0
	if pl.total > 0 {
		pct = pl.done * 100 / pl.total
	}
	bar := progressBar(pl.done, pl.total, barWidth)
	if pageNum <= 0 {
		return fmt.Sprintf("%s %3d%%  %s", bar, pct, status)
	}
	return fmt.Sprintf("%s %3d%%  page %d/%d · %s", bar, pct, pageNum, pl.total, status)
}

// Status updates the spinner for the page currently being processed.
func (pl *Pipeline) Status(pageNum int, format string, a ...any) {
	pl.spinner.UpdateText(pl.render(pageNum, fmt.Sprintf(format, a...)))
}

// RetryObserver returns a RetryObserver that narrates retries for pageNum
// through the pipeline's spinner instead of printing new lines.
func (pl *Pipeline) RetryObserver(pageNum int, phase string) RetryObserver {
	return func(attempt, maxAttempts int, err error, delay time.Duration) {
		if delay <= 0 {
			pl.spinner.UpdateText(pl.render(pageNum, fmt.Sprintf("%s timed out: %v", phase, err)))
			return
		}
		pl.spinner.UpdateText(pl.render(pageNum, fmt.Sprintf(
			"%s retry %d/%d in %v: %v", phase, attempt, maxAttempts, delay.Round(time.Millisecond), err,
		)))
	}
}

// PageSucceeded logs a permanent success line for a page and advances the
// hand-drawn progress bar.
func (pl *Pipeline) PageSucceeded(r pageResult) {
	pl.okCount++
	pl.done++
	pl.totalDownloadBytes += r.downloadBytes
	pl.totalSaved += r.savedBytes
	pterm.Success.Printfln(
		"[%d/%d] %03d.png saved · %dx%d · %s → %s PNG · %v",
		r.pageNum, pl.total, r.pageNum, r.width, r.height,
		humanBytes(r.downloadBytes), humanBytes(r.savedBytes), r.elapsed.Round(time.Millisecond),
	)
}

// PageFailed logs a permanent failure line for a page and advances the
// hand-drawn progress bar (a failed page still counts as "handled").
func (pl *Pipeline) PageFailed(pageNum int, err error) {
	pl.failCount++
	pl.done++
	pterm.Error.Printfln("[%d/%d] giving up: %v", pageNum, pl.total, err)
}

// Finish stops the spinner and returns the run's statistics.
func (pl *Pipeline) Finish(outDir string) RunStats {
	if pl.failCount == 0 {
		pl.spinner.Success(fmt.Sprintf("All %d page(s) processed", pl.total))
	} else {
		pl.spinner.Warning(fmt.Sprintf("Processed %d page(s), %d failed", pl.total, pl.failCount))
	}

	return RunStats{
		Total:         pl.total,
		Succeeded:     pl.okCount,
		Failed:        pl.failCount,
		OutDir:        outDir,
		Elapsed:       time.Since(pl.start),
		DownloadBytes: pl.totalDownloadBytes,
		SavedBytes:    pl.totalSaved,
	}
}

// ---------------------------------------------------------------------------
// Stage 3 — closing report
// ---------------------------------------------------------------------------

// RunStats summarizes a completed download run for the final report.
type RunStats struct {
	Total, Succeeded, Failed  int
	OutDir                    string
	Elapsed                   time.Duration
	DownloadBytes, SavedBytes int64
}

// printFinalSummary renders the closing report: a stats table plus a
// colour-coded verdict box.
func printFinalSummary(stats RunStats) {
	rows := [][]string{
		{"Property", "Value"},
		{"Pages processed", strconv.Itoa(stats.Total)},
		{"Succeeded", pterm.LightGreen(strconv.Itoa(stats.Succeeded))},
	}
	if stats.Failed > 0 {
		rows = append(rows, []string{"Failed", pterm.LightRed(strconv.Itoa(stats.Failed))})
	}
	rows = append(rows,
		[]string{"Downloaded", humanBytes(stats.DownloadBytes)},
		[]string{"Saved to disk", humanBytes(stats.SavedBytes)},
		[]string{"Elapsed", stats.Elapsed.Round(time.Second).String()},
		[]string{"Output directory", stats.OutDir},
	)
	pterm.DefaultTable.WithHasHeader().WithData(rows).WithBoxed().Render()
	pterm.Println()

	if stats.Failed == 0 {
		pterm.DefaultBox.
			WithTitle(" ✓ Done ").
			WithBoxStyle(pterm.NewStyle(pterm.FgGreen)).
			Println(pterm.LightGreen(fmt.Sprintf("All %d page(s) saved to %s", stats.Total, stats.OutDir)))
		return
	}
	pterm.DefaultBox.
		WithTitle(" ⚠ Done with errors ").
		WithBoxStyle(pterm.NewStyle(pterm.FgYellow)).
		Println(pterm.LightYellow(fmt.Sprintf(
			"%d/%d page(s) saved, %d failed. Check the log above for details.",
			stats.Succeeded, stats.Total, stats.Failed,
		)))
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

// humanBytes formats a byte count using binary (KiB/MiB/...) units.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
