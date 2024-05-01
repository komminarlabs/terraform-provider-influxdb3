package influxdb3

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
)

type DatabaseAPI interface {
	CreateDatabase(ctx context.Context, databaseParams *DatabaseParams) (*database, error)
	DeleteDatabase(ctx context.Context, databaseName string) error
	GetDatabases(ctx context.Context) ([]database, error)
	GetDatabaseByName(ctx context.Context, databaseName string) (*database, error)
	UpdateDatabase(ctx context.Context, databaseParams *DatabaseParams) (*database, error)
}

const (
	DatabaseAPIPath = "databases"
)

type database struct {
	AccountId          string              `json:"accountId"`
	ClusterId          string              `json:"clusterId"`
	Name               string              `json:"name"`
	MaxTables          int64               `json:"maxTables"`
	MaxColumnsPerTable int64               `json:"maxColumnsPerTable"`
	RetentionPeriod    int64               `json:"retentionPeriod"`
	PartitionTemplate  []PartitionTemplate `json:"partitionTemplate"`
}

type DatabaseParams struct {
	Name               string              `json:"name"`
	MaxTables          int                 `json:"maxTables"`
	MaxColumnsPerTable int                 `json:"maxColumnsPerTable"`
	RetentionPeriod    int64               `json:"retentionPeriod"`
	PartitionTemplate  []PartitionTemplate `json:"partitionTemplate"`
}

type PartitionTemplate struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

func (c *client) CreateDatabase(ctx context.Context, databaseParams *DatabaseParams) (*database, error) {
	reqBody, err := json.Marshal(databaseParams)
	if err != nil {
		return nil, err
	}

	respBody, err := c.makeAPICall(http.MethodPost, DatabaseAPIPath, bytes.NewBuffer(reqBody))
	if err != nil {
		if err.Error() == "unexpected status code: 400" {
			return nil, fmt.Errorf("bad request, check your input")
		}
		return nil, err
	}

	database := database{}
	err = json.Unmarshal(respBody, &database)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return &database, nil
}

func (c *client) DeleteDatabase(ctx context.Context, databaseName string) error {
	_, err := c.makeAPICall(http.MethodDelete, path.Join(DatabaseAPIPath, databaseName), nil)
	if err != nil {
		if err.Error() == "unexpected status code: 204" {
			return nil
		}
		return fmt.Errorf("error deleting database: %w", err)
	}
	return nil
}

func (c *client) GetDatabases(ctx context.Context) ([]database, error) {
	databases := []database{}
	body, err := c.makeAPICall(http.MethodGet, DatabaseAPIPath, nil)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &databases)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}
	return databases, nil
}

func (c *client) GetDatabaseByName(ctx context.Context, databaseName string) (*database, error) {
	databases, err := c.GetDatabases(ctx)
	if err != nil {
		return nil, err
	}

	for _, db := range databases {
		if db.Name == databaseName {
			return &db, nil
		}
	}
	return nil, fmt.Errorf("error getting database: %s not found", databaseName)
}

func (c *client) UpdateDatabase(ctx context.Context, databaseParams *DatabaseParams) (*database, error) {
	reqBody, err := json.Marshal(databaseParams)
	if err != nil {
		return nil, err
	}

	respBody, err := c.makeAPICall(http.MethodPatch, path.Join(DatabaseAPIPath, databaseParams.Name), bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	database := database{}
	err = json.Unmarshal(respBody, &database)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	return &database, nil
}
