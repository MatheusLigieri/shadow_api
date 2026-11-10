package diff

import (
	"encoding/json"
	"testing"
)

func TestCompareBodies_identical(t *testing.T) {
	body := `{"name":"alice","age":30}`
	r := CompareBodies(body, body, nil)
	if !r.Passed {
		t.Fatalf("identical bodies should pass, got changes: %s", r.DiffJSON)
	}
	if r.Summary != "no changes" {
		t.Fatalf("expected 'no changes', got %q", r.Summary)
	}
}

func TestCompareBodies_missingKey(t *testing.T) {
	a := `{"name":"alice","age":30}`
	b := `{"name":"alice"}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("missing key should not pass")
	}
	var changes []Change
	if err := json.Unmarshal([]byte(r.DiffJSON), &changes); err != nil {
		t.Fatalf("parse diff_json: %v", err)
	}
	if len(changes) == 0 {
		t.Fatal("expected at least 1 change")
	}
}

func TestCompareBodies_addedKey(t *testing.T) {
	a := `{"name":"alice"}`
	b := `{"name":"alice","extra":"new"}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("added key should not pass")
	}
}

func TestCompareBodies_valueChanged(t *testing.T) {
	a := `{"name":"alice"}`
	b := `{"name":"bob"}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("value change should not pass")
	}
}

func TestCompareBodies_ignorePath(t *testing.T) {
	a := `{"name":"alice","ts":"2024-01-01"}`
	b := `{"name":"alice","ts":"2025-01-01"}`
	r := CompareBodies(a, b, []string{"$.ts"})
	if !r.Passed {
		t.Fatalf("ignored path should not cause failure, got: %s", r.DiffJSON)
	}
}

func TestCompareBodies_boolVsString(t *testing.T) {
	// This was broken with fmt.Sprintf: bool `true` == string `"true"`
	a := `{"active":true}`
	b := `{"active":"true"}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("bool true vs string 'true' should differ (valuesEqual was broken before fix)")
	}
}

func TestCompareBodies_arrayLengthDiff(t *testing.T) {
	a := `{"items":[1,2,3]}`
	b := `{"items":[1,2]}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("array length change should not pass")
	}
}

func TestCompareBodies_arrayElementChanged(t *testing.T) {
	a := `{"items":[1,2,3]}`
	b := `{"items":[1,9,3]}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("array element change should not pass")
	}
}

func TestCompareBodies_plainTextIdentical(t *testing.T) {
	body := "not json at all"
	r := CompareBodies(body, body, nil)
	if !r.Passed {
		t.Fatal("identical non-JSON bodies should pass")
	}
}

func TestCompareBodies_plainTextDiffer(t *testing.T) {
	r := CompareBodies("hello", "world", nil)
	if r.Passed {
		t.Fatal("different non-JSON bodies should not pass")
	}
}

func TestCompareBodies_nestedObject(t *testing.T) {
	a := `{"user":{"name":"alice","role":"admin"}}`
	b := `{"user":{"name":"alice","role":"user"}}`
	r := CompareBodies(a, b, nil)
	if r.Passed {
		t.Fatal("nested role change should not pass")
	}
	var changes []Change
	json.Unmarshal([]byte(r.DiffJSON), &changes) //nolint:errcheck
	found := false
	for _, c := range changes {
		if c.Path == "$.user.role" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected change at $.user.role, got: %s", r.DiffJSON)
	}
}

func TestHashBody_deterministic(t *testing.T) {
	h1 := HashBody("hello world")
	h2 := HashBody("hello world")
	if h1 != h2 {
		t.Fatal("hash should be deterministic")
	}
	if HashBody("hello") == HashBody("world") {
		t.Fatal("different inputs should have different hashes")
	}
}
