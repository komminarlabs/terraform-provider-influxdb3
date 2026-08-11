package provider

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/thulasirajkomminar/influxdb3-management-go/cloud"
	"github.com/thulasirajkomminar/influxdb3-management-go/core"
	"github.com/thulasirajkomminar/influxdb3-management-go/enterprise"
)

// INFLUXDB3_HOST is the default InfluxDB V3 Cloud Dedicated API host.
// INFLUXDB3_API_ENDPOINT is the InfluxDB V3 Cloud Dedicated management API endpoint.
const (
	INFLUXDB3_HOST         = "https://console.influxdata.com"
	INFLUXDB3_API_ENDPOINT = "/api/v0"
)

// Supported InfluxDB 3 deployment types.
const (
	typeCloud      = "cloud"
	typeCore       = "core"
	typeEnterprise = "enterprise"
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
	Type      types.String `tfsdk:"type"`
}

// providerData carries the configured deployment type and the API client for
// that type. Exactly one of the clients is populated.
type providerData struct {
	deploymentType   string
	accountID        influxdb3cloud.UuidV4
	client           influxdb3cloud.ClientWithResponses
	clusterID        influxdb3cloud.UuidV4
	coreClient       *influxdb3core.ClientWithResponses
	enterpriseClient *influxdb3enterprise.ClientWithResponses
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
				Description: "The ID of the account that the cluster belongs to. Required for the `cloud` deployment type and not supported for `core` and `enterprise`. Can also be set with the `INFLUXDB3_ACCOUNT_ID` environment variable.",
				Optional:    true,
			},
			"cluster_id": schema.StringAttribute{
				Description: "The ID of the cluster that you want to manage. Required for the `cloud` deployment type and not supported for `core` and `enterprise`. Can also be set with the `INFLUXDB3_CLUSTER_ID` environment variable.",
				Optional:    true,
			},
			"host": schema.StringAttribute{
				Description: "The InfluxDB V3 API host URL. For the `cloud` deployment type the default is `" + INFLUXDB3_HOST + "`. Required for the `core` and `enterprise` deployment types (for example `http://localhost:8181`). Can also be set with the `INFLUXDB3_HOST` environment variable.",
				Optional:    true,
			},
			"token": schema.StringAttribute{
				Description: "The InfluxDB token. For the `cloud` deployment type this is a management token and is required. For the `core` and `enterprise` deployment types this is an admin token (`apiv3_...`) and may be omitted when the server runs without authentication. Can also be set with the `INFLUXDB3_TOKEN` environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"type": schema.StringAttribute{
				Description: "The InfluxDB 3 deployment type to manage. Valid values are `cloud` (InfluxDB Cloud Dedicated), `core` (InfluxDB 3 Core) or `enterprise` (InfluxDB 3 Enterprise). The default is `cloud`. Can also be set with the `INFLUXDB3_TYPE` environment variable.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(typeCloud, typeCore, typeEnterprise),
				},
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

	if config.Type.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Unknown InfluxDB 3 Deployment Type",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB 3 deployment type. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB3_TYPE environment variable.",
		)
	}

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
			"Unknown InfluxDB V3 Token",
			"The provider cannot create the InfluxDB client as there is an unknown configuration value for the InfluxDB V3 token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the INFLUXDB3_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	deploymentType := os.Getenv("INFLUXDB3_TYPE")
	accountID := os.Getenv("INFLUXDB3_ACCOUNT_ID")
	clusterID := os.Getenv("INFLUXDB3_CLUSTER_ID")
	host := os.Getenv("INFLUXDB3_HOST")
	token := os.Getenv("INFLUXDB3_TOKEN")

	if !config.Type.IsNull() {
		deploymentType = config.Type.ValueString()
	}

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

	if deploymentType == "" {
		deploymentType = typeCloud
	}

	// The schema validator only covers values set in the configuration, so
	// values from the INFLUXDB3_TYPE environment variable are checked here.
	switch deploymentType {
	case typeCloud, typeCore, typeEnterprise:
	default:
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Invalid InfluxDB 3 Deployment Type",
			"The InfluxDB 3 deployment type must be one of \"cloud\", \"core\" or \"enterprise\", got: "+deploymentType+". "+
				"Set the type value in the configuration or use the INFLUXDB3_TYPE environment variable.",
		)
		return
	}

	ctx = tflog.SetField(ctx, "INFLUXDB3_TYPE", deploymentType)

	// Create a new retryable HTTP client with backoff, shared by all
	// deployment types.
	retryClient := retryablehttp.NewClient()
	retryClient.Backoff = retryablehttp.LinearJitterBackoff
	retryClient.RetryWaitMin = 1 * time.Second
	retryClient.RetryWaitMax = 5 * time.Second
	retryClient.RetryMax = 3

	// Log every request/response (per retry attempt) when Terraform debug
	// logging is enabled, with credentials masked. The transport also adds
	// the Authorization header.
	retryClient.HTTPClient.Transport = newLoggingHTTPTransport(token, retryClient.HTTPClient.Transport)
	httpClient := retryClient.StandardClient()

	acceptJSON := func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Accept", "application/json")
		return nil
	}

	pd := providerData{deploymentType: deploymentType}

	switch deploymentType {
	case typeCloud:
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

		tflog.Debug(ctx, "Creating InfluxDB V3 Cloud Dedicated client")

		client, err := influxdb3cloud.NewClientWithResponses(url, influxdb3cloud.WithRequestEditorFn(acceptJSON), influxdb3cloud.WithHTTPClient(httpClient))
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Create InfluxDB V3 Client",
				"An unexpected error occurred when creating the InfluxDB V3 client. "+
					"If the error is not clear, please contact the provider developers.\n\n"+
					"InfluxDB V3 Client Error: "+err.Error(),
			)
			return
		}

		pd.accountID = accountUUID
		pd.clusterID = clusterUUID
		pd.client = *client

	case typeCore, typeEnterprise:
		// account_id and cluster_id are Cloud Dedicated concepts. Only values
		// set explicitly in the configuration are rejected, so environment
		// variables left over from managing a Cloud Dedicated cluster do not
		// break core/enterprise configurations.
		if !config.AccountID.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("account_id"),
				"Invalid Provider Configuration",
				"The account_id attribute is only supported for the \"cloud\" deployment type and must not be set when type is \""+deploymentType+"\".",
			)
		}

		if !config.ClusterID.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("cluster_id"),
				"Invalid Provider Configuration",
				"The cluster_id attribute is only supported for the \"cloud\" deployment type and must not be set when type is \""+deploymentType+"\".",
			)
		}

		if host == "" {
			resp.Diagnostics.AddAttributeError(
				path.Root("host"),
				"Missing InfluxDB 3 Host",
				"The provider cannot create the InfluxDB client as there is a missing or empty value for the InfluxDB 3 host, which is required for the \""+deploymentType+"\" deployment type (for example http://localhost:8181). "+
					"Set the host value in the configuration or use the INFLUXDB3_HOST environment variable.",
			)
		}

		if resp.Diagnostics.HasError() {
			return
		}

		url := strings.TrimSuffix(host, "/")

		ctx = tflog.SetField(ctx, "INFLUXDB3_URL", url)

		if token == "" {
			tflog.Warn(ctx, "No InfluxDB token configured; assuming the server runs without authentication")
		}

		tflog.Debug(ctx, "Creating InfluxDB 3 "+deploymentType+" client")

		switch deploymentType {
		case typeCore:
			client, err := influxdb3core.NewClientWithResponses(url, influxdb3core.WithRequestEditorFn(acceptJSON), influxdb3core.WithHTTPClient(httpClient))
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Create InfluxDB 3 Core Client",
					"An unexpected error occurred when creating the InfluxDB 3 Core client. "+
						"If the error is not clear, please contact the provider developers.\n\n"+
						"InfluxDB 3 Core Client Error: "+err.Error(),
				)
				return
			}
			pd.coreClient = client
		case typeEnterprise:
			client, err := influxdb3enterprise.NewClientWithResponses(url, influxdb3enterprise.WithRequestEditorFn(acceptJSON), influxdb3enterprise.WithHTTPClient(httpClient))
			if err != nil {
				resp.Diagnostics.AddError(
					"Unable to Create InfluxDB 3 Enterprise Client",
					"An unexpected error occurred when creating the InfluxDB 3 Enterprise client. "+
						"If the error is not clear, please contact the provider developers.\n\n"+
						"InfluxDB 3 Enterprise Client Error: "+err.Error(),
				)
				return
			}
			pd.enterpriseClient = client
		}
	}

	// Make the InfluxDB client available during DataSource and Resource
	// type Configure methods.

	resp.DataSourceData = pd
	resp.ResourceData = pd
	tflog.Info(ctx, "Configured InfluxDB V3 client", map[string]any{"success": true, "type": deploymentType})
}

// Resources defines the resources implemented in the provider.
func (p *InfluxDBProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// InfluxDB Cloud Dedicated
		NewCloudDatabaseResource,
		NewCloudTableResource,
		NewCloudTokenResource,

		// Deprecated aliases of the influxdb3_cloud_* resources, kept for
		// backwards compatibility. Removed in the next major version.
		NewDatabaseResource,
		NewTableResource,
		NewTokenResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *InfluxDBProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// InfluxDB Cloud Dedicated
		NewCloudDatabaseDataSource,
		NewCloudDatabasesDataSource,
		NewCloudTokenDataSource,
		NewCloudTokensDataSource,

		// Deprecated aliases of the influxdb3_cloud_* data sources, kept for
		// backwards compatibility. Removed in the next major version.
		NewDatabaseDataSource,
		NewDatabasesDataSource,
		NewTokenDataSource,
		NewTokensDataSource,
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
