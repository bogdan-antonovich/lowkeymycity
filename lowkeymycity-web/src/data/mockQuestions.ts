import type { QuizQuestion } from '@/types/quiz'

// Mock question sets. The real ones come from GET /api/quiz/questions —
// these define the tone and structure that endpoint should produce.
// One question per vibe axis, scenarios over survey-speak, 4 options each,
// and every option has to be a valid personality (nobody feels judged).

export const MATCH_QUESTIONS: QuizQuestion[] = [
  {
    id: 'climate',
    text: 'first, weather. what can you actually live with, long term?',
    options: [
      "eternal drizzle, i'm basically a houseplant",
      "heat. always. i don't own a real jacket",
      'four seasons, i like the plot twists',
      'mild and boring — weather should be background noise',
    ],
  },
  {
    id: 'pace',
    text: "it's 8am on a wednesday. the ideal version of you is:",
    options: [
      'already at a coffee shop, headphones in',
      'asleep. obviously.',
      'on a trail before work like a maniac',
      'halfway through emails on a packed train and weirdly fine',
    ],
  },
  {
    id: 'social',
    text: 'your neighbor spots you at the mailbox. best case scenario:',
    options: [
      'a nod. maybe. from a distance',
      'quick chat about nothing, then freedom',
      "we're actually friends, they're coming over later",
      'what neighbor? never seen them. perfect',
    ],
  },
  {
    id: 'night',
    text: "it's 11:47pm on a friday. be honest, where are you?",
    options: [
      'home. horizontal. thriving',
      "out, but it's a food thing, not a party thing",
      "just getting started, don't text me",
      'night walk with one friend, deep convos only',
    ],
  },
  {
    id: 'nature',
    text: 'pick the view from your window:',
    options: [
      'mountains, just doing their thing',
      'water. any water',
      'rooftops and city lights',
      "trees and a neighbor's suspiciously perfect garden",
    ],
  },
  {
    id: 'budget',
    text: 'real talk — monthly rent that lets you sleep at night:',
    options: [
      'under $1,200 and proud of it',
      '$1,500-ish if the city earns it',
      '$2,000+ for the right life',
      "whatever it takes, i'll figure it out",
    ],
  },
  {
    id: 'food',
    text: 'how much does the food scene actually matter?',
    options: [
      "it's the whole point of leaving the house",
      'a few solid spots is plenty',
      'i cook. restaurants are for birthdays',
      'i just need one good late-night place that knows me',
    ],
  },
  {
    id: 'chaos',
    text: "the train is 25 minutes late and it's loud everywhere. you:",
    options: [
      "headphones up, honestly didn't notice",
      'annoyed but alive, this is fine',
      'rage-texting the group chat in real time',
      'this is exactly why i drive everywhere',
    ],
  },
  {
    id: 'transit',
    text: 'how do you want to get around, ideally?',
    options: [
      "walk everywhere or it doesn't count",
      'bike lanes are a personality and i have it',
      'trains and buses, leave me alone to read',
      'my car, my playlist, my rules',
    ],
  },
  {
    id: 'aesthetic',
    text: 'pick a downtown:',
    options: [
      'old brick, cobblestone, slightly haunted',
      'glass towers, clean lines, fast walkers',
      'beach town main street, salt in the air',
      'low buildings, murals, plants on everything',
    ],
  },
  {
    id: 'energy',
    text: 'at the function, you are:',
    options: [
      "the host's calm friend everyone trusts",
      'in the kitchen with the dog',
      'running the playlist, obviously',
      'left an hour ago, told no one',
    ],
  },
  {
    id: 'dealbreaker',
    text: 'instant dealbreaker in a city — pick the worst one:',
    options: [
      'winters that go full survival mode',
      'needing a car for literally everything',
      'rent that eats half the paycheck',
      'everything closes at 9pm',
    ],
  },
]

export function cityQuestions(city: string): QuizQuestion[] {
  const name = city.split(',')[0].toLowerCase()
  return [
    {
      id: 'climate',
      text: `be honest — when ${name} weather acts up, you:`,
      options: [
        'dress for it and move on',
        "complain loudly but secretly don't mind",
        'let it ruin the entire day',
        'cancel everything and hibernate',
      ],
    },
    {
      id: 'pace',
      text: `the pace of a city feels right when:`,
      options: [
        'mornings are slow and nobody rushes me',
        "there's a steady hum but room to breathe",
        'things move fast and i move faster',
        'depends entirely on my caffeine level',
      ],
    },
    {
      id: 'social',
      text: `you run into someone you know at the store in ${name}. you:`,
      options: [
        'love it, this is why i live here',
        'nice 30-second chat, then escape',
        'suddenly very interested in the cereal aisle',
        'this is my nightmare. anonymity please',
      ],
    },
    {
      id: 'night',
      text: `a perfect ${name} friday night ends at:`,
      options: [
        '9pm, couch, victory',
        'midnight, after food with people i like',
        '3am, zero regrets',
        'sunrise. technically saturday',
      ],
    },
    {
      id: 'nature',
      text: 'how often do you actually need to touch grass?',
      options: [
        'daily or i malfunction',
        'weekends, minimum',
        'a park bench now and then is fine',
        'the city IS my nature',
      ],
    },
    {
      id: 'budget',
      text: `money check — rent in ${name} should feel like:`,
      options: [
        "barely noticeable, i'm saving",
        'fair for what i get',
        'a stretch, but worth it',
        'i stopped looking at the number',
      ],
    },
    {
      id: 'food',
      text: `eating out in ${name} — what's your honest pattern?`,
      options: [
        "a few times a week, it's a lifestyle",
        'weekly treat, chosen very carefully',
        'mostly cook, occasionally wander',
        'late-night spots only',
      ],
    },
    {
      id: 'chaos',
      text: 'traffic, crowds, sirens — where is your tolerance meter?',
      options: [
        'zero. i require calm',
        'low, but headphones fix it',
        "medium, it's part of the deal",
        'i honestly feed on it',
      ],
    },
    {
      id: 'transit',
      text: `getting around ${name}, ideal version:`,
      options: [
        'my own two feet',
        'bike, weather permitting',
        'public transit and a podcast',
        'car, always, no debate',
      ],
    },
    {
      id: 'aesthetic',
      text: 'what do you notice first in a city?',
      options: [
        'old buildings with stories',
        'the skyline at night',
        'green streets and front porches',
        'weird local details — murals, signs, gnomes',
      ],
    },
    {
      id: 'energy',
      text: `in ${name}, you'd rather be known as:`,
      options: [
        'the regular at one quiet spot',
        'the friend who knows every hidden gem',
        'a familiar face at every event',
        'completely unknown. a ghost. a legend',
      ],
    },
    {
      id: 'dealbreaker',
      text: `what would actually make you leave ${name}?`,
      options: [
        'rent going feral',
        'the weather winning',
        'the vibe turning corporate',
        'everything closing early',
      ],
    },
  ]
}
