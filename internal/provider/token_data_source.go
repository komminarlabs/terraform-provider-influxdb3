package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/influxdb3"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &TokenDataSource{}
	_ datasource.DataSourceWithConfigure = &TokenDataSource{}
)

// NewTokenDataSource is a helper function to simplify the provider implementation.
func NewTokenDataSource() datasource.DataSource {
	return &TokenDataSource{}
}

// TokensDataSource is the data source implementation.
type TokenDataSource struct {
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
}

// Metadata returns the data source type name.
func (d *TokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

// Schema defines the schema for the data source.
func (d *TokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Gets a database token. Use this data source to retrieve information about a database token, including the token's permissions.",

		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Computed:    true,
				Description: "The access token that can be used to authenticate query and write requests to the cluster. The access token is never stored by InfluxDB and is only returned once when the token is created. If the access token is lost, a new token must be created.",
				Sensitive:   true,
			},
			"account_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the account that the database token belongs to.",
			},
			"created_at": schema.StringAttribute{
				Computed:    true,
				Description: "The date and time that the database token was created. Uses RFC3339 format.",
			},
			"cluster_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the cluster that the database token belongs to.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The description of the database token.",
			},
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The ID of the database token.",
			},
			"permissions": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The list of permissions the database token allows.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Computed:    true,
							Description: "The action the database token permission allows.",
						},
						"resource": schema.StringAttribute{
							Computed:    true,
							Description: "The resource the database token permission applies to. `*` refers to all databases.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected influxdb3.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.accountID = pd.accountID
	d.client = pd.client
	d.clusterID = pd.clusterID
}

// Read refreshes the Terraform state with the latest data.
func (d *TokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TokenModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// parse the token ID
	tokenId, err := uuid.Parse(state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error. Ensure the Id is in UUID format.",
			err.Error(),
		)
		return
	}

	readTokenResponse, err := d.client.GetDatabaseTokenWithResponse(ctx, d.accountID, d.clusterID, tokenId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting token",
			err.Error(),
		)
		return
	}

	if readTokenResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error getting token",
			fmt.Sprintf("Status: %s", readTokenResponse.Status()),
		)
		return
	}
	readToken := *readTokenResponse.JSON200

	// Overwrite items with refreshed state
	state.AccountId = types.StringValue(readToken.AccountId.String())
	state.CreatedAt = types.StringValue(readToken.CreatedAt.String())
	state.ClusterId = types.StringValue(readToken.ClusterId.String())
	state.Description = types.StringValue(readToken.Description)
	state.Id = types.StringValue(readToken.Id.String())
	state.Permissions = getPermissions(readToken.Permissions)

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
