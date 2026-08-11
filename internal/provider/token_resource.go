package provider

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thulasirajkomminar/influxdb3-management-go/cloud"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TokenResource{}
	_ resource.ResourceWithConfigure   = &TokenResource{}
	_ resource.ResourceWithImportState = &TokenResource{}
	_ resource.ResourceWithMoveState   = &TokenResource{}
)

// NewTokenResource returns the deprecated influxdb3_token alias of the
// influxdb3_cloud_token resource.
func NewTokenResource() resource.Resource {
	return &TokenResource{aliasedType: aliasedType{typeSuffix: "_token", deprecated: true}}
}

// NewCloudTokenResource is a helper function to simplify the provider implementation.
func NewCloudTokenResource() resource.Resource {
	return &TokenResource{aliasedType: aliasedType{typeSuffix: "_cloud_token"}}
}

// TokenResource defines the resource implementation.
type TokenResource struct {
	aliasedType
	accountID influxdb3cloud.UuidV4
	client    influxdb3cloud.ClientWithResponses
	clusterID influxdb3cloud.UuidV4
}

// Metadata returns the resource type name.
func (r *TokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.typeSuffix
}

// Schema defines the schema for the resource.
func (r *TokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	defer r.applyResourceDeprecation(resp)
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Creates and manages a token and returns the generated database token. Use this resource to create/manage a token, which generates an database token with permissions to read or write to a specific database.",

		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				Computed:    true,
				Description: "The access token that can be used to authenticate query and write requests to the cluster. The access token is never stored by InfluxDB and is only returned once when the token is created. If the access token is lost, a new token must be created.",
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				Required:    true,
				Description: "The description of the database token.",
			},
			"expires_at": schema.StringAttribute{
				CustomType:  timetypes.RFC3339Type{},
				Optional:    true,
				Description: "The date and time that the database token expires, if applicable. Uses RFC3339 format(for example: 2020-01-01T00:00:00Z).",
				Validators: []validator.String{
					rfc3339NoSubsecondsValidator{},
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the database token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.SetNestedAttribute{
				Required:    true,
				Description: "The set of permissions the database token allows.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"action": schema.StringAttribute{
							Required:    true,
							Description: "The action the database token permission allows. Valid values are `read` or `write`.",
							Validators: []validator.String{
								stringvalidator.OneOf([]string{"read", "write"}...),
							},
						},
						"resource": schema.StringAttribute{
							Required:    true,
							Description: "The resource the database token permission applies to. `*` refers to all databases.",
						},
					},
				},
			},
			"revoked_at": schema.StringAttribute{
				CustomType:  timetypes.RFC3339Type{},
				Computed:    true,
				Description: "The date and time that the database token was revoked, if applicable. Uses RFC3339 format.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *TokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TokenModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	permissionsRequest, err := buildPermissions(plan.Permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error. Ensure the Resource is in the correct format.",
			err.Error(),
		)
		return
	}

	createTokenRequest := influxdb3cloud.CreateDatabaseTokenJSONRequestBody{
		Description: plan.Description.ValueString(),
		Permissions: &permissionsRequest,
	}

	if !plan.ExpiresAt.IsNull() && !plan.ExpiresAt.IsUnknown() {
		t, diags := plan.ExpiresAt.ValueRFC3339Time()
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createTokenRequest.ExpiresAt = &t
	}

	createTokenResponse, err := r.client.CreateDatabaseTokenWithResponse(ctx, r.accountID, r.clusterID, createTokenRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating token",
			"Could not create token, unexpected error: "+err.Error(),
		)
		return
	}

	if createTokenResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error creating token",
			formatErrorResponse(createTokenResponse, createTokenResponse.StatusCode()),
		)
		return
	}
	if createTokenResponse.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Error creating token",
			formatEmptyResponse(createTokenResponse, createTokenResponse.StatusCode()),
		)
		return
	}
	createToken := *createTokenResponse.JSON200

	// Map response body to schema and populate Computed attribute values
	plan.AccessToken = types.StringValue(createToken.AccessToken)
	plan.AccountId = types.StringValue(createToken.AccountId.String())
	plan.CreatedAt = types.StringValue(createToken.CreatedAt.Format(time.RFC3339Nano))
	plan.ClusterId = types.StringValue(createToken.ClusterId.String())
	plan.Description = types.StringValue(createToken.Description)
	plan.Id = types.StringValue(createToken.Id.String())
	plan.Permissions = getPermissions(createToken.Permissions)
	plan.ExpiresAt = timetypes.NewRFC3339TimePointerValue(createToken.ExpiresAt)
	plan.RevokedAt = timetypes.NewRFC3339TimePointerValue(createToken.RevokedAt)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *TokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state TokenModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	// Get refreshed token value from InfluxDB
	readTokenResponse, err := r.client.GetDatabaseTokenWithResponse(ctx, r.accountID, r.clusterID, tokenId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting token",
			err.Error(),
		)
		return
	}

	if readTokenResponse.StatusCode() == 404 {
		// The token no longer exists; remove it from state so
		// Terraform can plan to recreate it.
		resp.State.RemoveResource(ctx)
		return
	}

	if readTokenResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error getting token",
			formatErrorResponse(readTokenResponse, readTokenResponse.StatusCode()),
		)
		return
	}
	if readTokenResponse.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Error getting token",
			formatEmptyResponse(readTokenResponse, readTokenResponse.StatusCode()),
		)
		return
	}
	readToken := *readTokenResponse.JSON200

	// Overwrite items with refreshed state
	state.AccountId = types.StringValue(readToken.AccountId.String())
	state.CreatedAt = types.StringValue(readToken.CreatedAt.Format(time.RFC3339Nano))
	state.ClusterId = types.StringValue(readToken.ClusterId.String())
	state.Description = types.StringValue(readToken.Description)
	state.Id = types.StringValue(readToken.Id.String())
	state.Permissions = getPermissions(readToken.Permissions)
	state.ExpiresAt = timetypes.NewRFC3339TimePointerValue(readToken.ExpiresAt)
	state.RevokedAt = timetypes.NewRFC3339TimePointerValue(readToken.RevokedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *TokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TokenModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// parse the token ID
	tokenId, err := uuid.Parse(plan.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error. Ensure the Id is in UUID format.",
			err.Error(),
		)
		return
	}

	// Generate API request body from plan
	permissionsRequest, err := buildPermissions(plan.Permissions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Validation error. Ensure the Resource is in the correct format.",
			err.Error(),
		)
		return
	}

	updateTokenRequest := influxdb3cloud.UpdateDatabaseTokenJSONRequestBody{
		Description: plan.Description.ValueStringPointer(),
		Permissions: &permissionsRequest,
	}

	// Update existing token
	updateTokenResponse, err := r.client.UpdateDatabaseTokenWithResponse(ctx, r.accountID, r.clusterID, tokenId, updateTokenRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating token",
			"Could not update token, unexpected error: "+err.Error(),
		)
		return
	}

	if updateTokenResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error updating token",
			formatErrorResponse(updateTokenResponse, updateTokenResponse.StatusCode()),
		)
		return
	}
	if updateTokenResponse.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Error updating token",
			formatEmptyResponse(updateTokenResponse, updateTokenResponse.StatusCode()),
		)
		return
	}
	updateToken := *updateTokenResponse.JSON200

	// Overwrite items with refreshed state
	plan.AccountId = types.StringValue(updateToken.AccountId.String())
	plan.CreatedAt = types.StringValue(updateToken.CreatedAt.Format(time.RFC3339Nano))
	plan.ClusterId = types.StringValue(updateToken.ClusterId.String())
	plan.Description = types.StringValue(updateToken.Description)
	plan.Id = types.StringValue(updateToken.Id.String())
	plan.Permissions = getPermissions(updateToken.Permissions)
	plan.ExpiresAt = timetypes.NewRFC3339TimePointerValue(updateToken.ExpiresAt)
	plan.RevokedAt = timetypes.NewRFC3339TimePointerValue(updateToken.RevokedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TokenModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	// Delete existing token
	deleteTokenResponse, err := r.client.DeleteDatabaseTokenWithResponse(ctx, r.accountID, r.clusterID, tokenId)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting token",
			"Could not delete token, unexpected error: "+err.Error(),
		)
		return
	}

	if deleteTokenResponse.StatusCode() != 204 {
		resp.Diagnostics.AddError(
			"Error deleting token",
			formatErrorResponse(deleteTokenResponse, deleteTokenResponse.StatusCode()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *TokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	pd, ok := newProviderData(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !pd.requireDeploymentType(r.typeName(), &resp.Diagnostics, typeCloud) {
		return
	}

	r.accountID = pd.accountID
	r.client = pd.client
	r.clusterID = pd.clusterID
}

func (r *TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// MoveState enables a moved block from the deprecated influxdb3_token
// resource to influxdb3_cloud_token.
func (r *TokenResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	return aliasStateMover(r.legacyTypeName(), schemaResp.Schema)
}
