package influxdb3

import "net/http"

type ClientConfig struct {
	AccountID  string
	ClusterID  string
	Host       string
	HTTPClient *http.Client
	Token      string
}
