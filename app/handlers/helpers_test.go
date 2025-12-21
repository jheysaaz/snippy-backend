package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	respondError(c, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	if !contains(w.Body.String(), "test error") {
		t.Errorf("Expected body to contain 'test error', got %s", w.Body.String())
	}
}

func TestRespondSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := map[string]string{"message": "success"}
	respondSuccess(c, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !contains(w.Body.String(), "success") {
		t.Errorf("Expected body to contain 'success', got %s", w.Body.String())
	}
}

func TestRespondWithCount(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	items := []string{"item1", "item2", "item3"}
	respondWithCount(c, items, len(items))

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	if !contains(w.Body.String(), "items") || !contains(w.Body.String(), "count") {
		t.Errorf("Expected body to contain 'items' and 'count', got %s", w.Body.String())
	}
}

func TestHandleScanError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedStatus int
		shouldHandle   bool
	}{
		{"no rows error", sql.ErrNoRows, http.StatusNotFound, true},
		{"generic error", errors.New("db error"), http.StatusInternalServerError, true},
		{"no error", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			handled := handleScanError(c, tt.err, "Resource not found")

			if handled != tt.shouldHandle {
				t.Errorf("Expected handled=%v, got %v", tt.shouldHandle, handled)
			}

			if tt.shouldHandle && w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestCheckOwnership(t *testing.T) {
	tests := []struct {
		name           string
		resourceUserID string
		authUserID     string
		expected       bool
	}{
		{"matching IDs", "user123", "user123", true},
		{"different IDs", "user123", "user456", false},
		{"empty resource ID", "", "user123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			result := checkOwnership(c, tt.resourceUserID, tt.authUserID, "resource")

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}

			if !tt.expected && w.Code != http.StatusForbidden {
				t.Errorf("Expected status %d when ownership fails, got %d", http.StatusForbidden, w.Code)
			}
		})
	}
}

func TestValueOrNilString(t *testing.T) {
	tests := []struct {
		name     string
		input    *string
		expected interface{}
	}{
		{"nil pointer", nil, nil},
		{"empty string", strPtr(""), ""},
		{"non-empty string", strPtr("test"), "test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valueOrNilString(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestArrayOrNilStringSlice(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		isNil bool
	}{
		{"nil slice", nil, true},
		{"empty slice", []string{}, false},
		{"non-empty slice", []string{"a", "b"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := arrayOrNilStringSlice(tt.input)
			if tt.isNil && result != nil {
				t.Errorf("Expected nil, got %v", result)
			}
			if !tt.isNil && result == nil {
				t.Error("Expected non-nil result")
			}
		})
	}
}

func TestHandleUserUniqueViolation(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		shouldHandle bool
		skipNil      bool
	}{
		{"duplicate username", errors.New("duplicate key value violates unique constraint \"users_username_key\""), true, false},
		{"duplicate email", errors.New("duplicate key value violates unique constraint \"users_email_key\""), true, false},
		{"generic error", errors.New("generic database error"), false, false},
		{"no error", nil, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipNil && tt.err == nil {
				t.Skip("Skipping nil error test")
				return
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			handled := handleUserUniqueViolation(c, tt.err)

			if handled != tt.shouldHandle {
				t.Errorf("Expected handled=%v, got %v", tt.shouldHandle, handled)
			}

			if tt.shouldHandle && w.Code != http.StatusConflict {
				t.Errorf("Expected status %d, got %d", http.StatusConflict, w.Code)
			}
		})
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
