package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHumanReadable_ErrorResponses(t *testing.T) {
	setupInventoryTestDB(t)

	// 1. Missing required fields in CreateRoom (empty JSON object)
	emptyObjBody := bytes.NewBufferString("{}")
	req, _ := http.NewRequest(http.MethodPost, "/api/admin/rooms", emptyObjBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+hotelAJWT)
	w := httptest.NewRecorder()
	testRouter.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d", w.Code)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	// Verify dual error and message fields
	errMsg, okErr := res["error"].(string)
	msgVal, okMsg := res["message"].(string)
	if !okErr || !okMsg {
		t.Fatalf("expected both 'error' and 'message' string fields in response, got %+v", res)
	}
	if errMsg != msgVal {
		t.Errorf("expected 'error' and 'message' to match, got error=%q, message=%q", errMsg, msgVal)
	}

	// Verify no internal Go/Gin reflection or struct leak
	forbiddenSubstrings := []string{
		"Key:",
		"Field validation",
		"CreateRoomRequest",
		"failed on the 'required' tag",
		"Error:Field",
	}
	for _, sub := range forbiddenSubstrings {
		if strings.Contains(errMsg, sub) {
			t.Errorf("error message %q leaked internal validator structure (%s)", errMsg, sub)
		}
	}

	// Verify clear human readability
	if !strings.Contains(errMsg, "is required") {
		t.Errorf("expected human-readable 'is required' error, got %q", errMsg)
	}

	// 2. Empty request body
	emptyBody := bytes.NewBufferString("")
	reqEmpty, _ := http.NewRequest(http.MethodPost, "/api/admin/rooms", emptyBody)
	reqEmpty.Header.Set("Content-Type", "application/json")
	reqEmpty.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wEmpty := httptest.NewRecorder()
	testRouter.ServeHTTP(wEmpty, reqEmpty)

	if wEmpty.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty body, got %d", wEmpty.Code)
	}

	var resEmpty map[string]interface{}
	json.Unmarshal(wEmpty.Body.Bytes(), &resEmpty)
	if resEmpty["error"] != "Request body cannot be empty" {
		t.Errorf("expected 'Request body cannot be empty', got %v", resEmpty["error"])
	}

	// 3. Malformed JSON syntax
	malformedBody := bytes.NewBufferString("{invalid_json:")
	reqMalformed, _ := http.NewRequest(http.MethodPost, "/api/admin/rooms", malformedBody)
	reqMalformed.Header.Set("Content-Type", "application/json")
	reqMalformed.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wMalformed := httptest.NewRecorder()
	testRouter.ServeHTTP(wMalformed, reqMalformed)

	if wMalformed.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed json, got %d", wMalformed.Code)
	}

	var resMalformed map[string]interface{}
	json.Unmarshal(wMalformed.Body.Bytes(), &resMalformed)
	if resMalformed["error"] != "Request body contains invalid or malformed JSON" {
		t.Errorf("expected 'Request body contains invalid or malformed JSON', got %v", resMalformed["error"])
	}

	// 4. Missing Category Name
	catPayload := map[string]interface{}{
		"description":  "Suite without a name",
		"defaultRate":  3500,
		"maxOccupancy": 2,
	}
	catBody, _ := json.Marshal(catPayload)
	reqCat, _ := http.NewRequest(http.MethodPost, "/api/admin/room-categories", bytes.NewBuffer(catBody))
	reqCat.Header.Set("Content-Type", "application/json")
	reqCat.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wCat := httptest.NewRecorder()
	testRouter.ServeHTTP(wCat, reqCat)

	if wCat.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for category missing name, got %d", wCat.Code)
	}

	var resCat map[string]interface{}
	json.Unmarshal(wCat.Body.Bytes(), &resCat)
	if resCat["error"] != "Name is required" {
		t.Errorf("expected 'Name is required', got %v", resCat["error"])
	}
	if resCat["message"] != "Name is required" {
		t.Errorf("expected message to match 'Name is required', got %v", resCat["message"])
	}

	// 5. Category with negative rate
	catNegPayload := map[string]interface{}{
		"name":         "Economy",
		"defaultRate":  -100,
		"maxOccupancy": 2,
	}
	catNegBody, _ := json.Marshal(catNegPayload)
	reqCatNeg, _ := http.NewRequest(http.MethodPost, "/api/admin/room-categories", bytes.NewBuffer(catNegBody))
	reqCatNeg.Header.Set("Content-Type", "application/json")
	reqCatNeg.Header.Set("Authorization", "Bearer "+hotelAJWT)
	wCatNeg := httptest.NewRecorder()
	testRouter.ServeHTTP(wCatNeg, reqCatNeg)

	if wCatNeg.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for category with negative rate, got %d", wCatNeg.Code)
	}

	var resCatNeg map[string]interface{}
	json.Unmarshal(wCatNeg.Body.Bytes(), &resCatNeg)
	if resCatNeg["error"] != "A valid positive rate is required" {
		t.Errorf("expected 'A valid positive rate is required', got %v", resCatNeg["error"])
	}
}
