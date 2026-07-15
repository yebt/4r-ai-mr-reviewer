-- review_humanizations persist every humanize run so paid LLM output survives a
-- page reload. Each run is one tab: a summary rewrite (target 'summary',
-- finding_index -1) or a single finding rewrite (target 'finding', finding_index
-- = the finding position). tab_index preserves the order of runs within a
-- (review_id, target, finding_index) group.
CREATE TABLE review_humanizations (
    id            TEXT PRIMARY KEY,
    review_id     TEXT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    profile_id    TEXT NOT NULL,
    target        TEXT NOT NULL,
    finding_index INTEGER NOT NULL,
    tab_index     INTEGER NOT NULL,
    summary       TEXT NOT NULL DEFAULT '',
    issue         TEXT NOT NULL DEFAULT '',
    why           TEXT NOT NULL DEFAULT '',
    fix           TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL
);
CREATE INDEX review_humanizations_review ON review_humanizations(review_id);
