package db

import (
	"context"
	"time"
)

type Peer struct {
	ID        string
	Workspace string
	Name      string
	Kind      string
	Tags      []string
	IP        string
	Port      int
	LastSeen  time.Time
}

type DB interface {
	RegisterPeer(ctx context.Context, peer Peer) error
	GetPeer(ctx context.Context, workspace, name string) (*Peer, error)
	ListPeers(ctx context.Context, workspace string) ([]Peer, error)
	DeletePeer(ctx context.Context, id string) error
	Close() error
}
