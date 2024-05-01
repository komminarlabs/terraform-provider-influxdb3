package influxdb3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
)

type TokenAPI interface {
	CreateToken(ctx context.Context, tokenParams *TokenParams) (*token, error)
	DeleteToken(ctx context.Context, tokenID string) error
	GetTokens(ctx context.Context) ([]token, error)
	GetTokenByID(ctx context.Context, tokenID string) (*token, error)
	UpdateToken(ctx context.Context, tokenID string, tokenParams *TokenParams) (*token, error)
}

const (
	TokenAPIPath = "tokens"
)

type TokenParams struct {
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

type token struct {
	AccessToken string       `json:"accessToken"`
	AccountId   string       `json:"accountId"`
	CreatedAt   string       `json:"createdAt"`
	ClusterId   string       `json:"clusterId"`
	Description string       `json:"description"`
	Id          string       `json:"id"`
	Permissions []Permission `json:"permissions"`
}

type Permission struct {
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

func (c *client) CreateToken(ctx context.Context, tokenParams *TokenParams) (*token, error) {
	reqBody, err := json.Marshal(tokenParams)
	if err != nil {
		return nil, err
	}

	respBody, err := c.makeAPICall(http.MethodPost, TokenAPIPath, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	token := token{}
	err = json.Unmarshal(respBody, &token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return &token, nil
}

func (c *client) DeleteToken(ctx context.Context, tokenID string) error {
	_, err := c.makeAPICall(http.MethodDelete, path.Join(TokenAPIPath, tokenID), nil)
	if err != nil {
		if err.Error() == "unexpected status code: 204" {
			return nil
		}
		return fmt.Errorf("error deleting token: %w", err)
	}
	return nil
}

func (c *client) GetTokens(ctx context.Context) ([]token, error) {
	tokens := []token{}
	body, err := c.makeAPICall(http.MethodGet, TokenAPIPath, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tokens)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}
	return tokens, nil
}

func (c *client) GetTokenByID(ctx context.Context, tokenID string) (*token, error) {
	token := token{}
	body, err := c.makeAPICall(http.MethodGet, path.Join(TokenAPIPath, tokenID), nil)
	if err != nil {
		if err.Error() == "unexpected status code: 404" {
			return nil, fmt.Errorf("error getting token: %s not found", tokenID)
		}
		return nil, err
	}

	err = json.Unmarshal(body, &token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}
	return &token, nil
}

func (c *client) UpdateToken(ctx context.Context, tokenID string, tokenParams *TokenParams) (*token, error) {
	reqBody, err := json.Marshal(tokenParams)
	if err != nil {
		return nil, err
	}

	respBody, err := c.makeAPICall(http.MethodPatch, path.Join(TokenAPIPath, tokenID), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	token := token{}
	err = json.Unmarshal(respBody, &token)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return &token, nil
}
