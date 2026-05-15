package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/AniruthKarthik/llm-orchestrator/internal/providers"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

func NewClient(apikey string, baseURL string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey:  apikey,
		baseURL: baseURL,
	}
}

func (c *Client) Post(
	ctx context.Context,
	endpoint string,
	headers map[string]string,
	body any,
) (*http.Response, error) {

	jsonBody, err := json.Marshal(body)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+endpoint,
		bytes.NewBuffer(jsonBody),
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer"+c.apiKey)
	}

	for key, val := range headers {
		req.Header.Set(key, val)
	}

	resp, err := c.httpClient.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}
