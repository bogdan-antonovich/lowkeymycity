-- +goose Up

INSERT INTO match_questions (meaning_id, position, text, options) VALUES
('climate', 0, 'first, weather. what can you actually live with, long term?', '[
    {"id":"a","text":"eternal drizzle, i''m basically a houseplant"},
    {"id":"b","text":"heat. always. i don''t own a real jacket"},
    {"id":"c","text":"four seasons, i like the plot twists"},
    {"id":"d","text":"mild and boring — weather should be background noise"}
]'),
('pace', 1, 'it''s 8am on a wednesday. the ideal version of you is:', '[
    {"id":"a","text":"already at a coffee shop, headphones in"},
    {"id":"b","text":"asleep. obviously."},
    {"id":"c","text":"on a trail before work like a maniac"},
    {"id":"d","text":"halfway through emails on a packed train and weirdly fine"}
]'),
('social', 2, 'your neighbor spots you at the mailbox. best case scenario:', '[
    {"id":"a","text":"a nod. maybe. from a distance"},
    {"id":"b","text":"quick chat about nothing, then freedom"},
    {"id":"c","text":"we''re actually friends, they''re coming over later"},
    {"id":"d","text":"what neighbor? never seen them. perfect"}
]'),
('night', 3, 'it''s 11:47pm on a friday. be honest, where are you?', '[
    {"id":"a","text":"home. horizontal. thriving"},
    {"id":"b","text":"out, but it''s a food thing, not a party thing"},
    {"id":"c","text":"just getting started, don''t text me"},
    {"id":"d","text":"night walk with one friend, deep convos only"}
]'),
('nature', 4, 'pick the view from your window:', '[
    {"id":"a","text":"mountains, just doing their thing"},
    {"id":"b","text":"water. any water"},
    {"id":"c","text":"rooftops and city lights"},
    {"id":"d","text":"trees and a neighbor''s suspiciously perfect garden"}
]'),
('budget', 5, 'real talk — monthly rent that lets you sleep at night:', '[
    {"id":"a","text":"under $1,200 and proud of it"},
    {"id":"b","text":"$1,500-ish if the city earns it"},
    {"id":"c","text":"$2,000+ for the right life"},
    {"id":"d","text":"whatever it takes, i''ll figure it out"}
]'),
('food', 6, 'how much does the food scene actually matter?', '[
    {"id":"a","text":"it''s the whole point of leaving the house"},
    {"id":"b","text":"a few solid spots is plenty"},
    {"id":"c","text":"i cook. restaurants are for birthdays"},
    {"id":"d","text":"i just need one good late-night place that knows me"}
]'),
('chaos', 7, 'the train is 25 minutes late and it''s loud everywhere. you:', '[
    {"id":"a","text":"headphones up, honestly didn''t notice"},
    {"id":"b","text":"annoyed but alive, this is fine"},
    {"id":"c","text":"rage-texting the group chat in real time"},
    {"id":"d","text":"this is exactly why i drive everywhere"}
]'),
('transit', 8, 'how do you want to get around, ideally?', '[
    {"id":"a","text":"walk everywhere or it doesn''t count"},
    {"id":"b","text":"bike lanes are a personality and i have it"},
    {"id":"c","text":"trains and buses, leave me alone to read"},
    {"id":"d","text":"my car, my playlist, my rules"}
]'),
('aesthetic', 9, 'pick a downtown:', '[
    {"id":"a","text":"old brick, cobblestone, slightly haunted"},
    {"id":"b","text":"glass towers, clean lines, fast walkers"},
    {"id":"c","text":"beach town main street, salt in the air"},
    {"id":"d","text":"low buildings, murals, plants on everything"}
]'),
('energy', 10, 'at the function, you are:', '[
    {"id":"a","text":"the host''s calm friend everyone trusts"},
    {"id":"b","text":"in the kitchen with the dog"},
    {"id":"c","text":"running the playlist, obviously"},
    {"id":"d","text":"left an hour ago, told no one"}
]'),
('dealbreaker', 11, 'instant dealbreaker in a city — pick the worst one:', '[
    {"id":"a","text":"winters that go full survival mode"},
    {"id":"b","text":"needing a car for literally everything"},
    {"id":"c","text":"rent that eats half the paycheck"},
    {"id":"d","text":"everything closes at 9pm"}
]');

-- +goose Down

DELETE FROM match_questions;
