package provider

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/thulasirajkomminar/influxdb3-management-go"
)

// INFLUXDB3_HOST is the default InfluxDB V3 API host.
// INFLUXDB3_API_ENDPOINT is the default InfluxDB V3 API endpoint.
const (
	INFLUXDB3_HOST         = "https://console.influxdata.com"
	INFLUXDB3_API_ENDPOINT = "/api/v0"
)

// Ensure the implementation satisfies the expected interfaces.
var _ provider.Provider = &InfluxDBProvider{}

// InfluxDBProvider defines the provider implementation.
type InfluxDBProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// InfluxDBProviderModel maps provider schema data to a Go type.
type InfluxDBProviderModel struct {
	AccountID types.String `tfsdk:"account_id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Host      types.String `tfsdk:"host"`
	Token     types.String `tfsdk:"token"`
}

type providerData struct {
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
}

// Metadata returns the provider type name.
func (p *InfluxDBProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "influxdb3"
	resp.Version = p.version
}

// Schema defines the provider-level schema for configuration data.
func (p *InfluxDBProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "InfluxDB provider to deploy and manage resources supported by InfluxDB V3.",

		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Description: "The ID of the account that the cluster belongs to. Can also be set with the `INFLUXDB3_ACCOUNT_ID` environment variable.",
				Optional:    true,
			},
			"cluster_id": schema.StringAttribute{
				Description: "The ID of the cluster that you want to manage. Can also be set with the `INFLUXDB3_CLUSTER_ID` environment variable.",
				Optional:    true,
			},
			"host": schema.StringAttribute{
				Description: "The InfluxDB V3 management API host URL. The default is `" + INFLUXDB3_HOST + "`. Can also be set with the `INFLUXDB3_HOST` environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "The InfluxDB management token. Can also be set with the `INFLUXDB3_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
		},
	}
}

// Configure prepares a InfluxDB API client for data sources and resources.
func (p *InfluxDBProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Retrieve provider data from configuration
	var config InfluxDBProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If practitioner provided a configuration value for any of the
	// attributes, it must be a known value.

	if config.AccountID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Unknown InfluxDB V3 Account ID",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB V3 Account ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB3_ACCOUNT_ID environment variable.",
		)
	}

	if config.ClusterID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("cluster_id"),
			"Unknown InfluxDB V3 Cluster ID",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB V3 Cluster ID. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB3_CLUSTER_ID environment variable.",
		)
	}

	if config.Host.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("host"),
			"Unknown InfluxDB V3 Host",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB V3 Host. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB3_HOST environment variable.",
		)
	}

	if config.Token.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Unknown InfluxDB V3 Management Token",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB V3 Management Token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB3_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	accountID := os.Getenv("INFLUXDB3_ACCOUNT_ID")
	clusterID := os.Getenv("INFLUXDB3_CLUSTER_ID")
	host := os.Getenv("INFLUXDB3_HOST")
	token := os.Getenv("INFLUXDB3_TOKEN")

	if !config.AccountID.IsNull() {
		accountID = config.AccountID.ValueString()
	}

	if !config.ClusterID.IsNull() {
		clusterID = config.ClusterID.ValueString()
	}

	if !config.Host.IsNull() {
		host = config.Host.ValueString()
	}

	if !config.Token.IsNull() {
		token = config.Token.ValueString()
	}

	if host == "" {
		host = INFLUXDB3_HOST
	}

	// Combine host and endpoint
	url := strings.TrimSuffix(host, "/") + INFLUXDB3_API_ENDPOINT

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if accountID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Missing InfluxDB V3 Account ID",
			"The provider cannot create the InfluxDB client as there is a missing or empty value for the InfluxDB V3 Account ID. "+
				"Set the Account ID value in the configuration or use the INFLUXDB3_ACCOUNT_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if clusterID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("cluster_id"),
			"Missing InfluxDB V3 Cluster ID",
			"The provider cannot create the InfluxDB client as there is a missing or empty value for the InfluxDB V3 Cluster ID. "+
				"Set the Cluster ID value in the configuration or use the INFLUXDB3_CLUSTER_ID environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("token"),
			"Missing InfluxDB Management Token",
			"The provider cannot create the InfluxDB client as there is a missing or empty value for the InfluxDB V3 Management Token. "+
				"Set the Management Token value in the configuration or use the INFLUXDB3_TOKEN environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// If any of the expected configurations are in wrong format, return
	// errors with provider-specific guidance.

	accountUUID, err := uuid.Parse(accountID)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Invalid InfluxDB V3 Account ID",
			"The provider cannot create the InfluxDB client as there is an incorrect value for the InfluxDB V3 Account ID. "+
				"Set the Account ID value in the configuration or use the INFLUXDB3_ACCOUNT_ID environment variable. "+
				"If either is already set, ensure the value is in UUID format.",
		)
		return
	}

	clusterUUID, err := uuid.Parse(clusterID)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("cluster_id"),
			"Invalid InfluxDB V3 Cluster ID",
			"The provider cannot create the InfluxDB client as there is an incorrect value for the InfluxDB V3 Cluster ID. "+
				"Set the Cluster ID value in the configuration or use the INFLUXDB3_CLUSTER_ID environment variable. "+
				"If either is already set, ensure the value is in UUID format.",
		)
		return
	}

	ctx = tflog.SetField(ctx, "INFLUXDB3_ACCOUNT_ID", accountID)
	ctx = tflog.SetField(ctx, "INFLUXDB3_CLUSTER_ID", clusterID)
	ctx = tflog.SetField(ctx, "INFLUXDB3_URL", url)

	tflog.Debug(ctx, "Creating InfluxDB V3 client")

	// Create a new InfluxDB client using the configuration values

	// Create a new retryable HTTP client with exponential backoff
	retryClient := retryablehttp.NewClient()
	retryClient.Backoff = retryablehttp.LinearJitterBackoff
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.RetryMax = 3

	// Log every request/response (per retry attempt) when Terraform debug
	// logging is enabled, with credentials masked. The transport also adds
	// the Authorization header.
	retryClient.HTTPClient.Transport = newLoggingHTTPTransport(token, retryClient.HTTPClient.Transport)

	client, err := influxdb3.NewClientWithResponses(url, influxdb3.WithRequestEditorFn(func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Accept", "application/json")
		return nil
	}), influxdb3.WithHTTPClient(retryClient.StandardClient()))
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create InfluxDB V3 Client",
			"An unexpected error occurred when creating the InfluxDB V3 client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"InfluxDB V3 Client Error: "+err.Error(),
		)
		return
	}

	// Make the InfluxDB client available during DataSource and Resource
	// type Configure methods.

	providerData := &providerData{
		accountID: accountUUID,
		client:    *client,
		clusterID: clusterUUID,
	}
	resp.DataSourceData = *providerData
	resp.ResourceData = *providerData
	tflog.Info(ctx, "Configured InfluxDB V3 client", map[string]any{"success": true})
}

// Resources defines the resources implemented in the provider.
func (p *InfluxDBProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewTokenResource,
		NewDatabaseResource,
		NewTableResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *InfluxDBProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewTokenDataSource,
		NewTokensDataSource,
		NewDatabaseDataSource,
		NewDatabasesDataSource,
	}
}

// New is a helper function to simplify provider server and testing implementation.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &InfluxDBProvider{
			version: version,
		}
	}
}
