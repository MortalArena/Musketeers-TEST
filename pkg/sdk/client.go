package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MortalArena/Musketeers/pkg/node/subsystems"
	"github.com/MortalArena/Musketeers/pkg/registry"
)

type NeuroClient struct {
	network    *subsystems.NetworkSubsystem
	storage    *subsystems.StorageSubsystem
	agent      any
	identity   any
	registry   registry.Registry
	httpClient *http.Client
	baseURL    string
}

func New(network *subsystems.NetworkSubsystem, storage *subsystems.StorageSubsystem, agent any, identity any, registry registry.Registry) *NeuroClient {
	return &NeuroClient{network: network, storage: storage, agent: agent, identity: identity, registry: registry, httpClient: &http.Client{Timeout: 30 * time.Second}}
}

func (c *NeuroClient) Network() *subsystems.NetworkSubsystem { return c.network }
func (c *NeuroClient) Storage() *subsystems.StorageSubsystem { return c.storage }
func (c *NeuroClient) Agent() any                            { return c.agent }
func (c *NeuroClient) Identity() any                         { return c.identity }

func (c *NeuroClient) SetHTTP(baseURL string, client *http.Client) {
	c.baseURL = baseURL
	if client != nil {
		c.httpClient = client
	}
}

func (c *NeuroClient) Get(ctx context.Context, path string, out any) error {
	if c.baseURL == "" {
		return fmt.Errorf("http base URL is not configured")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http request failed: %s", resp.Status)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *NeuroClient) Registry() registry.Registry {
	return c.registry
}
