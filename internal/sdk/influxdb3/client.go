package influxdb3

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type Client interface {
	DatabaseAPI() DatabaseAPI
	TokenAPI() TokenAPI
	Close()
}

type client struct {
	config        ClientConfig
	authorization string
	apiURL        *url.URL
}

func New(config *ClientConfig) (Client, error) {
	var err error
	c := &client{config: *config}

	hostAddress := config.Host
	if !strings.HasSuffix(config.Host, "/") {
		hostAddress = config.Host + "/"
	}

	c.apiURL, err = url.Parse(hostAddress)
	if err != nil {
		return nil, fmt.Errorf("parsing host URL: %w", err)
	}

	c.apiURL.Path = path.Join(c.apiURL.Path, fmt.Sprintf("/api/v0/accounts/%s/clusters/%s", c.config.AccountID, c.config.ClusterID)) + "/"
	c.authorization = "Bearer " + c.config.Token

	if c.config.HTTPClient == nil {
		c.config.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	return c, nil
}

func (c *client) Close() {
	c.config.HTTPClient.CloseIdleConnections()
}

func (c *client) DatabaseAPI() DatabaseAPI {
	return c
}

func (c *client) TokenAPI() TokenAPI {
	return c
}

func (c *client) makeAPICall(httpMethod, path string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequest(httpMethod, c.apiURL.String()+path, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", c.authorization)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}
	return respBody, nil
}
