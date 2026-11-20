package util

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSendSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendSuccess(c, http.StatusOK, "Hotel retrieved successfully", gin.H{"id": "h-1", "name": "Goa Sands"})

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if !resp.Success {
		t.Error("expected resp.Success to be true")
	}
	if resp.Message != "Hotel retrieved successfully" {
		t.Errorf("unexpected message: %s", resp.Message)
	}
}

func TestSendError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	SendError(c, http.StatusNotFound, "Room not found")

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var resp APIError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("expected resp.Success to be false")
	}
	if resp.Error != "Room not found" {
		t.Errorf("unexpected error message: %s", resp.Error)
	}
}

func TestSendValidationError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	details := gin.H{"field": "checkIn", "reason": "date must be in the future"}
	SendValidationError(c, "Invalid booking dates", details)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, w.Code)
	}

	var resp APIError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("expected resp.Success to be false")
	}
	if resp.Error != "Invalid booking dates" {
		t.Errorf("unexpected error message: %s", resp.Error)
	}
}

type dummyPayload struct {
	Name  string `json:"name" binding:"required"`
	Email string `json:"email" binding:"required,email"`
}

func TestSendBindingError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Simulate binding validation error
	req, _ := http.NewRequest("POST", "/", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	var dummy dummyPayload
	err := c.ShouldBindJSON(&dummy)
	if err == nil {
		t.Fatal("expected validation error")
	}

	SendBindingError(c, err)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var resp APIError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Success {
		t.Error("expected resp.Success to be false")
	}
	if !strings.Contains(resp.Error, "Name is required") {
		t.Errorf("expected 'Name is required' in error, got: %s", resp.Error)
	}
	if resp.Message != resp.Error {
		t.Errorf("expected message to match error, got: %s", resp.Message)
	}
}
