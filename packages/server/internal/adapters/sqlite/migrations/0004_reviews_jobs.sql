-- reviews are 4R reviews of a merge request.
CREATE TABLE reviews (
    id             TEXT PRIMARY KEY,
    repo_id        TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    mr_iid         INTEGER NOT NULL,
    context_mode   TEXT NOT NULL DEFAULT 'fast',
    status         TEXT NOT NULL,
    summary        TEXT NOT NULL DEFAULT '',
    recommendation TEXT NOT NULL DEFAULT '',
    score          INTEGER NOT NULL DEFAULT 0,
    error          TEXT NOT NULL DEFAULT '',
    input_tokens   INTEGER NOT NULL DEFAULT 0,
    output_tokens  INTEGER NOT NULL DEFAULT 0,
    created_at     TEXT NOT NULL,
    updated_at     TEXT NOT NULL
);
CREATE INDEX reviews_repo ON reviews(repo_id);

-- review_findings are the located issues of a review. position preserves the
-- order the model produced them.
CREATE TABLE review_findings (
    id         TEXT PRIMARY KEY,
    review_id  TEXT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    position   INTEGER NOT NULL,
    dimension  TEXT NOT NULL,
    severity   TEXT NOT NULL,
    file       TEXT NOT NULL DEFAULT '',
    line       INTEGER NOT NULL DEFAULT 0,
    issue      TEXT NOT NULL DEFAULT '',
    why        TEXT NOT NULL DEFAULT '',
    fix        TEXT NOT NULL DEFAULT '',
    blocking   INTEGER NOT NULL DEFAULT 0,
    published  INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX review_findings_review ON review_findings(review_id);

-- jobs drive async review execution. A retry creates a new review + new job;
-- the errored ones are kept for history.
CREATE TABLE jobs (
    id         TEXT PRIMARY KEY,
    review_id  TEXT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    status     TEXT NOT NULL,
    attempts   INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX jobs_status ON jobs(status);
