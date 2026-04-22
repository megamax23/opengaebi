package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMessages_SendAndPoll(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := `{"from":"agent-a","to":"agent-b","workspace":"msgws","payload":"hello from a"}`
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("send: expected 201, got %d — %s", rec.Code, rec.Body.String())
	}

	var sendResp struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rec.Body).Decode(&sendResp)
	if sendResp.ID == "" {
		t.Fatal("expected non-empty message id")
	}

	req2 := httptest.NewRequest("GET", "/v1/messages?workspace=msgws&to=agent-b&limit=10", nil)
	req2.Header.Set("Authorization", "Bearer test-api-key")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("poll: expected 200, got %d", rec2.Code)
	}
	var pollResp struct {
		Messages []struct {
			Payload string `json:"Payload"`
		} `json:"messages"`
	}
	json.NewDecoder(rec2.Body).Decode(&pollResp)
	if len(pollResp.Messages) != 1 || pollResp.Messages[0].Payload != "hello from a" {
		t.Errorf("unexpected poll response: %+v", pollResp)
	}
}

func TestMessages_Delete(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := `{"from":"x","to":"y","workspace":"delws","payload":"bye"}`
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var r struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rec.Body).Decode(&r)

	delReq := httptest.NewRequest("DELETE", fmt.Sprintf("/v1/messages/%s", r.ID), nil)
	delReq.Header.Set("Authorization", "Bearer test-api-key")
	delRec := httptest.NewRecorder()
	h.ServeHTTP(delRec, delReq)

	if delRec.Code != http.StatusNoContent {
		t.Errorf("delete: expected 204, got %d", delRec.Code)
	}
}

func TestMessages_PayloadTooLarge(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	bigPayload := strings.Repeat("x", 64*1024+1)
	body := fmt.Sprintf(`{"from":"a","to":"b","workspace":"ws","payload":%q}`, bigPayload)
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d", rec.Code)
	}
}

func TestMessages_Broadcast(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := `{"workspace":"bcastws","from":"orchestrator","to":"","payload":"모든 에이전트에게"}`
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d — %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest("GET", "/v1/messages?workspace=bcastws&to=bot-a", nil)
	req2.Header.Set("Authorization", "Bearer test-api-key")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	var resp struct {
		Messages []map[string]any `json:"messages"`
	}
	json.NewDecoder(rec2.Body).Decode(&resp)
	if len(resp.Messages) != 1 {
		t.Errorf("expected 1 broadcast message, got %d", len(resp.Messages))
	}
}
