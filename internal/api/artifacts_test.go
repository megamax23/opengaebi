package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestArtifacts_SaveAndGet(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := `{"workspace":"artws","name":"result.txt","kind":"text","content":"hello artifact"}`
	req := httptest.NewRequest("POST", "/v1/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("save: expected 201, got %d — %s", rec.Code, rec.Body.String())
	}
	var saveResp struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rec.Body).Decode(&saveResp)
	if saveResp.ID == "" {
		t.Fatal("expected non-empty artifact id")
	}

	req2 := httptest.NewRequest("GET", "/v1/artifacts/"+saveResp.ID, nil)
	req2.Header.Set("Authorization", "Bearer test-api-key")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", rec2.Code)
	}
	var getResp struct {
		Content string `json:"content"`
	}
	json.NewDecoder(rec2.Body).Decode(&getResp)
	if getResp.Content != "hello artifact" {
		t.Errorf("content mismatch: %s", getResp.Content)
	}
}

func TestArtifacts_TooLarge(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	bigContent := strings.Repeat("x", 1024*1024+1)
	body := `{"workspace":"artws","name":"big.txt","kind":"text","content":"` + bigContent + `"}`
	req := httptest.NewRequest("POST", "/v1/artifacts", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rec.Code)
	}
}
