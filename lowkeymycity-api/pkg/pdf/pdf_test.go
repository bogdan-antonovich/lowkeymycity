package pdf

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"lowkeymycity/pkg/types"
)

// trimmed from web/src/data/mockResults.ts — the reference for what the
// printed page looks like
var fixtures = map[string]types.QuizResult{
	"city": {
		ID:      "demo-city-high",
		Mode:    "city",
		City:    "Portland, OR",
		Score:   86,
		Title:   "portland is lowkey your soulmate",
		Summary: "you want quiet streets, strong coffee, and the freedom to cancel plans without a group apology. portland built a whole economy around exactly that.",
		GreenFlags: []string{
			"you and portland want the same things. it's a city built for people who like their fun loosely scheduled and their evenings quiet — neighborhood bars over clubs, food carts over scenes, and a bookstore (powell's) so big it hands out maps at the door. your answers about slow mornings and small-group plans describe the default culture here, not the exception.",
			"the nature access is absurd for a city this size. forest park alone is 5,200 acres of real trails inside city limits, the columbia gorge is forty minutes east, and mount hood shows up at the end of half the streets like a screensaver. you said you recharge outside — in portland that's a tuesday, not a production.",
		},
		RedFlags: []string{
			"the grey is real. from november to may the sky commits to one shade and stays there, and the drizzle works remote too. locals cope with sad lamps, saunas, and pretending to enjoy winter hiking. if your weather answer was even slightly optimistic, knock a few points off yourself.",
			"the job market is thinner than the lifestyle deserves — salaries run lower than seattle or the bay, while rent no longer does. it's a much better city to bring an income to than to find one in.",
		},
		Alternatives: []types.Alternative{
			{City: "Bellingham, WA", Blurb: "portland's vibe in a smaller font. a college waterfront town with mountains behind it, half the traffic, and the same rain subscription. if you wanted portland but even quieter, this is it."},
			{City: "Bend, OR", Blurb: "the sunny-side alternative — about 300 clear days a year, high desert trails out the back door, and a beer scene that takes itself exactly as seriously as portland does. winters are colder but brighter, which might matter for you."},
			{City: "Olympia, WA", Blurb: "small, green, and genuinely lowkey. state-capital steady, evergreens everywhere, less food and culture but more calm — a fair trade depending on the week."},
		},
		Closing: "but honestly? 86 is a high score and it's earned. portland matches the way you actually live, not the way you answer quizzes. visit once in february before you commit — if you survive that, you're home.",
	},
	"match": {
		ID:      "demo-match",
		Mode:    "match",
		City:    "Asheville, NC",
		Title:   "certified slow-living enjoyer",
		Summary: "nobody there is in a hurry, and neither are you. you'd fit in by tuesday.",
		GreenFlags: []string{
			"asheville is what happens when a mountain town gets a record store and decides that's the ceiling. the river arts district is full of working studios you can wander into, the blue ridge parkway starts practically at the city limits, and the default friday night is a brewery patio that wraps up by ten. your answers about small crowds, big nature, and unhurried everything map onto this place almost one to one.",
			"it's also the right size for your social battery: big enough to stay anonymous at the farmers market, small enough that the barista learns your order by week two and then — crucially — leaves it at that.",
		},
		RedFlags: []string{
			"october is a siege. leaf season brings traffic that doubles your drive and fills every trailhead lot by 9am, and summer weekends aren't much gentler. the town you'd live in from november to may shares an address with a theme park the rest of the year.",
			"local wages never caught up with local rents — this is a bring-your-own-income town now. and the airport is small: direct flights exist, but visiting far-away friends usually means a layover in charlotte and a small prayer.",
		},
		Alternatives: []types.Alternative{
			{City: "Bend, OR", Blurb: "same blueprint — trails, breweries, unbothered people — but high desert instead of blue ridge. drier, sunnier, snowier. if asheville is a flannel, bend is a puffer jacket."},
			{City: "Burlington, VT", Blurb: "the lake-town version. church street instead of a river district, winters that demand commitment, and a town that treats 'going outside' as the entire social calendar. equally unhurried, somehow even more granola."},
		},
		Closing: "we're confident in this one. asheville isn't a compromise between city and mountains — it just never separated the two. go in shoulder season, rent for a month, and see if you ever come back.",
	},
}

// TestRender renders both quiz modes from the fixtures and fails unless
// each output starts with a PDF header. With PDF_SAMPLES_DIR set, it also
// writes <mode>.pdf samples there for eyeballing.
func TestRender(t *testing.T) {
	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			doc, err := Render(fixture)
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			if !bytes.HasPrefix(doc, []byte("%PDF-")) {
				t.Fatalf("output is not a PDF (starts with %q)", doc[:min(8, len(doc))])
			}

			// drop samples next to the test output for eyeballing
			if dir := os.Getenv("PDF_SAMPLES_DIR"); dir != "" {
				if err := os.WriteFile(filepath.Join(dir, name+".pdf"), doc, 0o644); err != nil {
					t.Fatalf("writing sample: %v", err)
				}
			}
		})
	}
}
