package diff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type Change struct {
	Path   string `json:"path"`
	Before any    `json:"before"`
	After  any    `json:"after"`
}

type Result struct {
	Passed   bool
	DiffJSON string
	Summary  string
}

func CompareBodies(before string, after string, ignorePaths []string) Result {
	var a any
	var b any
	aErr := json.Unmarshal([]byte(before), &a)
	bErr := json.Unmarshal([]byte(after), &b)
	if aErr == nil && bErr == nil {
		applyIgnorePaths(a, ignorePaths)
		applyIgnorePaths(b, ignorePaths)
		var changes []Change
		diffJSON(a, b, "$", &changes)
		passed := len(changes) == 0
		payload, _ := json.Marshal(changes)
		summary := "no changes"
		if !passed {
			summary = fmt.Sprintf("%d change(s)", len(changes))
		}
		return Result{Passed: passed, DiffJSON: string(payload), Summary: summary}
	}

	if before == after {
		return Result{Passed: true, DiffJSON: "[]", Summary: "no changes"}
	}

	payload, _ := json.Marshal([]Change{{Path: "$.body", Before: before, After: after}})
	return Result{Passed: false, DiffJSON: string(payload), Summary: "body changed"}
}

func HashBody(body string) string {
	sha := sha256.Sum256([]byte(body))
	return hex.EncodeToString(sha[:])
}

func applyIgnorePaths(obj any, paths []string) {
	for _, path := range paths {
		trim := strings.TrimSpace(path)
		if trim == "" {
			continue
		}
		trim = strings.TrimPrefix(trim, "$.")
		parts := strings.Split(trim, ".")
		removePath(obj, parts)
	}
}

func removePath(obj any, parts []string) {
	if len(parts) == 0 {
		return
	}

	m, ok := obj.(map[string]any)
	if !ok {
		return
	}

	if len(parts) == 1 {
		delete(m, parts[0])
		return
	}

	next, ok := m[parts[0]]
	if !ok {
		return
	}
	removePath(next, parts[1:])
}

func diffJSON(a any, b any, path string, changes *[]Change) {
	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok {
			*changes = append(*changes, Change{Path: path, Before: a, After: b})
			return
		}
		seen := map[string]struct{}{}
		for k := range va {
			seen[k] = struct{}{}
		}
		for k := range vb {
			seen[k] = struct{}{}
		}
		for k := range seen {
			np := path + "." + k
			diffJSON(va[k], vb[k], np, changes)
		}
	case []any:
		vb, ok := b.([]any)
		if !ok {
			*changes = append(*changes, Change{Path: path, Before: a, After: b})
			return
		}
		if len(va) != len(vb) {
			*changes = append(*changes, Change{Path: path + ".length", Before: len(va), After: len(vb)})
		}
		minLen := len(va)
		if len(vb) < minLen {
			minLen = len(vb)
		}
		for i := 0; i < minLen; i++ {
			np := fmt.Sprintf("%s[%d]", path, i)
			diffJSON(va[i], vb[i], np, changes)
		}
	default:
		if !valuesEqual(a, b) {
			*changes = append(*changes, Change{Path: path, Before: a, After: b})
		}
	}
}

func valuesEqual(a any, b any) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return string(aj) == string(bj)
}
