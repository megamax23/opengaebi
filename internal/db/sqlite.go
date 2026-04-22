package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteDB struct {
	db *sql.DB
}

func NewSQLite(path string) (*SQLiteDB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	s := &SQLiteDB{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *SQLiteDB) migrate() error {
	// WAL 모드: 동시 읽기/쓰기 성능 향상. :memory: DB에서는 무시됨.
	s.db.Exec(`PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000`)
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS peers (
			id        TEXT PRIMARY KEY,
			workspace TEXT,
			name      TEXT NOT NULL,
			kind      TEXT DEFAULT 'session',
			tags      TEXT DEFAULT '[]',
			ip        TEXT,
			port      INTEGER,
			last_seen DATETIME
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_peers_workspace_name ON peers(workspace, name);

		CREATE TABLE IF NOT EXISTS messages (
			id         TEXT PRIMARY KEY,
			from_peer  TEXT NOT NULL,
			to_peer    TEXT NOT NULL,
			workspace  TEXT NOT NULL,
			payload    TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(workspace, to_peer, created_at);

		CREATE TABLE IF NOT EXISTS artifacts (
			id         TEXT PRIMARY KEY,
			workspace  TEXT NOT NULL,
			name       TEXT NOT NULL,
			kind       TEXT NOT NULL DEFAULT 'text',
			content    BLOB,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	return err
}

func (s *SQLiteDB) RegisterPeer(ctx context.Context, peer Peer) error {
	tags, _ := json.Marshal(peer.Tags)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO peers (id, workspace, name, kind, tags, ip, port, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(workspace, name) DO UPDATE SET
			id=excluded.id, kind=excluded.kind, tags=excluded.tags,
			ip=excluded.ip, port=excluded.port, last_seen=excluded.last_seen
	`, peer.ID, peer.Workspace, peer.Name, peer.Kind, string(tags), peer.IP, peer.Port, time.Now())
	return err
}

func (s *SQLiteDB) GetPeer(ctx context.Context, workspace, name string) (*Peer, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, workspace, name, kind, tags, ip, port, last_seen FROM peers WHERE workspace=? AND name=?`,
		workspace, name)
	return scanPeer(row)
}

func (s *SQLiteDB) ListPeers(ctx context.Context, workspace string) ([]Peer, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, workspace, name, kind, tags, ip, port, last_seen FROM peers WHERE workspace=?`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []Peer
	for rows.Next() {
		p, err := scanPeer(rows)
		if err != nil {
			return nil, err
		}
		peers = append(peers, *p)
	}
	return peers, rows.Err()
}

func (s *SQLiteDB) DeletePeer(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM peers WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("peer not found: %s", id)
	}
	return nil
}

func (s *SQLiteDB) ListPeersByTags(ctx context.Context, workspace string, tags []string) ([]Peer, error) {
	peers, err := s.ListPeers(ctx, workspace)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return peers, nil
	}
	var filtered []Peer
	for _, p := range peers {
		if hasAllTags(p.Tags, tags) {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}

func (s *SQLiteDB) Close() error {
	return s.db.Close()
}

func (s *SQLiteDB) SendMessage(ctx context.Context, msg Message) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO messages (id, from_peer, to_peer, workspace, payload) VALUES (?,?,?,?,?)`,
		msg.ID, msg.FromPeer, msg.ToPeer, msg.Workspace, msg.Payload)
	return err
}

func (s *SQLiteDB) PollMessages(ctx context.Context, workspace, toPeer string, limit int) ([]Message, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, from_peer, to_peer, workspace, payload, created_at
		 FROM messages WHERE workspace=? AND (to_peer=? OR to_peer='') ORDER BY created_at ASC LIMIT ?`,
		workspace, toPeer, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		var createdAt sql.NullTime
		if err := rows.Scan(&m.ID, &m.FromPeer, &m.ToPeer, &m.Workspace, &m.Payload, &createdAt); err != nil {
			return nil, err
		}
		if createdAt.Valid {
			m.CreatedAt = createdAt.Time
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *SQLiteDB) DeleteMessage(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM messages WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("message not found: %s", id)
	}
	return nil
}

func (s *SQLiteDB) SaveArtifact(ctx context.Context, a Artifact) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts (id, workspace, name, kind, content) VALUES (?,?,?,?,?)`,
		a.ID, a.Workspace, a.Name, a.Kind, a.Content)
	return err
}

func (s *SQLiteDB) GetArtifact(ctx context.Context, id string) (*Artifact, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, workspace, name, kind, content, created_at FROM artifacts WHERE id=?`, id)
	var a Artifact
	var createdAt sql.NullTime
	if err := row.Scan(&a.ID, &a.Workspace, &a.Name, &a.Kind, &a.Content, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("artifact not found: %s", id)
		}
		return nil, err
	}
	if createdAt.Valid {
		a.CreatedAt = createdAt.Time
	}
	return &a, nil
}

func hasAllTags(peerTags, required []string) bool {
	set := make(map[string]struct{}, len(peerTags))
	for _, t := range peerTags {
		set[t] = struct{}{}
	}
	for _, t := range required {
		if _, ok := set[t]; !ok {
			return false
		}
	}
	return true
}

type scanner interface {
	Scan(dest ...any) error
}

func scanPeer(s scanner) (*Peer, error) {
	var p Peer
	var tagsJSON string
	var lastSeen sql.NullTime
	err := s.Scan(&p.ID, &p.Workspace, &p.Name, &p.Kind, &tagsJSON, &p.IP, &p.Port, &lastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("peer not found")
		}
		return nil, err
	}
	json.Unmarshal([]byte(tagsJSON), &p.Tags)
	if lastSeen.Valid {
		p.LastSeen = lastSeen.Time
	}
	return &p, nil
}
