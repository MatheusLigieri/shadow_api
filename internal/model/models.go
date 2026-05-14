package model

import "time"

type Suite struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SuiteStep struct {
	ID             string  `json:"id"`
	SuiteID        string  `json:"suite_id"`
	Position       int     `json:"position"`
	Method         string  `json:"method"`
	Path           string  `json:"path"`
	HeadersJSON    string  `json:"headers_json"`
	BodyText       string  `json:"body_text"`
	ExtractorsJSON *string `json:"extractors_json,omitempty"`
}

type Run struct {
	ID         string    `json:"id"`
	SuiteID    string    `json:"suite_id"`
	Mode       string    `json:"mode"`
	Status     string    `json:"status"`
	BaseA      *string   `json:"base_a,omitempty"`
	BaseB      *string   `json:"base_b,omitempty"`
	BaseSingle *string   `json:"base_single,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Snapshot struct {
	ID         string    `json:"id"`
	RunID      string    `json:"run_id"`
	StepID     string    `json:"step_id"`
	Variant    string    `json:"variant"`
	StatusCode int       `json:"status_code"`
	Body       string    `json:"body"`
	BodyHash   string    `json:"body_hash"`
	LatencyMs  int64     `json:"latency_ms"`
	Bytes      int64     `json:"bytes"`
	CreatedAt  time.Time `json:"created_at"`
}

type Diff struct {
	ID       string `json:"id"`
	RunID    string `json:"run_id"`
	StepID   string `json:"step_id"`
	Passed   bool   `json:"passed"`
	DiffJSON string `json:"diff_json"`
	Summary  string `json:"summary"`
}
