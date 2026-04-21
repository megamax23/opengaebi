package mcp_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opengaebi/opengaebi/internal/db"
	"github.com/opengaebi/opengaebi/internal/mcp"
)

func newMCPServer(t *testing.T) *mcp.Server {
	t.Helper()
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return mcp.New(store)
}

func mcpRPC(t *testing.T, srv *mcp.Server, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("mcp rpc: expected 200, got %d — %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	json.NewDecoder(rec.Body).Decode(&resp)
	return resp
}

func TestMCP_Initialize(t *testing.T) {
	srv := newMCPServer(t)

	// 2024-11-05 버전 요청 → 에코 확인
	resp := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}`)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in response: %+v", resp)
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected echo of 2024-11-05, got: %v", result["protocolVersion"])
	}

	// 2025-03-26 버전 요청 → 에코 확인 (Claude Code 최신 버전 대응)
	resp2 := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":2,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"claude-code","version":"2.0"}}}`)
	result2, ok := resp2["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in response2: %+v", resp2)
	}
	if result2["protocolVersion"] != "2025-03-26" {
		t.Errorf("expected echo of 2025-03-26, got: %v", result2["protocolVersion"])
	}
}

func TestMCP_ToolsList(t *testing.T) {
	srv := newMCPServer(t)
	resp := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %+v", resp)
	}
	tools, ok := result["tools"].([]any)
	if !ok || len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %+v", result["tools"])
	}
}

func TestMCP_FindAgents(t *testing.T) {
	srv := newMCPServer(t)
	resp := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"find_agents","arguments":{"workspace":"mcp-ws"}}}`)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %+v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content array: %+v", result)
	}
}

func TestMCP_AskAgent(t *testing.T) {
	srv := newMCPServer(t)
	resp := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"ask_agent","arguments":{"workspace":"mcp-ws","to":"bot-x","from":"session-y","payload":"hello"}}}`)

	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %+v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content: %+v", result)
	}
}
