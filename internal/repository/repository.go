package repository

import (
	"context"

	"shadow_api/internal/model"
)

type SuiteRepository interface {
	CreateSuite(ctx context.Context, suite *model.Suite) error
	ListSuites(ctx context.Context) ([]model.Suite, error)
	GetSuite(ctx context.Context, id string) (*model.Suite, error)
}

type SuiteStepRepository interface {
	CreateStep(ctx context.Context, step *model.SuiteStep) error
	UpdateStep(ctx context.Context, step *model.SuiteStep) error
	DeleteStep(ctx context.Context, id string) error
	ListStepsBySuite(ctx context.Context, suiteID string) ([]model.SuiteStep, error)
}

type RunRepository interface {
	CreateRun(ctx context.Context, run *model.Run) error
	GetRun(ctx context.Context, id string) (*model.Run, error)
	UpdateRunStatus(ctx context.Context, id string, status string) error
}

type SnapshotRepository interface {
	CreateSnapshot(ctx context.Context, snapshot *model.Snapshot) error
	ListSnapshotsByRun(ctx context.Context, runID string) ([]model.Snapshot, error)
}

type DiffRepository interface {
	CreateDiff(ctx context.Context, diff *model.Diff) error
	ListDiffsByRun(ctx context.Context, runID string) ([]model.Diff, error)
}

// Repository aggregates all repository interfaces.
type Repository interface {
	SuiteRepository
	SuiteStepRepository
	RunRepository
	SnapshotRepository
	DiffRepository
}
