package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/terraform-provider-influxdb3/internal/sdk/influxdb3"
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
	client influxdb3.Client
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
	var permissions []influxdb3.Permission
	for _, permissionData := range plan.Permissions {
		permission := influxdb3.Permission{
			Action:   permissionData.Action.ValueString(),
			Resource: permissionData.Resource.ValueString(),
		}
		permissions = append(permissions, permission)
	}

	createToken := influxdb3.TokenParams{
		Description: plan.Description.ValueString(),
		Permissions: permissions,
	}

	apiResponse, err := r.client.TokenAPI().CreateToken(ctx, &createToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating token",
			"Could not create token, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.AccessToken = types.StringValue(apiResponse.AccessToken)
	plan.AccountId = types.StringValue(apiResponse.AccountId)
	plan.CreatedAt = types.StringValue(apiResponse.CreatedAt)
	plan.ClusterId = types.StringValue(apiResponse.ClusterId)
	plan.Description = types.StringValue(apiResponse.Description)
	plan.Id = types.StringValue(apiResponse.Id)
	plan.Permissions = getPermissions(apiResponse.Permissions)

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

	// Get refreshed token value from InfluxDB
	readToken, err := r.client.TokenAPI().GetTokenByID(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Tokens",
			err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.AccountId = types.StringValue(readToken.AccountId)
	state.CreatedAt = types.StringValue(readToken.CreatedAt)
	state.ClusterId = types.StringValue(readToken.ClusterId)
	state.Description = types.StringValue(readToken.Description)
	state.Id = types.StringValue(readToken.Id)
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

	// Generate API request body from plan
	var permissions []influxdb3.Permission
	for _, permissionData := range plan.Permissions {
		permission := influxdb3.Permission{
			Action:   permissionData.Action.ValueString(),
			Resource: permissionData.Resource.ValueString(),
		}
		permissions = append(permissions, permission)
	}

	updateToken := influxdb3.TokenParams{
		Description: plan.Description.ValueString(),
		Permissions: permissions,
	}

	// Update existing token
	apiResponse, err := r.client.TokenAPI().UpdateToken(ctx, plan.Id.ValueString(), &updateToken)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating token",
			"Could not update token, unexpected error: "+err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	plan.AccountId = types.StringValue(apiResponse.AccountId)
	plan.CreatedAt = types.StringValue(apiResponse.CreatedAt)
	plan.ClusterId = types.StringValue(apiResponse.ClusterId)
	plan.Description = types.StringValue(apiResponse.Description)
	plan.Id = types.StringValue(apiResponse.Id)
	plan.Permissions = getPermissions(apiResponse.Permissions)

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

	// Delete existing token
	err := r.client.TokenAPI().DeleteToken(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting token",
			"Could not delete token, unexpected error: "+err.Error(),
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

	client, ok := req.ProviderData.(influxdb3.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected influxdb3.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *TokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func getPermissions(permissions []influxdb3.Permission) []TokenPermissionModel {
	permissionsState := []TokenPermissionModel{}
	for _, permission := range permissions {
		permissionState := TokenPermissionModel{
			Action:   types.StringValue(permission.Action),
			Resource: types.StringValue(permission.Resource),
		}
		permissionsState = append(permissionsState, permissionState)
	}
	return permissionsState
}
