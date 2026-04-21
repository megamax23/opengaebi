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

type Message struct {
	ID        string
	FromPeer  string
	ToPeer    string
	Workspace string
	Payload   string
	CreatedAt time.Time
}

type Artifact struct {
	ID        string
	Workspace string
	Name      string
	Kind      string
	Content   []byte
	CreatedAt time.Time
}

type DB interface {
	RegisterPeer(ctx context.Context, peer Peer) error
	GetPeer(ctx context.Context, workspace, name string) (*Peer, error)
	ListPeers(ctx context.Context, workspace string) ([]Peer, error)
	DeletePeer(ctx context.Context, id string) error

	SendMessage(ctx context.Context, msg Message) error
	PollMessages(ctx context.Context, workspace, toPeer string, limit int) ([]Message, error)
	DeleteMessage(ctx context.Context, id string) error

	SaveArtifact(ctx context.Context, a Artifact) error
	GetArtifact(ctx context.Context, id string) (*Artifact, error)

	Close() error
}
