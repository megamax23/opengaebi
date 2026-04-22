package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, url string) (*PostgresDB, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	p := &PostgresDB{pool: pool}
	if err := p.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return p, nil
}

func (p *PostgresDB) migrate(ctx context.Context) error {
	_, err := p.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS peers (
			id        TEXT PRIMARY KEY,
			workspace TEXT,
			name      TEXT NOT NULL,
			kind      TEXT DEFAULT 'session',
			tags      JSONB DEFAULT '[]',
			ip        TEXT,
			port      INTEGER,
			last_seen TIMESTAMPTZ,
			UNIQUE(workspace, name)
		);

		CREATE TABLE IF NOT EXISTS messages (
			id         TEXT PRIMARY KEY,
			from_peer  TEXT NOT NULL,
			to_peer    TEXT NOT NULL,
			workspace  TEXT NOT NULL,
			payload    TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_messages_to ON messages(workspace, to_peer, created_at);

		CREATE TABLE IF NOT EXISTS artifacts (
			id         TEXT PRIMARY KEY,
			workspace  TEXT NOT NULL,
			name       TEXT NOT NULL,
			kind       TEXT NOT NULL DEFAULT 'text',
			content    BYTEA,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	return err
}

func (p *PostgresDB) RegisterPeer(ctx context.Context, peer Peer) error {
	tags, _ := json.Marshal(peer.Tags)
	_, err := p.pool.Exec(ctx, `
		INSERT INTO peers (id, workspace, name, kind, tags, ip, port, last_seen)
		VALUES ($1,$2,$3,$4,$5::jsonb,$6,$7,$8)
		ON CONFLICT (workspace, name) DO UPDATE SET
			id=EXCLUDED.id, kind=EXCLUDED.kind, tags=EXCLUDED.tags,
			ip=EXCLUDED.ip, port=EXCLUDED.port, last_seen=EXCLUDED.last_seen
	`, peer.ID, peer.Workspace, peer.Name, peer.Kind, string(tags), peer.IP, peer.Port, time.Now())
	return err
}

func (p *PostgresDB) GetPeer(ctx context.Context, workspace, name string) (*Peer, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT id, workspace, name, kind, tags, ip, port, last_seen FROM peers WHERE workspace=$1 AND name=$2`,
		workspace, name)
	var peer Peer
	var tagsJSON []byte
	err := row.Scan(&peer.ID, &peer.Workspace, &peer.Name, &peer.Kind, &tagsJSON, &peer.IP, &peer.Port, &peer.LastSeen)
	if err != nil {
		return nil, fmt.Errorf("peer not found: %w", err)
	}
	json.Unmarshal(tagsJSON, &peer.Tags)
	return &peer, nil
}

func (p *PostgresDB) ListPeers(ctx context.Context, workspace string) ([]Peer, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, workspace, name, kind, tags, ip, port, last_seen FROM peers WHERE workspace=$1`, workspace)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var peers []Peer
	for rows.Next() {
		var peer Peer
		var tagsJSON []byte
		if err := rows.Scan(&peer.ID, &peer.Workspace, &peer.Name, &peer.Kind, &tagsJSON, &peer.IP, &peer.Port, &peer.LastSeen); err != nil {
			return nil, err
		}
		json.Unmarshal(tagsJSON, &peer.Tags)
		peers = append(peers, peer)
	}
	return peers, rows.Err()
}

func (p *PostgresDB) DeletePeer(ctx context.Context, id string) error {
	res, err := p.pool.Exec(ctx, `DELETE FROM peers WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("peer not found: %s", id)
	}
	return nil
}

func (p *PostgresDB) ListPeersByTags(ctx context.Context, workspace string, tags []string) ([]Peer, error) {
	peers, err := p.ListPeers(ctx, workspace)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return peers, nil
	}
	var filtered []Peer
	for _, peer := range peers {
		if hasAllTags(peer.Tags, tags) {
			filtered = append(filtered, peer)
		}
	}
	return filtered, nil
}

func (p *PostgresDB) Close() error {
	p.pool.Close()
	return nil
}

func (p *PostgresDB) SendMessage(ctx context.Context, msg Message) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO messages (id, from_peer, to_peer, workspace, payload) VALUES ($1,$2,$3,$4,$5)`,
		msg.ID, msg.FromPeer, msg.ToPeer, msg.Workspace, msg.Payload)
	return err
}

func (p *PostgresDB) PollMessages(ctx context.Context, workspace, toPeer string, limit int) ([]Message, error) {
	rows, err := p.pool.Query(ctx,
		`SELECT id, from_peer, to_peer, workspace, payload, created_at
		 FROM messages WHERE workspace=$1 AND (to_peer=$2 OR to_peer='') ORDER BY created_at ASC LIMIT $3`,
		workspace, toPeer, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.FromPeer, &m.ToPeer, &m.Workspace, &m.Payload, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (p *PostgresDB) DeleteMessage(ctx context.Context, id string) error {
	res, err := p.pool.Exec(ctx, `DELETE FROM messages WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("message not found: %s", id)
	}
	return nil
}

func (p *PostgresDB) SaveArtifact(ctx context.Context, a Artifact) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO artifacts (id, workspace, name, kind, content) VALUES ($1,$2,$3,$4,$5)`,
		a.ID, a.Workspace, a.Name, a.Kind, a.Content)
	return err
}

func (p *PostgresDB) GetArtifact(ctx context.Context, id string) (*Artifact, error) {
	row := p.pool.QueryRow(ctx,
		`SELECT id, workspace, name, kind, content, created_at FROM artifacts WHERE id=$1`, id)
	var a Artifact
	if err := row.Scan(&a.ID, &a.Workspace, &a.Name, &a.Kind, &a.Content, &a.CreatedAt); err != nil {
		return nil, fmt.Errorf("artifact not found: %w", err)
	}
	return &a, nil
}
