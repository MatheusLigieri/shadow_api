package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin=*, got %q", got)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %v", resp["status"])
	}
}

func TestCORSPreflight(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodOptions, "/suites", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set("Access-Control-Request-Headers", "Content-Type, X-Custom-Header")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin=*, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS" {
		t.Fatalf("unexpected Access-Control-Allow-Methods: %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, X-Custom-Header" {
		t.Fatalf("unexpected Access-Control-Allow-Headers: %q", got)
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

func TestDualRunDiffsReturnFullBodies(t *testing.T) {
	upstreamA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"price":100,"items":[{"id":1}]}`)) //nolint:errcheck
	}))
	defer upstreamA.Close()
	upstreamB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"price":120,"items":[{"id":1}]}`)) //nolint:errcheck
	}))
	defer upstreamB.Close()

	h := newTestHandler(t)
	do := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body) //nolint:errcheck
		}
		req := httptest.NewRequest(method, path, &buf)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	var suite map[string]any
	json.Unmarshal(do(http.MethodPost, "/suites", map[string]any{"name": "s"}).Body.Bytes(), &suite) //nolint:errcheck
	suiteID := suite["id"].(string)

	if w := do(http.MethodPost, "/suites/"+suiteID+"/steps", map[string]any{
		"position": 1, "method": "GET", "path": "/product",
	}); w.Code != http.StatusCreated {
		t.Fatalf("create step: %d %s", w.Code, w.Body.String())
	}

	w := do(http.MethodPost, "/runs/dual", map[string]any{
		"suite_id": suiteID, "base_a": upstreamA.URL, "base_b": upstreamB.URL,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("dual run: %d %s", w.Code, w.Body.String())
	}
	var run map[string]any
	json.Unmarshal(w.Body.Bytes(), &run) //nolint:errcheck

	w = do(http.MethodGet, "/runs/"+run["id"].(string)+"/diffs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("diffs: %d %s", w.Code, w.Body.String())
	}
	var diffs []struct {
		Passed bool `json:"passed"`
		Before struct {
			Variant    string         `json:"variant"`
			StatusCode int            `json:"status_code"`
			Body       map[string]any `json:"body"`
		} `json:"before"`
		After struct {
			Variant string         `json:"variant"`
			Body    map[string]any `json:"body"`
		} `json:"after"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &diffs); err != nil {
		t.Fatalf("decode diffs: %v: %s", err, w.Body.String())
	}
	if len(diffs) != 1 {
		t.Fatalf("expected 1 diff, got %d", len(diffs))
	}
	d := diffs[0]
	if d.Passed {
		t.Fatal("expected diff to fail")
	}
	if d.Before.Variant != "A" || d.After.Variant != "B" {
		t.Fatalf("unexpected variants: %q / %q", d.Before.Variant, d.After.Variant)
	}
	if d.Before.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status code: %d", d.Before.StatusCode)
	}
	if d.Before.Body["price"] != float64(100) || d.After.Body["price"] != float64(120) {
		t.Fatalf("unexpected bodies: %v / %v", d.Before.Body, d.After.Body)
	}
	if _, ok := d.After.Body["items"].([]any); !ok {
		t.Fatalf("expected full body with items, got %v", d.After.Body)
	}
}

func TestSingleRunDiffsReturnBeforeAfter(t *testing.T) {
	var phase atomic.Value
	phase.Store("before")
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/text" {
			fmt.Fprintf(w, "phase=%s", phase.Load())
			return
		}
		fmt.Fprintf(w, `{"phase":%q}`, phase.Load())
	}))
	defer upstream.Close()

	h := newTestHandler(t)
	do := func(method, path string, body any) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		if body != nil {
			json.NewEncoder(&buf).Encode(body) //nolint:errcheck
		}
		req := httptest.NewRequest(method, path, &buf)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}

	var suite map[string]any
	json.Unmarshal(do(http.MethodPost, "/suites", map[string]any{"name": "s"}).Body.Bytes(), &suite) //nolint:errcheck
	suiteID := suite["id"].(string)

	stepPaths := map[string]string{}
	for i, path := range []string{"/json", "/text"} {
		w := do(http.MethodPost, "/suites/"+suiteID+"/steps", map[string]any{
			"position": i + 1, "method": "GET", "path": path,
		})
		if w.Code != http.StatusCreated {
			t.Fatalf("create step: %d %s", w.Code, w.Body.String())
		}
		var step map[string]any
		json.Unmarshal(w.Body.Bytes(), &step) //nolint:errcheck
		stepPaths[step["id"].(string)] = path
	}

	w := do(http.MethodPost, "/runs/single/start", map[string]any{"suite_id": suiteID, "base": upstream.URL})
	if w.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", w.Code, w.Body.String())
	}
	var run map[string]any
	json.Unmarshal(w.Body.Bytes(), &run) //nolint:errcheck
	runID := run["id"].(string)

	phase.Store("after")
	if w := do(http.MethodPost, "/runs/"+runID+"/continue", nil); w.Code != http.StatusOK {
		t.Fatalf("continue: %d %s", w.Code, w.Body.String())
	}

	w = do(http.MethodGet, "/runs/"+runID+"/diffs", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("diffs: %d %s", w.Code, w.Body.String())
	}
	var diffs []struct {
		StepID string `json:"step_id"`
		Before struct {
			Variant string `json:"variant"`
			Body    any    `json:"body"`
		} `json:"before"`
		After struct {
			Variant string `json:"variant"`
			Body    any    `json:"body"`
		} `json:"after"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &diffs); err != nil {
		t.Fatalf("decode diffs: %v: %s", err, w.Body.String())
	}
	if len(diffs) != 2 {
		t.Fatalf("expected 2 diffs, got %d: %s", len(diffs), w.Body.String())
	}
	for _, d := range diffs {
		if d.Before.Variant != "BEFORE" || d.After.Variant != "AFTER" {
			t.Fatalf("unexpected variants: %q / %q", d.Before.Variant, d.After.Variant)
		}
		switch stepPaths[d.StepID] {
		case "/json":
			before, ok := d.Before.Body.(map[string]any)
			after, ok2 := d.After.Body.(map[string]any)
			if !ok || !ok2 || before["phase"] != "before" || after["phase"] != "after" {
				t.Fatalf("expected JSON bodies, got %#v / %#v", d.Before.Body, d.After.Body)
			}
		case "/text":
			if d.Before.Body != "phase=before" || d.After.Body != "phase=after" {
				t.Fatalf("expected string bodies, got %#v / %#v", d.Before.Body, d.After.Body)
			}
		default:
			t.Fatalf("unexpected step id %q", d.StepID)
		}
	}
}

func assertJSONError(t *testing.T, w *httptest.ResponseRecorder) string {
	t.Helper()
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var resp struct {
		Error     string `json:"error"`
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode error body: %v: %q", err, w.Body.String())
	}
	if resp.Error == "" || resp.Timestamp == "" {
		t.Fatalf("expected error and timestamp, got %q", w.Body.String())
	}
	return resp.Error
}

func TestMethodNotAllowedUsesJSONError(t *testing.T) {
	h := newTestHandler(t)
	cases := []struct{ method, path, allow string }{
		{http.MethodPost, "/health", "GET"},
		{http.MethodPost, "/ready", "GET"},
		{http.MethodDelete, "/suites", "GET, POST"},
		{http.MethodPost, "/suites/x", "GET"},
		{http.MethodDelete, "/suites/x/steps", "GET, POST"},
		{http.MethodGet, "/steps/x", "PUT, DELETE"},
		{http.MethodGet, "/runs/dual", "POST"},
		{http.MethodGet, "/runs/single/start", "POST"},
		{http.MethodGet, "/runs/x/continue", "POST"},
		{http.MethodPost, "/runs/x/diffs", "GET"},
		{http.MethodPost, "/runs/x", "GET"},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(c.method, c.path, nil))
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected 405, got %d", w.Code)
			}
			if got := w.Header().Get("Allow"); got != c.allow {
				t.Fatalf("expected Allow %q, got %q", c.allow, got)
			}
			assertJSONError(t, w)
		})
	}
}

func TestNotFoundUsesJSONError(t *testing.T) {
	h := newTestHandler(t)
	cases := []struct{ path, message string }{
		{"/", "not found"},
		{"/nope", "not found"},
		{"/suites/", "not found"},
		{"/suites/x/y", "not found"},
		{"/steps/", "not found"},
		{"/runs/", "not found"},
		{"/suites/missing", "suite not found"},
		{"/runs/missing", "run not found"},
		{"/runs/missing/diffs", "run not found"},
	}
	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, c.path, nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
			}
			if got := assertJSONError(t, w); got != c.message {
				t.Fatalf("expected message %q, got %q", c.message, got)
			}
		})
	}
}

func TestDocsRedirectStillWorks(t *testing.T) {
	h := newTestHandler(t)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if w.Code != http.StatusMovedPermanently {
		t.Fatalf("expected 301, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Location"); got != "/docs/" {
		t.Fatalf("expected redirect to /docs/, got %q", got)
	}

	// /docs/ must serve the UI itself: redirecting to /docs/index.html loops,
	// because http.FileServer redirects index.html back to the directory.
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (Location %q)", w.Code, w.Header().Get("Location"))
	}
	if !strings.Contains(w.Body.String(), "swagger") {
		t.Fatalf("expected Swagger UI page, got %q", w.Body.String())
	}
}

func TestListDiffsRunNotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/runs/nonexistent/diffs", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
