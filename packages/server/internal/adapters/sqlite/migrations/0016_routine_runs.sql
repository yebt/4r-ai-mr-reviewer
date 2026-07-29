-- routine_run is a resumable, checkpointed sequence of GitLab actions on a repo
-- (e.g. approve_and_tag). params_json is the immutable input, state_json the
-- mutable accumulator, and steps_json the per-step ledger that lets a resume
-- skip already-completed (irreversible) actions.
CREATE TABLE routine_run (
    id          TEXT PRIMARY KEY,
    kind        TEXT NOT NULL,
    repo_id     TEXT NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    mr_iid      INTEGER,
    status      TEXT NOT NULL,
    params_json TEXT NOT NULL DEFAULT '{}',
    state_json  TEXT NOT NULL DEFAULT '{}',
    steps_json  TEXT NOT NULL DEFAULT '[]',
    last_error  TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL,
    updated_at  TEXT NOT NULL
);
CREATE INDEX routine_run_repo ON routine_run(repo_id);
CREATE INDEX routine_run_status ON routine_run(status);
