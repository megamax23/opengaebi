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
		)
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

func (p *PostgresDB) Close() error {
	p.pool.Close()
	return nil
}
