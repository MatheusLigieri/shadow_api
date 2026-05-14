package validation

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com", false},
		{"with path", "https://api.example.com/v1/users", false},
		{"empty string", "", true},
		{"whitespace only", "   ", true},
		{"invalid format", "not a url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateHTTPMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{"get", "GET", false},
		{"lowercase get", "get", false},
		{"post", "POST", false},
		{"put", "PUT", false},
		{"delete", "DELETE", false},
		{"patch", "PATCH", false},
		{"invalid", "INVALID", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPMethod(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHTTPMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSONString(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{"valid object", `{"key":"value"}`, false},
		{"valid array", `[1,2,3]`, false},
		{"valid string", `"hello"`, false},
		{"valid number", `123`, false},
		{"empty string", "", false},
		{"whitespace only", "   ", false},
		{"invalid json", `{invalid}`, true},
		{"malformed", `{"key":}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSONString(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSONString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequiredString(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		fieldName string
		wantErr   bool
	}{
		{"valid", "value", "field", false},
		{"empty", "", "field", true},
		{"whitespace", "   ", "field", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequiredString(tt.value, tt.fieldName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequiredString() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
