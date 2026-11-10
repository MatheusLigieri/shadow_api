package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"shadow_api/internal/config"
	"shadow_api/internal/diff"
	"shadow_api/internal/model"
	"shadow_api/internal/repository"
)

type Runner struct {
	repo   repository.Repository
	cfg    *config.Config
	client *http.Client
}

func New(repo repository.Repository, cfg *config.Config) *Runner {
	return &Runner{
		repo:   repo,
		cfg:    cfg,
		client: &http.Client{Timeout: time.Duration(cfg.Runner.TimeoutMs) * time.Millisecond},
	}
}

func (r *Runner) RunDual(ctx context.Context, run *model.Run, steps []model.SuiteStep) error {
	for _, step := range steps {
		snapA, err := r.executeStep(ctx, run.ID, step, "A", deref(run.BaseA))
		if err != nil {
			return err
		}
		snapB, err := r.executeStep(ctx, run.ID, step, "B", deref(run.BaseB))
		if err != nil {
			return err
		}

		res := diff.CompareBodies(snapA.Body, snapB.Body, r.cfg.Runner.IgnoreJSONPaths)
		d := &model.Diff{
			ID:             uuid.NewString(),
			RunID:          run.ID,
			StepID:         step.ID,
			Passed:         res.Passed,
			DiffJSON:       res.DiffJSON,
			Summary:        res.Summary,
			DeltaLatencyMs: snapB.LatencyMs - snapA.LatencyMs,
			DeltaBytes:     snapB.Bytes - snapA.Bytes,
		}
		if err := r.repo.CreateDiff(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) RunSingleBefore(ctx context.Context, run *model.Run, steps []model.SuiteStep) error {
	for _, step := range steps {
		_, err := r.executeStep(ctx, run.ID, step, "BEFORE", deref(run.BaseSingle))
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) RunSingleAfter(ctx context.Context, run *model.Run, steps []model.SuiteStep) error {
	for _, step := range steps {
		snapBefore, err := r.latestSnapshot(ctx, run.ID, step.ID, "BEFORE")
		if err != nil {
			return err
		}
		snapAfter, err := r.executeStep(ctx, run.ID, step, "AFTER", deref(run.BaseSingle))
		if err != nil {
			return err
		}
		res := diff.CompareBodies(snapBefore.Body, snapAfter.Body, r.cfg.Runner.IgnoreJSONPaths)
		d := &model.Diff{
			ID:             uuid.NewString(),
			RunID:          run.ID,
			StepID:         step.ID,
			Passed:         res.Passed,
			DiffJSON:       res.DiffJSON,
			Summary:        res.Summary,
			DeltaLatencyMs: snapAfter.LatencyMs - snapBefore.LatencyMs,
			DeltaBytes:     snapAfter.Bytes - snapBefore.Bytes,
		}
		if err := r.repo.CreateDiff(ctx, d); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) executeStep(ctx context.Context, runID string, step model.SuiteStep, variant string, base string) (*model.Snapshot, error) {
	if base == "" {
		return nil, errors.New("base url is required")
	}

	url := strings.TrimRight(base, "/") + "/" + strings.TrimLeft(step.Path, "/")
	var body io.Reader
	if step.BodyText != "" {
		body = bytes.NewBufferString(step.BodyText)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(step.Method), url, body)
	if err != nil {
		return nil, err
	}
	if err := applyHeaders(req, step.HeadersJSON); err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	limit := r.cfg.App.MaxBodyBytes
	if limit <= 0 {
		limit = 1048576
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return nil, err
	}

	elapsed := time.Since(start)

	snap := &model.Snapshot{
		ID:         uuid.NewString(),
		RunID:      runID,
		StepID:     step.ID,
		Variant:    variant,
		StatusCode: resp.StatusCode,
		Body:       string(data),
		BodyHash:   diff.HashBody(string(data)),
		LatencyMs:  elapsed.Milliseconds(),
		Bytes:      int64(len(data)),
		CreatedAt:  time.Now().UTC(),
	}
	if err := r.repo.CreateSnapshot(ctx, snap); err != nil {
		return nil, err
	}
	return snap, nil
}

func applyHeaders(req *http.Request, headersJSON string) error {
	if strings.TrimSpace(headersJSON) == "" {
		return nil
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(headersJSON), &raw); err != nil {
		return err
	}
	for k, v := range raw {
		req.Header.Set(k, toString(v))
	}
	return nil
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		return fmt.Sprint(t)
	}
}

func (r *Runner) latestSnapshot(ctx context.Context, runID string, stepID string, variant string) (*model.Snapshot, error) {
	all, err := r.repo.ListSnapshotsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	for i := len(all) - 1; i >= 0; i-- {
		if all[i].StepID == stepID && all[i].Variant == variant {
			return &all[i], nil
		}
	}
	return nil, errors.New("snapshot not found")
}

func deref(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
