import type { QuizResult } from '@/types/quiz'

// Hardcoded results for the /preview page. Once the backend exists these
// double as fixtures for the mock API layer, and as the reference for how
// long and how structured the LLM output should be.
export const MOCK_RESULTS = {
  cityHigh: {
    mode: 'city',
    city: 'Portland, OR',
    score: 86,
    title: 'portland is lowkey your soulmate',
    summary:
      'you want quiet streets, strong coffee, and the freedom to cancel plans without a group apology. portland built a whole economy around exactly that.',
    greenFlags: [
      "you and portland want the same things. it's a city built for people who like their fun loosely scheduled and their evenings quiet, neighborhood bars over clubs, food carts over scenes, and a bookstore (powell's) so big it hands out maps at the door. your answers about slow mornings and small-group plans describe the default culture here, not the exception.",
      "the nature access is absurd for a city this size. forest park alone is 5,200 acres of real trails inside city limits, the columbia gorge is forty minutes east, and mount hood shows up at the end of half the streets like a screensaver. you said you recharge outside, in portland that's a tuesday, not a production.",
    ],
    redFlags: [
      'the grey is real. from november to may the sky commits to one shade and stays there, and the drizzle works remote too. locals cope with sad lamps, saunas, and pretending to enjoy winter hiking. if your weather answer was even slightly optimistic, knock a few points off yourself.',
      "the job market is thinner than the lifestyle deserves, salaries run lower than seattle or the bay, while rent no longer does. it's a much better city to bring an income to than to find one in.",
    ],
    alternatives: [
      {
        city: 'Bellingham, WA',
        blurb:
          "portland's vibe in a smaller font. a college waterfront town with mountains behind it, half the traffic, and the same rain subscription. if you wanted portland but even quieter, this is it.",
      },
      {
        city: 'Bend, OR',
        blurb:
          'the sunny-side alternative, about 300 clear days a year, high desert trails out the back door, and a beer scene that takes itself exactly as seriously as portland does. winters are colder but brighter, which might matter for you.',
      },
      {
        city: 'Olympia, WA',
        blurb:
          'small, green, and genuinely lowkey. state-capital steady, evergreens everywhere, less food and culture but more calm, a fair trade depending on the week.',
      },
    ],
    closing:
      "but honestly? 86 is a high score and it's earned. portland matches the way you actually live, not the way you answer quizzes. visit once in february before you commit, if you survive that, you're home.",
  },
  cityLow: {
    mode: 'city',
    city: 'New York, NY',
    score: 19,
    title: "new york said you're too soft for this",
    summary:
      "you're chasing calm and this city simply does not stock it. keep visiting, the long-distance thing is working.",
    greenFlags: [
      "credit where it's due: the parts of new york that do fit you are world-class. museums you can disappear into alone, the best walking city in the country, and food from every continent at any hour. your curiosity would be fed daily, that part of your answers and this city agree completely.",
      'and when you need to be invisible in a crowd, nowhere does it better. nobody is watching you here. for someone whose social battery runs small, that anonymity can genuinely be restful, in three-day doses.',
    ],
    redFlags: [
      "everything else in your answers points the other way. you asked for quiet, space, and slow mornings; new york offers garbage trucks at 4am, 400 square feet at $4,500, and a sidewalk pace that treats strolling as an obstruction. the city's whole personality is the exact thing you said drains you.",
      "the math is a vibe of its own. to buy back the calm you want, a quieter block, a bigger place, regular weekend escapes, you'd need a budget where 'living in new york' becomes 'paying a fortune to avoid new york'. that's the actual trade on the table.",
    ],
    alternatives: [
      {
        city: 'Providence, RI',
        blurb:
          'east-coast texture without the assault. real neighborhoods, a serious food scene for its size, and boston or new york reachable by train when you want your dose on your own terms.',
      },
      {
        city: 'Hudson, NY',
        blurb:
          'two hours up the river and the volume drops 90%. one walkable main street, antique stores, mountains across the water, and an amtrak back to the city for when you miss it. you will, occasionally.',
      },
      {
        city: 'Portland, ME',
        blurb:
          'the lowkey coast. a working harbor, lobster rolls that justify the hype, winters that mean it. small enough to stay calm, salty enough to never get boring.',
      },
    ],
    closing:
      "19/100 isn't 'never', it's 'don't sign a lease'. new york is the perfect city for your three-day version and a hostile one for your tuesday version. keep it as a fling.",
  },
  match: {
    mode: 'match',
    city: 'Asheville, NC',
    title: 'certified slow-living enjoyer',
    summary: "nobody there is in a hurry, and neither are you. you'd fit in by tuesday.",
    greenFlags: [
      "asheville is what happens when a mountain town gets a record store and decides that's the ceiling. the river arts district is full of working studios you can wander into, the blue ridge parkway starts practically at the city limits, and the default friday night is a brewery patio that wraps up by ten. your answers about small crowds, big nature, and unhurried everything map onto this place almost one to one.",
      "it's also the right size for your social battery: big enough to stay anonymous at the farmers market, small enough that the barista learns your order by week two and then, crucially, leaves it at that.",
    ],
    redFlags: [
      "october is a siege. leaf season brings traffic that doubles your drive and fills every trailhead lot by 9am, and summer weekends aren't much gentler. the town you'd live in from november to may shares an address with a theme park the rest of the year.",
      'local wages never caught up with local rents, this is a bring-your-own-income town now. and the airport is small: direct flights exist, but visiting far-away friends usually means a layover in charlotte and a small prayer.',
    ],
    alternatives: [
      {
        city: 'Bend, OR',
        blurb:
          'same blueprint, trails, breweries, unbothered people, but high desert instead of blue ridge. drier, sunnier, snowier. if asheville is a flannel, bend is a puffer jacket.',
      },
      {
        city: 'Burlington, VT',
        blurb:
          "the lake-town version. church street instead of a river district, winters that demand commitment, and a town that treats 'going outside' as the entire social calendar. equally unhurried, somehow even more granola.",
      },
    ],
    closing:
      "we're confident in this one. asheville isn't a compromise between city and mountains, it just never separated the two. go in shoulder season, rent for a month, and see if you ever come back.",
  },
} satisfies Record<string, QuizResult>
