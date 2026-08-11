package provider

import (
	"net/http"
	"regexp"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
)

// accessTokenRegex matches access token values in API response bodies (for
// example the one-time access_token returned when creating a database token)
// so they can be masked in debug logs.
var accessTokenRegex = regexp.MustCompile(`"access_token":\s*"[^"]*"`)

// newLoggingHTTPTransport builds the HTTP transport chain used by the
// InfluxDB client. With Terraform debug logging enabled (TF_LOG=DEBUG), every
// HTTP request and response is logged via tflog.
//
// The chain is ordered so that no credentials reach the logs:
//
//	maskingRoundTripper -> loggingHttpTransport -> bearerTokenRoundTripper -> base
//
// The Authorization header is added below the logging transport and is
// therefore never logged, and the masking wrapper masks token values that
// appear in logged response bodies.
func newLoggingHTTPTransport(token string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	// An empty token (core/enterprise servers running without authentication)
	// must not be turned into a mask regex, as the empty pattern matches
	// everywhere.
	maskRegexes := []*regexp.Regexp{accessTokenRegex}
	if token != "" {
		maskRegexes = append(maskRegexes, regexp.MustCompile(regexp.QuoteMeta(token)))
	}
	return &maskingRoundTripper{
		maskRegexes: maskRegexes,
		next: logging.NewLoggingHTTPTransport(&bearerTokenRoundTripper{
			token: token,
			next:  base,
		}),
	}
}

// maskingRoundTripper injects tflog masking rules into the request context so
// sensitive values are masked in the HTTP logs emitted further down the chain.
type maskingRoundTripper struct {
	maskRegexes []*regexp.Regexp
	next        http.RoundTripper
}

func (m *maskingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := tflog.MaskAllFieldValuesRegexes(req.Context(), m.maskRegexes...)
	return m.next.RoundTrip(req.WithContext(ctx))
}

// bearerTokenRoundTripper adds the Authorization header. It sits below the
// logging transport so the header is never logged.
type bearerTokenRoundTripper struct {
	token string
	next  http.RoundTripper
}

func (b *bearerTokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if b.token == "" {
		// core/enterprise servers can run without authentication.
		return b.next.RoundTrip(req)
	}
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+b.token)
	return b.next.RoundTrip(req)
}
