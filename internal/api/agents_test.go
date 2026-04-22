package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAgents_RegisterAndList(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := `{"workspace":"ws1","name":"bot-a","kind":"agent","tags":["role:worker"],"ip":"10.0.0.1","port":9000}`
	req := httptest.NewRequest("POST", "/v1/agents", bytes.NewBufferString(body))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d — %s", rec.Code, rec.Body.String())
	}

	req2 := httptest.NewRequest("GET", "/v1/agents?workspace=ws1", nil)
	req2.Header.Set("Authorization", "Bearer test-api-key")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rec2.Code)
	}

	var resp struct {
		Peers []struct {
			Name string `json:"Name"`
		} `json:"peers"`
	}
	json.NewDecoder(rec2.Body).Decode(&resp)
	if len(resp.Peers) != 1 || resp.Peers[0].Name != "bot-a" {
		t.Errorf("expected bot-a in list, got %+v", resp)
	}
}

func TestAgents_Delete(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	regBody := `{"workspace":"ws2","name":"temp","kind":"agent","ip":"127.0.0.1","port":0}`
	req := httptest.NewRequest("POST", "/v1/agents", bytes.NewBufferString(regBody))
	req.Header.Set("Authorization", "Bearer test-api-key")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var regResp struct {
		ID string `json:"id"`
	}
	json.NewDecoder(rec.Body).Decode(&regResp)

	delReq := httptest.NewRequest("DELETE", "/v1/agents/"+regResp.ID, nil)
	delReq.Header.Set("Authorization", "Bearer test-api-key")
	delRec := httptest.NewRecorder()
	h.ServeHTTP(delRec, delReq)

	if delRec.Code != http.StatusNoContent {
		t.Errorf("delete: expected 204, got %d", delRec.Code)
	}
}

func TestAgents_Unauthorized(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	req := httptest.NewRequest("GET", "/v1/agents?workspace=x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestAgents_ListByTags(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	for _, body := range []string{
		`{"workspace":"tagws","name":"bot-a","kind":"agent","tags":["role:worker","lang:go"]}`,
		`{"workspace":"tagws","name":"bot-b","kind":"agent","tags":["role:worker","lang:python"]}`,
		`{"workspace":"tagws","name":"bot-c","kind":"agent","tags":["role:coordinator"]}`,
	} {
		req := httptest.NewRequest("POST", "/v1/agents", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer test-api-key")
		req.Header.Set("Content-Type", "application/json")
		h.ServeHTTP(httptest.NewRecorder(), req)
	}

	req := httptest.NewRequest("GET", "/v1/agents?workspace=tagws&tags=role:worker", nil)
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Peers []map[string]any `json:"peers"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Peers) != 2 {
		t.Errorf("expected 2 peers with role:worker, got %d", len(resp.Peers))
	}

	req2 := httptest.NewRequest("GET", "/v1/agents?workspace=tagws&tags=role:worker,lang:go", nil)
	req2.Header.Set("Authorization", "Bearer test-api-key")
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	var resp2 struct {
		Peers []map[string]any `json:"peers"`
	}
	json.NewDecoder(rec2.Body).Decode(&resp2)
	if len(resp2.Peers) != 1 {
		t.Errorf("expected 1 peer with role:worker+lang:go, got %d", len(resp2.Peers))
	}
}
