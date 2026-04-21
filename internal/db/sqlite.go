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

func (s *SQLiteDB) Close() error {
	return s.db.Close()
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
