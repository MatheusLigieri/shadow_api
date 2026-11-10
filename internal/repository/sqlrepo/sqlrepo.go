package sqlrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"shadow_api/internal/model"
	"shadow_api/internal/repository"
)

type SQLRepository struct {
	db          *sql.DB
	placeholder func(int) string
}

func New(db *sql.DB, driver string) repository.Repository {
	return &SQLRepository{db: db, placeholder: placeholderFor(driver)}
}

func placeholderFor(driver string) func(int) string {
	switch strings.ToLower(driver) {
	case "postgres", "pgx":
		return func(i int) string { return fmt.Sprintf("$%d", i) }
	default:
		return func(_ int) string { return "?" }
	}
}

func (r *SQLRepository) CreateSuite(ctx context.Context, suite *model.Suite) error {
	q := "INSERT INTO suite (id, name) VALUES (" + r.placeholder(1) + ", " + r.placeholder(2) + ")"
	_, err := r.db.ExecContext(ctx, q, suite.ID, suite.Name)
	return err
}

func (r *SQLRepository) ListSuites(ctx context.Context) ([]model.Suite, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, name FROM suite ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suites []model.Suite
	for rows.Next() {
		var s model.Suite
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		suites = append(suites, s)
	}
	return suites, rows.Err()
}

func (r *SQLRepository) GetSuite(ctx context.Context, id string) (*model.Suite, error) {
	q := "SELECT id, name FROM suite WHERE id = " + r.placeholder(1)
	row := r.db.QueryRowContext(ctx, q, id)
	var s model.Suite
	if err := row.Scan(&s.ID, &s.Name); err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SQLRepository) CreateStep(ctx context.Context, step *model.SuiteStep) error {
	q := "INSERT INTO suite_step (id, suite_id, position, method, path, headers_json, body_text, extractors_json) VALUES (" +
		r.placeholder(1) + ", " + r.placeholder(2) + ", " + r.placeholder(3) + ", " + r.placeholder(4) + ", " + r.placeholder(5) + ", " + r.placeholder(6) + ", " + r.placeholder(7) + ", " + r.placeholder(8) + ")"
	_, err := r.db.ExecContext(ctx, q, step.ID, step.SuiteID, step.Position, step.Method, step.Path, step.HeadersJSON, step.BodyText, step.ExtractorsJSON)
	return err
}

func (r *SQLRepository) UpdateStep(ctx context.Context, step *model.SuiteStep) error {
	q := "UPDATE suite_step SET position = " + r.placeholder(1) + ", method = " + r.placeholder(2) + ", path = " + r.placeholder(3) + ", headers_json = " + r.placeholder(4) + ", body_text = " + r.placeholder(5) + ", extractors_json = " + r.placeholder(6) + " WHERE id = " + r.placeholder(7)
	_, err := r.db.ExecContext(ctx, q, step.Position, step.Method, step.Path, step.HeadersJSON, step.BodyText, step.ExtractorsJSON, step.ID)
	return err
}

func (r *SQLRepository) DeleteStep(ctx context.Context, id string) error {
	q := "DELETE FROM suite_step WHERE id = " + r.placeholder(1)
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *SQLRepository) ListStepsBySuite(ctx context.Context, suiteID string) ([]model.SuiteStep, error) {
	q := "SELECT id, suite_id, position, method, path, headers_json, body_text, extractors_json FROM suite_step WHERE suite_id = " + r.placeholder(1) + " ORDER BY position ASC"
	rows, err := r.db.QueryContext(ctx, q, suiteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []model.SuiteStep
	for rows.Next() {
		var s model.SuiteStep
		if err := rows.Scan(&s.ID, &s.SuiteID, &s.Position, &s.Method, &s.Path, &s.HeadersJSON, &s.BodyText, &s.ExtractorsJSON); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (r *SQLRepository) CreateRun(ctx context.Context, run *model.Run) error {
	q := "INSERT INTO run (id, suite_id, mode, status, base_a, base_b, base_single, created_at) VALUES (" +
		r.placeholder(1) + ", " + r.placeholder(2) + ", " + r.placeholder(3) + ", " + r.placeholder(4) + ", " + r.placeholder(5) + ", " + r.placeholder(6) + ", " + r.placeholder(7) + ", " + r.placeholder(8) + ")"
	_, err := r.db.ExecContext(ctx, q, run.ID, run.SuiteID, run.Mode, run.Status, run.BaseA, run.BaseB, run.BaseSingle, run.CreatedAt)
	return err
}

func (r *SQLRepository) GetRun(ctx context.Context, id string) (*model.Run, error) {
	q := "SELECT id, suite_id, mode, status, base_a, base_b, base_single, created_at FROM run WHERE id = " + r.placeholder(1)
	row := r.db.QueryRowContext(ctx, q, id)
	var run model.Run
	if err := row.Scan(&run.ID, &run.SuiteID, &run.Mode, &run.Status, &run.BaseA, &run.BaseB, &run.BaseSingle, &run.CreatedAt); err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *SQLRepository) UpdateRunStatus(ctx context.Context, id string, status string) error {
	q := "UPDATE run SET status = " + r.placeholder(1) + " WHERE id = " + r.placeholder(2)
	_, err := r.db.ExecContext(ctx, q, status, id)
	return err
}

func (r *SQLRepository) CreateSnapshot(ctx context.Context, snapshot *model.Snapshot) error {
	q := "INSERT INTO snapshot (id, run_id, step_id, variant, status_code, body, body_hash, latency_ms, bytes, created_at) VALUES (" +
		r.placeholder(1) + ", " + r.placeholder(2) + ", " + r.placeholder(3) + ", " + r.placeholder(4) + ", " + r.placeholder(5) + ", " + r.placeholder(6) + ", " + r.placeholder(7) + ", " + r.placeholder(8) + ", " + r.placeholder(9) + ", " + r.placeholder(10) + ")"
	_, err := r.db.ExecContext(ctx, q, snapshot.ID, snapshot.RunID, snapshot.StepID, snapshot.Variant, snapshot.StatusCode, snapshot.Body, snapshot.BodyHash, snapshot.LatencyMs, snapshot.Bytes, snapshot.CreatedAt)
	return err
}

func (r *SQLRepository) ListSnapshotsByRun(ctx context.Context, runID string) ([]model.Snapshot, error) {
	q := "SELECT id, run_id, step_id, variant, status_code, body, body_hash, latency_ms, bytes, created_at FROM snapshot WHERE run_id = " + r.placeholder(1) + " ORDER BY created_at ASC"
	rows, err := r.db.QueryContext(ctx, q, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var snaps []model.Snapshot
	for rows.Next() {
		var s model.Snapshot
		if err := rows.Scan(&s.ID, &s.RunID, &s.StepID, &s.Variant, &s.StatusCode, &s.Body, &s.BodyHash, &s.LatencyMs, &s.Bytes, &s.CreatedAt); err != nil {
			return nil, err
		}
		snaps = append(snaps, s)
	}
	return snaps, rows.Err()
}

func (r *SQLRepository) CreateDiff(ctx context.Context, diff *model.Diff) error {
	q := "INSERT INTO diff (id, run_id, step_id, passed, diff_json, summary, delta_latency_ms, delta_bytes) VALUES (" + r.placeholder(1) + ", " + r.placeholder(2) + ", " + r.placeholder(3) + ", " + r.placeholder(4) + ", " + r.placeholder(5) + ", " + r.placeholder(6) + ", " + r.placeholder(7) + ", " + r.placeholder(8) + ")"
	_, err := r.db.ExecContext(ctx, q, diff.ID, diff.RunID, diff.StepID, diff.Passed, diff.DiffJSON, diff.Summary, diff.DeltaLatencyMs, diff.DeltaBytes)
	return err
}

func (r *SQLRepository) ListDiffsByRun(ctx context.Context, runID string) ([]model.Diff, error) {
	q := "SELECT id, run_id, step_id, passed, diff_json, summary, delta_latency_ms, delta_bytes FROM diff WHERE run_id = " + r.placeholder(1)
	rows, err := r.db.QueryContext(ctx, q, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var diffs []model.Diff
	for rows.Next() {
		var d model.Diff
		if err := rows.Scan(&d.ID, &d.RunID, &d.StepID, &d.Passed, &d.DiffJSON, &d.Summary, &d.DeltaLatencyMs, &d.DeltaBytes); err != nil {
			return nil, err
		}
		diffs = append(diffs, d)
	}
	return diffs, rows.Err()
}
