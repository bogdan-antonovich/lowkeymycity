-- +goose Up

-- LLM-generated questions, cached per city after the first quiz request
CREATE TABLE city_questions (
    city_label TEXT    NOT NULL REFERENCES cities(label),
    position   INTEGER NOT NULL,
    meaning_id TEXT    NOT NULL,
    text       TEXT    NOT NULL,
    options    JSONB   NOT NULL,
    PRIMARY KEY (city_label, meaning_id)
);

-- the static question bank for match mode
CREATE TABLE match_questions (
    meaning_id TEXT    PRIMARY KEY,
    position   INTEGER NOT NULL,
    text       TEXT    NOT NULL,
    options    JSONB   NOT NULL
);

-- one row per unique answer combination; the combination is the canonical
-- JSON of mode+city+answers, so identical submissions reuse the stored result
CREATE TABLE results (
    id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    combination  TEXT UNIQUE NOT NULL,
    mode         TEXT NOT NULL,
    city         TEXT NOT NULL,
    score        INTEGER NOT NULL DEFAULT 0,
    title        TEXT NOT NULL,
    summary      TEXT NOT NULL,
    green_flags  TEXT[] NOT NULL,
    red_flags    TEXT[] NOT NULL,
    alternatives JSONB NOT NULL,
    closing      TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE results;
DROP TABLE match_questions;
DROP TABLE city_questions;
