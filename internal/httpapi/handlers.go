package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"shadow_api/internal/config"
	"shadow_api/internal/model"
	"shadow_api/internal/repository"
	"shadow_api/internal/runner"
	"shadow_api/internal/validation"
)

type Handler struct {
	repo   repository.Repository
	runner *runner.Runner
}

func New(repo repository.Repository, cfg *config.Config) http.Handler {
	h := &Handler{repo: repo, runner: runner.New(repo, cfg)}
	mux := http.NewServeMux()

	mux.HandleFunc("/health", h.handleHealth)
	mux.HandleFunc("/ready", h.handleReady)
	mux.HandleFunc("/suites", h.handleSuites)
	mux.HandleFunc("/suites/", h.handleSuitesByID)
	mux.HandleFunc("/steps/", h.handleStepsByID)
	mux.HandleFunc("/runs/dual", h.handleRunDual)
	mux.HandleFunc("/runs/single/start", h.handleRunSingleStart)
	mux.HandleFunc("/runs/", h.handleRuns)
	mux.Handle("/docs/", serveSwagger())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, errors.New("not found"))
	})

	return corsMiddleware(contentTypeMiddleware(mux))
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	// Readiness checks DB connectivity by querying an existing lightweight operation.
	if _, err := h.repo.ListSuites(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, errors.New("database not ready"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) handleSuites(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.listSuites(w, r)
	case http.MethodPost:
		h.createSuite(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) handleSuitesByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/suites/")
	if path == "" {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}

	if strings.HasSuffix(path, "/steps") {
		suiteID := strings.TrimSuffix(path, "/steps")
		if suiteID == "" {
			writeError(w, http.StatusNotFound, errors.New("not found"))
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.listSteps(w, r, suiteID)
		case http.MethodPost:
			h.createStep(w, r, suiteID)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
		return
	}

	if strings.Contains(path, "/") {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.getSuite(w, r, path)
	default:
		writeMethodNotAllowed(w, http.MethodGet)
	}
}

func (h *Handler) handleStepsByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/steps/")
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}

	switch r.Method {
	case http.MethodPut:
		h.updateStep(w, r, id)
	case http.MethodDelete:
		h.deleteStep(w, r, id)
	default:
		writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
	}
}

func (h *Handler) handleRunDual(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		SuiteID string `json:"suite_id"`
		BaseA   string `json:"base_a"`
		BaseB   string `json:"base_b"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validation.ValidateRequiredString(req.SuiteID, "suite_id"); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validation.ValidateURL(req.BaseA); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("base_a: "+err.Error()))
		return
	}
	if err := validation.ValidateURL(req.BaseB); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("base_b: "+err.Error()))
		return
	}

	if _, err := h.repo.GetSuite(r.Context(), req.SuiteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("suite not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	baseA := req.BaseA
	baseB := req.BaseB
	run := &model.Run{
		ID:        uuid.NewString(),
		SuiteID:   req.SuiteID,
		Mode:      "DUAL",
		Status:    "RUNNING",
		BaseA:     &baseA,
		BaseB:     &baseB,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.repo.CreateRun(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	steps, err := h.repo.ListStepsBySuite(r.Context(), req.SuiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.runner.RunDual(r.Context(), run, steps); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_ = h.repo.UpdateRunStatus(r.Context(), run.ID, "COMPLETED")
	run.Status = "COMPLETED"
	writeJSON(w, http.StatusCreated, run)
}

func (h *Handler) handleRunSingleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		SuiteID string `json:"suite_id"`
		Base    string `json:"base"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validation.ValidateRequiredString(req.SuiteID, "suite_id"); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validation.ValidateURL(req.Base); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("base: "+err.Error()))
		return
	}

	if _, err := h.repo.GetSuite(r.Context(), req.SuiteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("suite not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	base := req.Base
	run := &model.Run{
		ID:         uuid.NewString(),
		SuiteID:    req.SuiteID,
		Mode:       "SINGLE",
		Status:     "RUNNING",
		BaseSingle: &base,
		CreatedAt:  time.Now().UTC(),
	}
	if err := h.repo.CreateRun(r.Context(), run); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	steps, err := h.repo.ListStepsBySuite(r.Context(), req.SuiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.runner.RunSingleBefore(r.Context(), run, steps); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_ = h.repo.UpdateRunStatus(r.Context(), run.ID, "WAITING_USER")
	run.Status = "WAITING_USER"
	writeJSON(w, http.StatusCreated, run)
}

func (h *Handler) handleRuns(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/runs/")
	if path == "" {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}

	if strings.HasSuffix(path, "/continue") {
		runID := strings.TrimSuffix(path, "/continue")
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		h.continueRun(w, r, runID)
		return
	}

	if strings.HasSuffix(path, "/diffs") {
		runID := strings.TrimSuffix(path, "/diffs")
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		h.listDiffs(w, r, runID)
		return
	}

	if strings.Contains(path, "/") {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}

	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}
	h.getRun(w, r, path)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request, id string) {
	run, err := h.repo.GetRun(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("run not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// snapshotView exposes a compared response. Body is returned as raw JSON when
// the stored body is valid JSON, otherwise as a plain string.
type snapshotView struct {
	Variant    string `json:"variant"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	Bytes      int64  `json:"bytes"`
	BodyHash   string `json:"body_hash"`
	Body       any    `json:"body"`
}

type diffView struct {
	model.Diff
	Before *snapshotView `json:"before"`
	After  *snapshotView `json:"after"`
}

func (h *Handler) listDiffs(w http.ResponseWriter, r *http.Request, runID string) {
	run, err := h.repo.GetRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("run not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	diffs, err := h.repo.ListDiffsByRun(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	snaps, err := h.repo.ListSnapshotsByRun(r.Context(), runID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	// Latest snapshot per step+variant (snapshots are ordered by created_at).
	latest := make(map[string]model.Snapshot, len(snaps))
	for _, s := range snaps {
		latest[s.StepID+"|"+s.Variant] = s
	}
	beforeVariant, afterVariant := "A", "B"
	if run.Mode == "SINGLE" {
		beforeVariant, afterVariant = "BEFORE", "AFTER"
	}

	views := make([]diffView, 0, len(diffs))
	for _, d := range diffs {
		v := diffView{Diff: d}
		if s, ok := latest[d.StepID+"|"+beforeVariant]; ok {
			v.Before = newSnapshotView(s)
		}
		if s, ok := latest[d.StepID+"|"+afterVariant]; ok {
			v.After = newSnapshotView(s)
		}
		views = append(views, v)
	}
	writeJSON(w, http.StatusOK, views)
}

func newSnapshotView(s model.Snapshot) *snapshotView {
	var body any = s.Body
	if json.Valid([]byte(s.Body)) {
		body = json.RawMessage(s.Body)
	}
	return &snapshotView{
		Variant:    s.Variant,
		StatusCode: s.StatusCode,
		LatencyMs:  s.LatencyMs,
		Bytes:      s.Bytes,
		BodyHash:   s.BodyHash,
		Body:       body,
	}
}

func (h *Handler) continueRun(w http.ResponseWriter, r *http.Request, runID string) {
	run, err := h.repo.GetRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("run not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if run.Mode != "SINGLE" || run.Status != "WAITING_USER" {
		writeError(w, http.StatusBadRequest, errors.New("run is not in WAITING_USER state"))
		return
	}

	steps, err := h.repo.ListStepsBySuite(r.Context(), run.SuiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := h.runner.RunSingleAfter(r.Context(), run, steps); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_ = h.repo.UpdateRunStatus(r.Context(), run.ID, "COMPLETED")
	run.Status = "COMPLETED"
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) createSuite(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, errors.New("name is required"))
		return
	}

	suite := &model.Suite{ID: uuid.NewString(), Name: req.Name}
	if err := h.repo.CreateSuite(r.Context(), suite); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, suite)
}

func (h *Handler) listSuites(w http.ResponseWriter, r *http.Request) {
	suites, err := h.repo.ListSuites(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, suites)
}

func (h *Handler) getSuite(w http.ResponseWriter, r *http.Request, id string) {
	suite, err := h.repo.GetSuite(r.Context(), id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, errors.New("suite not found"))
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, suite)
}

func (h *Handler) createStep(w http.ResponseWriter, r *http.Request, suiteID string) {
	var req struct {
		Position       int     `json:"position"`
		Method         string  `json:"method"`
		Path           string  `json:"path"`
		HeadersJSON    string  `json:"headers_json"`
		BodyText       string  `json:"body_text"`
		ExtractorsJSON *string `json:"extractors_json"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validation.ValidateHTTPMethod(req.Method); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("method: "+err.Error()))
		return
	}
	if err := validation.ValidateRequiredString(req.Path, "path"); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.HeadersJSON) != "" {
		if err := validation.ValidateJSONString(req.HeadersJSON); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("headers_json: "+err.Error()))
			return
		}
	}
	if req.ExtractorsJSON != nil && strings.TrimSpace(*req.ExtractorsJSON) != "" {
		if err := validation.ValidateJSONString(*req.ExtractorsJSON); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("extractors_json: "+err.Error()))
			return
		}
	}

	step := &model.SuiteStep{
		ID:             uuid.NewString(),
		SuiteID:        suiteID,
		Position:       req.Position,
		Method:         strings.ToUpper(req.Method),
		Path:           req.Path,
		HeadersJSON:    req.HeadersJSON,
		BodyText:       req.BodyText,
		ExtractorsJSON: req.ExtractorsJSON,
	}
	if err := h.repo.CreateStep(r.Context(), step); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusCreated, step)
}

func (h *Handler) listSteps(w http.ResponseWriter, r *http.Request, suiteID string) {
	steps, err := h.repo.ListStepsBySuite(r.Context(), suiteID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, steps)
}

func (h *Handler) updateStep(w http.ResponseWriter, r *http.Request, id string) {
	var req struct {
		Position       int     `json:"position"`
		Method         string  `json:"method"`
		Path           string  `json:"path"`
		HeadersJSON    string  `json:"headers_json"`
		BodyText       string  `json:"body_text"`
		ExtractorsJSON *string `json:"extractors_json"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validation.ValidateHTTPMethod(req.Method); err != nil {
		writeError(w, http.StatusBadRequest, errors.New("method: "+err.Error()))
		return
	}
	if err := validation.ValidateRequiredString(req.Path, "path"); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.HeadersJSON) != "" {
		if err := validation.ValidateJSONString(req.HeadersJSON); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("headers_json: "+err.Error()))
			return
		}
	}
	if req.ExtractorsJSON != nil && strings.TrimSpace(*req.ExtractorsJSON) != "" {
		if err := validation.ValidateJSONString(*req.ExtractorsJSON); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("extractors_json: "+err.Error()))
			return
		}
	}

	step := &model.SuiteStep{
		ID:             id,
		Position:       req.Position,
		Method:         strings.ToUpper(req.Method),
		Path:           req.Path,
		HeadersJSON:    req.HeadersJSON,
		BodyText:       req.BodyText,
		ExtractorsJSON: req.ExtractorsJSON,
	}
	if err := h.repo.UpdateStep(r.Context(), step); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, step)
}

func (h *Handler) deleteStep(w http.ResponseWriter, r *http.Request, id string) {
	if err := h.repo.DeleteStep(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error":     err.Error(),
		"timestamp": time.Now().UTC(),
	})
}

func writeMethodNotAllowed(w http.ResponseWriter, allowed ...string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}
