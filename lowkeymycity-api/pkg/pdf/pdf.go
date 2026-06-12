// Package pdf renders a stored quiz result as the same one-pager the
// frontend produces with window.print(): A4, @page margin 10mm 12mm, and
// the Tailwind print: styles from ResultBody.vue and friends. Sizes are
// converted from CSS px (96dpi) to pt (×0.75) and mm (×25.4/96).
package pdf

import (
	"bytes"
	_ "embed"
	"strconv"
	"strings"

	"lowkeymycity/pkg/types"

	"github.com/go-pdf/fpdf"
)

//go:embed fonts/Inter-Regular.ttf
var interRegular []byte

//go:embed fonts/Inter-SemiBold.ttf
var interSemiBold []byte

//go:embed fonts/BricolageGrotesque-Bold.ttf
var bricolageBold []byte

//go:embed fonts/BricolageGrotesque-ExtraBold.ttf
var bricolageExtraBold []byte

const (
	pxToMM = 25.4 / 96
	pxToPt = 0.75

	pageW   = 210.0
	marginX = 12.0 // @page margin: 10mm 12mm
	marginY = 10.0

	// max-w-md (448px) — the centered header text block
	headerW = 448 * pxToMM
)

// theme colors from main.css
var (
	ink       = [3]int{0x22, 0x1d, 0x17}
	inkSoft   = [3]int{0x6b, 0x62, 0x58}
	lilacDeep = [3]int{0x7c, 0x5c, 0xf0}
	coral     = [3]int{0xff, 0x6b, 0x5e}
	// border-ink/10 flattened onto white paper
	inkBorder = [3]int{0xe9, 0xe8, 0xe8}
)

// renderer wraps the document being assembled; its methods are the print
// stylesheet's vocabulary (kicker, heading, paragraph, gap, ...) that the
// header and body layouts are written in.
type renderer struct {
	doc *fpdf.Fpdf
}

// Render lays result out as the PDF the result page produces when printed.
// result.Mode picks the layout: "city" gets the verdict header with the
// big coral score, every other value the match header with the city name
// in lilac; both continue with the green/red flags, alternatives, closing
// and the "made at lowkeymycity.com" sign-off. Nothing in result is
// validated — empty fields render as blank space, and content too long for
// one page flows onto extra pages automatically.
//
// It returns the complete PDF as bytes, with the site's fonts embedded
// (Bricolage Grotesque and Inter, compiled into the binary). The error is
// the underlying fpdf error if document assembly fails — with valid
// embedded fonts that effectively never happens.
func Render(result types.QuizResult) ([]byte, error) {
	doc := fpdf.New("P", "mm", "A4", "")
	doc.SetMargins(marginX, marginY+8*pxToMM, marginX) // + the page's print:pt-2
	doc.SetAutoPageBreak(true, marginY)

	// The display font is never used lighter than bold, so bold takes the
	// regular slot and extrabold takes "B". Inter's "B" is the semibold the
	// site uses for the uppercase kicker.
	doc.AddUTF8FontFromBytes("Bricolage", "", bricolageBold)
	doc.AddUTF8FontFromBytes("Bricolage", "B", bricolageExtraBold)
	doc.AddUTF8FontFromBytes("Inter", "", interRegular)
	doc.AddUTF8FontFromBytes("Inter", "B", interSemiBold)

	doc.AddPage()

	r := &renderer{doc: doc}
	if result.Mode == "city" {
		r.cityHeader(result)
		r.body(result, "better matches for you")
	} else {
		r.matchHeader(result)
		r.body(result, "also would've worked")
	}

	var buf bytes.Buffer
	if err := doc.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// color sets the text color for everything drawn after it.
func (r *renderer) color(c [3]int) {
	r.doc.SetTextColor(c[0], c[1], c[2])
}

// matchHeader draws the centered match-mode header from CityMatchResult.vue:
// the "your city is" kicker, the city name big in lilac (print flattens the
// screen gradient to the solid color), then title and summary. Leaves the
// cursor below the summary.
func (r *renderer) matchHeader(result types.QuizResult) {
	r.kicker("your city is")
	r.gap(4)

	r.doc.SetFont("Bricolage", "B", 30*pxToPt) // print:text-3xl extrabold
	r.color(lilacDeep)
	r.centered(result.City, 36*pxToMM)

	r.gap(12) // print:mt-3
	r.titleAndSummary(result)
}

// cityHeader draws the centered city-mode header from CityCheckResult.vue:
// the "the verdict on" kicker, the city name, the coral score line, then
// title and summary. Leaves the cursor below the summary.
func (r *renderer) cityHeader(result types.QuizResult) {
	r.kicker("the verdict on")
	r.gap(4)

	r.doc.SetFont("Bricolage", "", 24*pxToPt) // print:text-2xl bold
	r.color(ink)
	r.centered(result.City, 32*pxToMM)

	r.gap(8) // my-2
	r.scoreLine(result.Score)
	r.gap(8)

	r.titleAndSummary(result)
}

// titleAndSummary draws the shared tail of both headers: the result title
// in the display font, then the summary in muted body type, both centered
// and capped at max-w-md.
func (r *renderer) titleAndSummary(result types.QuizResult) {
	r.doc.SetFont("Bricolage", "", 20*pxToPt) // print:text-xl bold
	r.color(ink)
	r.centered(result.Title, 28*pxToMM)

	r.gap(4)                              // print:mt-1
	r.doc.SetFont("Inter", "", 14*pxToPt) // print:text-sm
	r.color(inkSoft)
	r.centered(result.Summary, 20*pxToMM)
}

// scoreLine draws "<score> /100 lowkey" as one centered, baseline-aligned
// line: the number giant and coral, the tail small and muted. Any int
// renders as-is — the caller decides whether a score makes sense. Leaves
// the cursor below the line.
func (r *renderer) scoreLine(score int) {
	scoreStr := strconv.Itoa(score)
	gap := 8 * pxToMM // ml-2

	r.doc.SetFont("Bricolage", "B", 48*pxToPt) // print:text-5xl extrabold
	scoreW := r.doc.GetStringWidth(scoreStr)
	r.doc.SetFont("Bricolage", "", 18*pxToPt) // print:text-lg bold
	tailW := r.doc.GetStringWidth("/100 lowkey")

	x := (pageW - scoreW - gap - tailW) / 2
	lineH := 48 * pxToMM // text-5xl line-height 1
	baseline := r.doc.GetY() + lineH*0.8

	r.doc.SetFont("Bricolage", "B", 48*pxToPt)
	r.color(coral) // print:text-coral, whatever the screen gradient was
	r.doc.Text(x, baseline, scoreStr)

	r.doc.SetFont("Bricolage", "", 18*pxToPt)
	r.color(inkSoft)
	r.doc.Text(x+scoreW+gap, baseline, "/100 lowkey")

	r.doc.SetY(r.doc.GetY() + lineH)
}

// body draws the single-column print body shared by both modes (the map
// and buttons of the screen layout are print-hidden): the green flags and
// red flags sections, the numbered alternatives under alternativesHeading
// (skipped entirely when there are none), the closing paragraph under a
// hairline rule, and the print-only "made at lowkeymycity.com" sign-off.
// Content that doesn't fit the page continues on the next one.
func (r *renderer) body(result types.QuizResult, alternativesHeading string) {
	r.gap(20) // print:mt-5

	r.section("green flags", result.GreenFlags)
	r.gap(12) // print:space-y-3
	r.section("red flags", result.RedFlags)

	if len(result.Alternatives) > 0 {
		r.gap(12)
		r.heading(alternativesHeading)
		r.gap(4)
		for i, alt := range result.Alternatives {
			if i > 0 {
				r.gap(8) // print:space-y-2
			}
			r.doc.SetFont("Bricolage", "", 12*pxToPt) // print:text-xs bold
			r.color(ink)
			r.doc.MultiCell(0, 16*pxToMM, strconv.Itoa(i+1)+". "+alt.City, "", "L", false)
			r.paragraph(alt.Blurb)
		}
	}

	// closing sits under a hairline rule (border-t border-ink/10 print:pt-2)
	r.gap(12)
	r.doc.SetDrawColor(inkBorder[0], inkBorder[1], inkBorder[2])
	r.doc.SetLineWidth(1 * pxToMM)
	y := r.doc.GetY()
	r.doc.Line(marginX, y, pageW-marginX, y)
	r.gap(8)
	r.paragraph(result.Closing)

	// the print-only sign-off
	r.gap(12)
	r.doc.SetFont("Inter", "", 10*pxToPt)
	r.color(inkSoft)
	r.doc.MultiCell(0, 14*pxToMM, "made at lowkeymycity.com", "", "L", false)
}

// section draws one flags section: the heading, then each paragraph with
// the print paragraph spacing between them. An empty paragraphs slice
// draws just the heading.
func (r *renderer) section(title string, paragraphs []string) {
	r.heading(title)
	r.gap(4) // print:mt-1
	for i, p := range paragraphs {
		if i > 0 {
			r.gap(8) // print:space-y-2
		}
		r.paragraph(p)
	}
}

// heading draws a left-aligned section heading in the bold display font,
// wrapping if it's too long for the line, and moves the cursor below it.
func (r *renderer) heading(text string) {
	r.doc.SetFont("Bricolage", "", 16*pxToPt) // print:text-base bold
	r.color(ink)
	r.doc.MultiCell(0, 24*pxToMM, text, "", "L", false)
}

// paragraph draws one body paragraph across the full content width at the
// print body size (11px Inter, snug leading), wrapping as needed, and
// moves the cursor below it.
func (r *renderer) paragraph(text string) {
	r.doc.SetFont("Inter", "", 11*pxToPt) // print:text-[11px] leading-snug
	r.color(ink)
	r.doc.MultiCell(0, 11*1.375*pxToMM, text, "", "L", false)
}

// kicker draws the small muted uppercase label above each header, centered
// on the page. The tracking-widest letter-spacing is simulated by placing
// each rune by hand, since fpdf has no letter-spacing of its own; text is
// uppercased here, so any casing can be passed in. Moves the cursor below
// the line.
func (r *renderer) kicker(text string) {
	size := 12 * pxToPt // print:text-xs
	r.doc.SetFont("Inter", "B", size)
	r.color(inkSoft)

	tracking := 0.1 * size * ptToMM() // tracking-widest = 0.1em
	runes := []rune(strings.ToUpper(text))
	var width float64
	for _, c := range runes {
		width += r.doc.GetStringWidth(string(c))
	}
	width += tracking * float64(len(runes)-1)

	lineH := 16 * pxToMM
	x := (pageW - width) / 2
	baseline := r.doc.GetY() + lineH*0.8
	for _, c := range runes {
		r.doc.Text(x, baseline, string(c))
		x += r.doc.GetStringWidth(string(c)) + tracking
	}
	r.doc.SetY(r.doc.GetY() + lineH)
}

// centered draws a text block centered on the page and capped at max-w-md
// width like the site's header copy, wrapping onto lineH-tall lines as
// needed, in whatever font and color are currently set. Moves the cursor
// below the block.
func (r *renderer) centered(text string, lineH float64) {
	left := (pageW - headerW) / 2
	r.doc.SetLeftMargin(left)
	r.doc.SetX(left)
	r.doc.MultiCell(headerW, lineH, text, "", "C", false)
	r.doc.SetLeftMargin(marginX)
}

// gap moves the cursor down by px CSS pixels, so the Tailwind spacing
// values read literally at the call sites.
func (r *renderer) gap(px float64) {
	r.doc.SetY(r.doc.GetY() + px*pxToMM)
}

// ptToMM converts typographic points to millimeters.
func ptToMM() float64 { return 25.4 / 72 }
