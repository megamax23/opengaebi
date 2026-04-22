package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) Enabled() bool { return c.baseURL != "" && c.apiKey != "" }

type PeerPayload struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Tags      []string `json:"tags"`
	IP        string   `json:"ip"`
	Port      int      `json:"port"`
	Status    string   `json:"status"`
	ExpiresIn int      `json:"expires_in"`
}

func (c *Client) PushPeer(ctx context.Context, peer PeerPayload) error {
	if !c.Enabled() {
		return nil
	}
	if peer.Status == "" {
		peer.Status = "online"
	}
	if peer.ExpiresIn == 0 {
		peer.ExpiresIn = 1800
	}
	return c.post(ctx, "/v1/registry/peers", peer, http.StatusCreated)
}

func (c *Client) DeletePeer(ctx context.Context, id string) error {
	if !c.Enabled() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, "DELETE",
		c.baseURL+"/v1/registry/peers/"+id, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil // Registry failure must not block Bridge operation
	}
	resp.Body.Close()
	return nil
}

func (c *Client) Heartbeat(ctx context.Context, id string) error {
	if !c.Enabled() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, "PATCH",
		fmt.Sprintf("%s/v1/registry/peers/%s/heartbeat", c.baseURL, id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil
	}
	resp.Body.Close()
	return nil
}

func (c *Client) post(ctx context.Context, path string, body any, expectedStatus int) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil // Registry failure must not block Bridge operation
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("registry: unexpected status %d", resp.StatusCode)
	}
	return nil
}
