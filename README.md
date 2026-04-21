# opengaebi-bridge

AI 에이전트 간 메시지 브로커 & 레지스트리. SQLite(로컬) 또는 PostgreSQL(프로덕션)을 백엔드로 사용하는 경량 서버입니다. REST API, MCP(JSON-RPC 2.0), A2A(Google 표준) 세 가지 프로토콜을 지원합니다.

## 빠른 시작

### 바이너리 빌드 & 실행

```bash
go build -o bridge ./cmd/bridge
./bridge
# → 서버 시작: :7777 (SQLite, ./opengaebi.db)
```

### Docker Compose

```bash
docker compose up           # SQLite 모드 (기본)
docker compose --profile postgres up  # PostgreSQL 모드
```

## 환경 변수

| 변수 | 기본값 | 설명 |
|------|--------|------|
| `BRIDGE_PORT` | `7777` | HTTP 리슨 포트 |
| `BRIDGE_DB` | `sqlite` | 백엔드 DB (`sqlite` 또는 `postgres`) |
| `DATABASE_URL` | `./opengaebi.db` | SQLite 파일 경로 또는 PostgreSQL DSN |
| `BRIDGE_API_KEY` | *(자동 생성)* | REST API 인증 키. 미설정 시 서버가 랜덤 생성 후 stdout 출력 |
| `BRIDGE_BASE_URL` | `http://localhost:{PORT}` | A2A agent card의 base URL |
| `REGISTRY_URL` | *(없음)* | 클라우드 레지스트리 URL (옵션) |

## REST API

모든 `/v1/*` 엔드포인트는 `Authorization: Bearer <BRIDGE_API_KEY>` 헤더 필요.

### 에이전트

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `POST` | `/v1/agents` | 에이전트 등록 |
| `GET` | `/v1/agents?workspace=&tags=` | 에이전트 목록 (태그 AND 필터 지원) |
| `DELETE` | `/v1/agents/{id}` | 에이전트 삭제 |

**POST /v1/agents 요청 예시**
```json
{
  "workspace": "gowit",
  "name": "ai-framework",
  "kind": "session",
  "tags": ["role:worker", "lang:go"],
  "ip": "127.0.0.1",
  "port": 8001
}
```

**GET /v1/agents 태그 필터 예시**
```
GET /v1/agents?workspace=gowit&tags=role:worker,lang:go
```
쉼표 구분, AND 조건. 태그 없으면 전체 반환.

### 메시지

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `POST` | `/v1/messages` | 메시지 전송 (to 미지정 시 브로드캐스트) |
| `GET` | `/v1/messages?workspace=&to=&limit=` | 메시지 폴링 (직접 수신 + 브로드캐스트) |
| `DELETE` | `/v1/messages/{id}` | 메시지 삭제 |

**브로드캐스트**: `to` 필드를 비우거나 생략하면 워크스페이스 전체 에이전트가 수신합니다.

### 아티팩트

| 메서드 | 경로 | 설명 |
|--------|------|------|
| `POST` | `/v1/artifacts` | 아티팩트 저장 |
| `GET` | `/v1/artifacts/{id}` | 아티팩트 조회 |

## MCP (Model Context Protocol)

`POST /mcp` — 인증 불필요. JSON-RPC 2.0.

### 도구 목록

| 도구 | 설명 |
|------|------|
| `ask_agent` | 특정 에이전트에 메시지 전송 |
| `find_agents` | 에이전트 조회 (태그 필터 지원) |
| `publish_change` | 아티팩트 저장 + 브로드캐스트 알림 옵션 |

**find_agents — 태그 필터**
```json
{
  "jsonrpc": "2.0", "id": 1, "method": "tools/call",
  "params": {
    "name": "find_agents",
    "arguments": { "workspace": "gowit", "tags": ["role:worker", "lang:go"] }
  }
}
```

**publish_change — 브로드캐스트**
```json
{
  "jsonrpc": "2.0", "id": 2, "method": "tools/call",
  "params": {
    "name": "publish_change",
    "arguments": {
      "workspace": "gowit", "from": "session-a",
      "name": "schema.json", "content": "{}", "broadcast": true
    }
  }
}
```
`broadcast: true` 시 아티팩트 저장 후 `to_peer=""` 브로드캐스트 메시지를 자동 전송합니다.

### Claude Code MCP 설정

```json
{
  "mcpServers": {
    "opengaebi": {
      "type": "http",
      "url": "http://localhost:7777/mcp"
    }
  }
}
```

## SessionStart Hook

Claude Code 세션 시작 시 브리지에 자동 등록하려면 `scripts/register-session.sh`를 SessionStart 훅으로 등록합니다.

```bash
# ~/.claude/settings.json 또는 프로젝트 .claude/settings.json
{
  "hooks": {
    "SessionStart": [
      {
        "matcher": "",
        "hooks": [
          { "type": "command", "command": "/path/to/opengaebi/scripts/register-session.sh" }
        ]
      }
    ]
  }
}
```

스크립트는 `BRIDGE_BASE_URL`, `BRIDGE_API_KEY`, `BRIDGE_WORKSPACE`, `BRIDGE_SESSION_NAME` 환경 변수를 읽습니다.

## A2A (Agent-to-Agent)

Google A2A 표준 지원. `GET /.well-known/agent.json` 으로 에이전트 카드 조회, `POST /a2a` 로 태스크 전송.

## 헬스체크

```bash
curl http://localhost:7777/health
# {"status":"ok"}
```
