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

func TestMCP_FindAgents_ByTags(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	srv := mcp.New(store)

	ctx := t.Context()
	peers := []db.Peer{
		{ID: "p1", Workspace: "ws", Name: "worker-go", Kind: "agent", Tags: []string{"role:worker", "lang:go"}},
		{ID: "p2", Workspace: "ws", Name: "worker-py", Kind: "agent", Tags: []string{"role:worker", "lang:python"}},
		{ID: "p3", Workspace: "ws", Name: "coord", Kind: "agent", Tags: []string{"role:coordinator"}},
	}
	for _, p := range peers {
		store.RegisterPeer(ctx, p)
	}

	// role:worker → 2 results
	resp := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"find_agents","arguments":{"workspace":"ws","tags":["role:worker"]}}}`)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %+v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content: %+v", result)
	}
	text := content[0].(map[string]any)["text"].(string)
	if !contains(text, "2") {
		t.Errorf("expected 2 agents in text, got: %s", text)
	}

	// role:worker + lang:go → 1 result
	resp2 := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"find_agents","arguments":{"workspace":"ws","tags":["role:worker","lang:go"]}}}`)
	result2, ok := resp2["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result2: %+v", resp2)
	}
	content2 := result2["content"].([]any)
	text2 := content2[0].(map[string]any)["text"].(string)
	if !contains(text2, "worker-go") {
		t.Errorf("expected worker-go in text, got: %s", text2)
	}
}

func TestMCP_PublishChange_Broadcast(t *testing.T) {
	store, err := db.NewSQLite(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	srv := mcp.New(store)

	resp := mcpRPC(t, srv, `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"publish_change","arguments":{"workspace":"ws","from":"session-a","name":"schema.json","content":"{}","broadcast":true}}}`)
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result: %+v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("expected content: %+v", result)
	}
	text := content[0].(map[string]any)["text"].(string)
	if !contains(text, "브로드캐스트") {
		t.Errorf("expected broadcast confirmation in text, got: %s", text)
	}

	// broadcast message must be pollable by any agent
	ctx := t.Context()
	msgs, err := store.PollMessages(ctx, "ws", "any-agent", 10)
	if err != nil {
		t.Fatalf("PollMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 broadcast message, got %d", len(msgs))
	}
	if msgs[0].ToPeer != "" {
		t.Errorf("expected empty ToPeer for broadcast, got: %s", msgs[0].ToPeer)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
