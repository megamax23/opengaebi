package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/opengaebi/opengaebi/internal/db"
)

type Server struct {
	db db.DB
}

func New(store db.DB) *Server {
	return &Server{db: store}
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeRPC(w, nil, nil, map[string]any{"code": -32700, "message": "parse error"})
		return
	}

	var result any
	var rpcErr any

	switch req.Method {
	case "initialize":
		// 클라이언트가 요청한 버전을 에코 — 버전 협상 호환성 보장 (Claude Code, Cursor 등)
		var initParams struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		json.Unmarshal(req.Params, &initParams)
		version := initParams.ProtocolVersion
		if version == "" {
			version = "2024-11-05"
		}
		result = map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "opengaebi-bridge", "version": "0.1.0"},
		}
	case "tools/list":
		result = map[string]any{"tools": toolsList()}
	case "tools/call":
		result, rpcErr = s.toolsCall(r, req.Params)
	default:
		rpcErr = map[string]any{"code": -32601, "message": "method not found"}
	}

	writeRPC(w, req.ID, result, rpcErr)
}

func writeRPC(w http.ResponseWriter, id, result, rpcErr any) {
	resp := rpcResponse{JSONRPC: "2.0", ID: id}
	if rpcErr != nil {
		resp.Error = rpcErr
	} else {
		resp.Result = result
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func toolsList() []map[string]any {
	return []map[string]any{
		{
			"name":        "ask_agent",
			"description": "에이전트에 메시지를 보내고 메시지 ID를 반환한다",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace": map[string]any{"type": "string", "description": "워크스페이스"},
					"to":        map[string]any{"type": "string", "description": "수신 에이전트 이름"},
					"from":      map[string]any{"type": "string", "description": "발신 에이전트 이름"},
					"payload":   map[string]any{"type": "string", "description": "메시지 내용"},
				},
				"required": []string{"workspace", "to", "payload"},
			},
		},
		{
			"name":        "find_agents",
			"description": "워크스페이스에 등록된 에이전트를 조회한다",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace": map[string]any{"type": "string", "description": "워크스페이스"},
					"kind":      map[string]any{"type": "string", "description": "에이전트 종류 (session|agent)"},
				},
				"required": []string{"workspace"},
			},
		},
		{
			"name":        "publish_change",
			"description": "아티팩트(텍스트/코드)를 워크스페이스에 공유한다",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"workspace": map[string]any{"type": "string"},
					"from":      map[string]any{"type": "string"},
					"name":      map[string]any{"type": "string", "description": "파일명"},
					"kind":      map[string]any{"type": "string", "description": "text 또는 code"},
					"content":   map[string]any{"type": "string"},
				},
				"required": []string{"workspace", "name", "content"},
			},
		},
	}
}

func (s *Server) toolsCall(r *http.Request, params json.RawMessage) (any, any) {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, map[string]any{"code": -32602, "message": "invalid params"}
	}

	switch p.Name {
	case "ask_agent":
		return s.toolAskAgent(r, p.Arguments)
	case "find_agents":
		return s.toolFindAgents(r, p.Arguments)
	case "publish_change":
		return s.toolPublishChange(r, p.Arguments)
	default:
		return nil, map[string]any{"code": -32601, "message": "unknown tool: " + p.Name}
	}
}

func (s *Server) toolAskAgent(r *http.Request, args map[string]any) (any, any) {
	workspace, _ := args["workspace"].(string)
	to, _ := args["to"].(string)
	from, _ := args["from"].(string)
	payload, _ := args["payload"].(string)

	if workspace == "" || to == "" {
		return nil, map[string]any{"code": -32602, "message": "workspace and to are required"}
	}

	msg := db.Message{
		ID:        uuid.New().String(),
		FromPeer:  from,
		ToPeer:    to,
		Workspace: workspace,
		Payload:   payload,
	}
	if err := s.db.SendMessage(r.Context(), msg); err != nil {
		return nil, map[string]any{"code": -32603, "message": err.Error()}
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": fmt.Sprintf("메시지 전달 완료 (id=%s)", msg.ID)},
		},
	}, nil
}

func (s *Server) toolFindAgents(r *http.Request, args map[string]any) (any, any) {
	workspace, _ := args["workspace"].(string)
	if workspace == "" {
		return nil, map[string]any{"code": -32602, "message": "workspace is required"}
	}

	peers, err := s.db.ListPeers(r.Context(), workspace)
	if err != nil {
		return nil, map[string]any{"code": -32603, "message": err.Error()}
	}

	summary := fmt.Sprintf("에이전트 %d개 발견", len(peers))
	if len(peers) > 0 {
		names := make([]string, len(peers))
		for i, p := range peers {
			names[i] = fmt.Sprintf("%s/%s (%s)", p.Workspace, p.Name, p.Kind)
		}
		summary = fmt.Sprintf("%s: %v", summary, names)
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": summary},
		},
	}, nil
}

func (s *Server) toolPublishChange(r *http.Request, args map[string]any) (any, any) {
	workspace, _ := args["workspace"].(string)
	name, _ := args["name"].(string)
	content, _ := args["content"].(string)
	kind, _ := args["kind"].(string)

	if workspace == "" || name == "" {
		return nil, map[string]any{"code": -32602, "message": "workspace and name are required"}
	}
	if len(content) > 1<<20 {
		return nil, map[string]any{"code": -32602, "message": "content exceeds 1MB limit"}
	}
	if kind == "" {
		kind = "text"
	}

	art := db.Artifact{
		ID:        uuid.New().String(),
		Workspace: workspace,
		Name:      name,
		Kind:      kind,
		Content:   []byte(content),
	}
	if err := s.db.SaveArtifact(r.Context(), art); err != nil {
		return nil, map[string]any{"code": -32603, "message": err.Error()}
	}

	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": fmt.Sprintf("아티팩트 저장 완료 (id=%s, name=%s)", art.ID, art.Name)},
		},
	}, nil
}
