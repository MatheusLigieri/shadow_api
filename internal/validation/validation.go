package validation

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

var (
	ErrInvalidURL      = errors.New("invalid URL format")
	ErrInvalidMethod   = errors.New("invalid HTTP method")
	ErrInvalidJSON     = errors.New("invalid JSON format")
	ErrEmptyRequired   = errors.New("required field is empty")
	ErrSuitNotFound    = errors.New("suite not found")
	ErrRunNotFound     = errors.New("run not found")
	ErrInvalidRunState = errors.New("run is not in a valid state")
)

func ValidateURL(urlStr string) error {
	if strings.TrimSpace(urlStr) == "" {
		return ErrEmptyRequired
	}
	u, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return ErrInvalidURL
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidURL
	}
	if strings.TrimSpace(u.Host) == "" {
		return ErrInvalidURL
	}
	return nil
}

func ValidateHTTPMethod(method string) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	validMethods := map[string]bool{
		"GET":     true,
		"POST":    true,
		"PUT":     true,
		"DELETE":  true,
		"PATCH":   true,
		"HEAD":    true,
		"OPTIONS": true,
	}
	if !validMethods[method] {
		return ErrInvalidMethod
	}
	return nil
}

func ValidateJSONString(jsonStr string) error {
	if strings.TrimSpace(jsonStr) == "" {
		return nil
	}
	var obj any
	if err := json.Unmarshal([]byte(jsonStr), &obj); err != nil {
		return ErrInvalidJSON
	}
	return nil
}

func ValidateRequiredString(value string, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New(fieldName + " is required")
	}
	return nil
}

func ValidateUUID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("id is required")
	}
	if len(id) != 36 {
		return errors.New("invalid uuid format")
	}
	return nil
}
