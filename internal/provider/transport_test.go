package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
)

func TestLoggingHTTPTransport(t *testing.T) {
	var buf bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &buf)

	const token = "super-secret-management-token"
	const accessToken = "apiv1_generated-access-token"

	var wireAuthHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wireAuthHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"` + accessToken + `","description":"test"}`))
	}))
	defer server.Close()

	client := &http.Client{Transport: newLoggingHTTPTransport(token, nil)}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %s", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response body: %s", err)
	}

	// The real request and response must be untouched by the logging chain.
	if wireAuthHeader != "Bearer "+token {
		t.Errorf("expected Authorization header on the wire, got: %q", wireAuthHeader)
	}
	if !strings.Contains(string(body), accessToken) {
		t.Errorf("caller must receive the unmasked response body, got: %s", body)
	}

	// The logs must contain the HTTP transaction, but no credentials.
	logs := buf.String()
	if !strings.Contains(logs, "tf_http_req_uri") || !strings.Contains(logs, "tf_http_res_body") {
		t.Errorf("expected request and response to be logged, got: %s", logs)
	}
	if strings.Contains(logs, token) {
		t.Errorf("management token leaked into logs: %s", logs)
	}
	if strings.Contains(logs, accessToken) {
		t.Errorf("access token leaked into logs: %s", logs)
	}
}
