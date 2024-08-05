package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/influxdb3"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &TokensDataSource{}
	_ datasource.DataSourceWithConfigure = &TokensDataSource{}
)

// NewTokensDataSource is a helper function to simplify the provider implementation.
func NewTokensDataSource() datasource.DataSource {
	return &TokensDataSource{}
}

// TokensDataSource is the data source implementation.
type TokensDataSource struct {
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
}

// TokensDataSourceModel describes the data source data model.
type TokensDataSourceModel struct {
	Tokens []TokenModel `tfsdk:"tokens"`
}

// Metadata returns the data source type name.
func (d *TokensDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tokens"
}

// Schema defines the schema for the data source.
func (d *TokensDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Gets all database tokens for a cluster.",

		Attributes: map[string]schema.Attribute{
			"tokens": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
							Computed:    true,
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *TokensDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *TokensDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TokensDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readTokens, err := d.client.GetDatabaseTokensWithResponse(ctx, d.accountID, d.clusterID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting tokens",
			err.Error(),
		)
		return
	}

	if readTokens.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error getting tokens",
			fmt.Sprintf("Status: %s", readTokens.Status()),
		)
		return
	}

	// Map response body to model
	for _, token := range *readTokens.JSON200 {
		tokenState := TokenModel{
			AccountId:   types.StringValue(token.AccountId.String()),
			CreatedAt:   types.StringValue(token.CreatedAt.String()),
			ClusterId:   types.StringValue(token.ClusterId.String()),
			Description: types.StringValue(token.Description),
			Id:          types.StringValue(token.Id.String()),
			Permissions: getPermissions(token.Permissions),
		}
		state.Tokens = append(state.Tokens, tokenState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
