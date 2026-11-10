-- +goose Up
CREATE TABLE IF NOT EXISTS suite (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS suite_step (
	id TEXT PRIMARY KEY,
	suite_id TEXT NOT NULL,
	position INTEGER NOT NULL,
	method TEXT NOT NULL,
	path TEXT NOT NULL,
	headers_json TEXT NOT NULL,
	body_text TEXT NOT NULL,
	extractors_json TEXT,
	FOREIGN KEY (suite_id) REFERENCES suite(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS run (
	id TEXT PRIMARY KEY,
	suite_id TEXT NOT NULL,
	mode TEXT NOT NULL CHECK (mode IN ('DUAL', 'SINGLE')),
	status TEXT NOT NULL CHECK (status IN ('RUNNING', 'WAITING_USER', 'COMPLETED')),
	base_a TEXT,
	base_b TEXT,
	base_single TEXT,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (suite_id) REFERENCES suite(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS snapshot (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	variant TEXT NOT NULL CHECK (variant IN ('A', 'B', 'BEFORE', 'AFTER')),
	status_code INTEGER NOT NULL,
	body TEXT NOT NULL,
	body_hash TEXT NOT NULL,
	latency_ms INTEGER NOT NULL,
	bytes INTEGER NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (run_id) REFERENCES run(id) ON DELETE CASCADE,
	FOREIGN KEY (step_id) REFERENCES suite_step(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS diff (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	passed BOOLEAN NOT NULL,
	diff_json TEXT NOT NULL,
	summary TEXT NOT NULL,
	FOREIGN KEY (run_id) REFERENCES run(id) ON DELETE CASCADE,
	FOREIGN KEY (step_id) REFERENCES suite_step(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_suite_step_suite_id ON suite_step(suite_id);
CREATE INDEX IF NOT EXISTS idx_run_suite_id ON run(suite_id);
CREATE INDEX IF NOT EXISTS idx_snapshot_run_id ON snapshot(run_id);
CREATE INDEX IF NOT EXISTS idx_snapshot_step_id ON snapshot(step_id);
CREATE INDEX IF NOT EXISTS idx_diff_run_id ON diff(run_id);
CREATE INDEX IF NOT EXISTS idx_diff_step_id ON diff(step_id);

-- +goose Down
DROP TABLE IF EXISTS diff;
DROP TABLE IF EXISTS snapshot;
DROP TABLE IF EXISTS run;
DROP TABLE IF EXISTS suite_step;
DROP TABLE IF EXISTS suite;
