package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"shadow_api/internal/config"
	"shadow_api/internal/repository/sqlrepo"

	_ "modernc.org/sqlite"
)

// testSchema mirrors 001_init.sql for in-memory test setup (without goose dependency).
const testSchema = `
CREATE TABLE suite (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL
);
CREATE TABLE suite_step (
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
CREATE TABLE run (
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
CREATE TABLE snapshot (
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
CREATE TABLE diff (
	id TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	step_id TEXT NOT NULL,
	passed BOOLEAN NOT NULL,
	diff_json TEXT NOT NULL,
	summary TEXT NOT NULL,
	delta_latency_ms INTEGER NOT NULL DEFAULT 0,
	delta_bytes INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (run_id) REFERENCES run(id) ON DELETE CASCADE,
	FOREIGN KEY (step_id) REFERENCES suite_step(id) ON DELETE CASCADE
);
`

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(testSchema); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	cfg := &config.Config{}
	cfg.App.MaxBodyBytes = 1 << 20
	repo := sqlrepo.New(db, "sqlite")
	return New(repo, cfg)
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", resp["status"])
	}
}

func TestReady(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateAndListSuites(t *testing.T) {
	h := newTestHandler(t)

	body := bytes.NewBufferString(`{"name":"integration-suite"}`)
	req := httptest.NewRequest(http.MethodPost, "/suites", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create suite: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("expected non-empty id in create response")
	}

	req = httptest.NewRequest(http.MethodGet, "/suites", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list suites: expected 200, got %d", w.Code)
	}
	var suites []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &suites); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(suites) != 1 {
		t.Fatalf("expected 1 suite, got %d", len(suites))
	}
}

func TestGetSuiteByID(t *testing.T) {
	h := newTestHandler(t)

	// Create
	body := bytes.NewBufferString(`{"name":"get-by-id-suite"}`)
	req := httptest.NewRequest(http.MethodPost, "/suites", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created) //nolint:errcheck
	id := created["id"].(string)

	// Get by ID
	req = httptest.NewRequest(http.MethodGet, "/suites/"+id, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get suite: expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetSuiteNotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/suites/does-not-exist", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if resp["error"] == nil {
		t.Fatal("expected error field in response body")
	}
}

func TestCreateSuiteInvalidBody(t *testing.T) {
	h := newTestHandler(t)
	body := bytes.NewBufferString(`{"name":""}`)
	req := httptest.NewRequest(http.MethodPost, "/suites", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["error"] == nil {
		t.Fatal("expected error field in response body")
	}
}

func TestCreateAndListSteps(t *testing.T) {
	h := newTestHandler(t)

	// Create suite
	suiteBody := bytes.NewBufferString(`{"name":"step-suite"}`)
	req := httptest.NewRequest(http.MethodPost, "/suites", suiteBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var suite map[string]any
	json.Unmarshal(w.Body.Bytes(), &suite) //nolint:errcheck
	suiteID := suite["id"].(string)

	// Create step
	stepBody := bytes.NewBufferString(`{"position":0,"method":"GET","path":"/api/test","headers_json":"{}","body_text":""}`)
	req = httptest.NewRequest(http.MethodPost, "/suites/"+suiteID+"/steps", stepBody)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create step: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	// List steps
	req = httptest.NewRequest(http.MethodGet, "/suites/"+suiteID+"/steps", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list steps: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var steps []map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &steps); err != nil {
		t.Fatalf("decode steps: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(steps))
	}
}

func TestUpdateAndDeleteStep(t *testing.T) {
	h := newTestHandler(t)

	// Create suite
	suiteBody := bytes.NewBufferString(`{"name":"step-update-suite"}`)
	req := httptest.NewRequest(http.MethodPost, "/suites", suiteBody)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var suite map[string]any
	json.Unmarshal(w.Body.Bytes(), &suite) //nolint:errcheck
	suiteID := suite["id"].(string)

	// Create step
	stepBody := bytes.NewBufferString(`{"position":0,"method":"GET","path":"/api/v1","headers_json":"{}","body_text":""}`)
	req = httptest.NewRequest(http.MethodPost, "/suites/"+suiteID+"/steps", stepBody)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	var step map[string]any
	json.Unmarshal(w.Body.Bytes(), &step) //nolint:errcheck
	stepID := step["id"].(string)

	// Update step
	updateBody := bytes.NewBufferString(`{"position":1,"method":"POST","path":"/api/v2","headers_json":"{}","body_text":""}`)
	req = httptest.NewRequest(http.MethodPut, "/steps/"+stepID, updateBody)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update step: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete step
	req = httptest.NewRequest(http.MethodDelete, "/steps/"+stepID, nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete step: expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodDelete, "/suites", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func TestGetRunNotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/runs/nonexistent-run-id", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp) //nolint:errcheck
	if resp["error"] == nil {
		t.Fatal("expected error field")
	}
}

func TestContinueRunNotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodPost, "/runs/nonexistent/continue", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
