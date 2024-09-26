package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/influxdb3"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TokenResource{}
	_ resource.ResourceWithImportState = &TokenResource{}
	_ resource.ResourceWithImportState = &TokenResource{}
)

// NewTokenResource is a helper function to simplify the provider implementation.
func NewTokenResource() resource.Resource {
	return &TokenResource{}
}

// TokenResource defines the resource implementation.
type TokenResource struct {
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
}

// Metadata returns the resource type name.
func (r *TokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_token"
}

// Schema defines the schema for the resource.
func (r *TokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the database token.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.ListNestedAttribute{
				Required:    true,
				Description: "The list of permissions the database token allows.",
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
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
	var permissionsRequest []influxdb3.DatabaseTokenPermission
	for _, permission := range plan.Permissions {
		resource := influxdb3.DatabaseTokenPermissionResource{}
		resource.FromClusterDatabaseName(permission.Resource.ValueString())
		permission := influxdb3.DatabaseTokenPermission{
			Action:   permission.Action.ValueStringPointer(),
			Resource: &resource,
		}
		permissionsRequest = append(permissionsRequest, permission)
	}

	createTokenRequest := influxdb3.CreateDatabaseTokenJSONRequestBody{
		Description: plan.Description.ValueString(),
		Permissions: &permissionsRequest,
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
		errMsg, err := formatErrorResponse(createTokenResponse, createTokenResponse.StatusCode())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error formatting error response",
				err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error creating token",
			errMsg,
		)
		return
	}
	createToken := *createTokenResponse.JSON200

	// Map response body to schema and populate Computed attribute values
	plan.AccessToken = types.StringValue(createToken.AccessToken)
	plan.AccountId = types.StringValue(createToken.AccountId.String())
	plan.CreatedAt = types.StringValue(createToken.CreatedAt.String())
	plan.ClusterId = types.StringValue(createToken.ClusterId.String())
	plan.Description = types.StringValue(createToken.Description)
	plan.Id = types.StringValue(createToken.Id.String())
	plan.Permissions = getPermissions(createToken.Permissions)

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

	if readTokenResponse.StatusCode() != 200 {
		errMsg, err := formatErrorResponse(readTokenResponse, readTokenResponse.StatusCode())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error formatting error response",
				err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error getting token",
			errMsg,
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
	var permissionsRequest []influxdb3.DatabaseTokenPermission
	for _, permission := range plan.Permissions {
		resource := influxdb3.DatabaseTokenPermissionResource{}
		resource.FromClusterDatabaseName(permission.Resource.ValueString())
		permission := influxdb3.DatabaseTokenPermission{
			Action:   permission.Action.ValueStringPointer(),
			Resource: &resource,
		}
		permissionsRequest = append(permissionsRequest, permission)
	}

	updateTokenRequest := influxdb3.UpdateDatabaseTokenJSONRequestBody{
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
		errMsg, err := formatErrorResponse(updateTokenResponse, updateTokenResponse.StatusCode())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error formatting error response",
				err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error updating token",
			errMsg,
		)
		return
	}
	updateToken := *updateTokenResponse.JSON200

	// Overwrite items with refreshed state
	plan.AccountId = types.StringValue(updateToken.AccountId.String())
	plan.CreatedAt = types.StringValue(updateToken.CreatedAt.String())
	plan.ClusterId = types.StringValue(updateToken.ClusterId.String())
	plan.Description = types.StringValue(updateToken.Description)
	plan.Id = types.StringValue(updateToken.Id.String())
	plan.Permissions = getPermissions(updateToken.Permissions)

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
		errMsg, err := formatErrorResponse(deleteTokenResponse, deleteTokenResponse.StatusCode())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error formatting error response",
				err.Error(),
			)
			return
		}
		resp.Diagnostics.AddError(
			"Error deleting token",
			errMsg,
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *TokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	pd, ok := req.ProviderData.(providerData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected influxdb3.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.accountID = pd.accountID
	r.client = pd.client
	r.clusterID = pd.clusterID
}

func (r *TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
