// Package sdk fornece um cliente oficial em Go para interagir com o daemon Vectora.
package sdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Kaffyn/Vectora/pkg/types"
)

// Client é o ponto de entrada para a API do Vectora.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Token      string
}

// NewClient cria uma nova instância do SDK.
func NewClient(baseURL string, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HealthCheck verifica se o daemon está online.
func (c *Client) HealthCheck(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/v1/health", nil)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("daemon respondeu com status %d", resp.StatusCode)
	}
	return nil
}

// Query envia uma pergunta ao mecanismo RAG.
func (c *Client) Query(ctx context.Context, workspaceID string, query string) (string, error) {
	payload := map[string]string{
		"workspace_id": workspaceID,
		"query":        query,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/api/v1/chat", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Answer string `json:"answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Answer, nil
}
