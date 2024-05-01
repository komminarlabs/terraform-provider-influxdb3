package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/terraform-provider-influxdb3/internal/sdk/influxdb3"
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
	client influxdb3.Client
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
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Computed:    true,
							Description: "The action the database token permission allows. Valid values are `read` or `write`.",
							Validators: []validator.String{
								stringvalidator.OneOf([]string{"read", "write"}...),
							},
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

	client, ok := req.ProviderData.(influxdb3.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected influxdb3.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

// Read refreshes the Terraform state with the latest data.
func (d *TokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state TokenModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	readToken, err := d.client.TokenAPI().GetTokenByID(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Tokens",
			err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.AccessToken = types.StringValue(readToken.AccessToken)
	state.AccountId = types.StringValue(readToken.AccountId)
	state.CreatedAt = types.StringValue(readToken.CreatedAt)
	state.ClusterId = types.StringValue(readToken.ClusterId)
	state.Description = types.StringValue(readToken.Description)
	state.Id = types.StringValue(readToken.Id)
	state.Permissions = getPermissions(readToken.Permissions)

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
