-- review_reasonings persist the model's captured chain-of-thought per 4R phase
-- of a review, upserted live as each phase completes so the reasoning can be
-- surfaced while the review is still running. One row per (review, phase).
CREATE TABLE review_reasonings (
  review_id  TEXT NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
  phase      TEXT NOT NULL,
  content    TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (review_id, phase)
);
