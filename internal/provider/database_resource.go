package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/influxdb3"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DatabaseResource{}
	_ resource.ResourceWithImportState = &DatabaseResource{}
	_ resource.ResourceWithImportState = &DatabaseResource{}
)

// NewDatabaseResource is a helper function to simplify the provider implementation.
func NewDatabaseResource() resource.Resource {
	return &DatabaseResource{}
}

// DatabaseResource defines the resource implementation.
type DatabaseResource struct {
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
}

// Metadata returns the resource type name.
func (r *DatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for the resource.
func (r *DatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Creates and manages a database.",

		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the account that the cluster belongs to.",
			},
			"cluster_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the cluster that you want to manage.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster database. The Length should be between `[ 1 .. 64 ]` characters. **Note:** Database names can't be updated. After a database is deleted, you cannot [reuse](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/delete/#cannot-reuse-database-names) the same name for a new database.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"max_tables": schema.Int64Attribute{
				Computed:    true,
				Optional:    true,
				Default:     int64default.StaticInt64(500),
				Description: "The maximum number of tables for the cluster database. The default is `500`",
			},
			"max_columns_per_table": schema.Int64Attribute{
				Computed:    true,
				Optional:    true,
				Default:     int64default.StaticInt64(200),
				Description: "The maximum number of columns per table for the cluster database. The default is `200`",
			},
			"retention_period": schema.Int64Attribute{
				Computed:    true,
				Optional:    true,
				Default:     int64default.StaticInt64(0),
				Description: "The retention period of the cluster database in nanoseconds. The default is `0`. If the retention period is not set or is set to `0`, the database will have infinite retention.",
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *DatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DatabaseModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	maxTables := int32(plan.MaxTables.ValueInt64())
	maxColumnsPerTable := int32(plan.MaxColumnsPerTable.ValueInt64())
	createDatabaseRequest := influxdb3.CreateClusterDatabaseJSONRequestBody{
		Name:               plan.Name.ValueString(),
		MaxTables:          &maxTables,
		MaxColumnsPerTable: &maxColumnsPerTable,
		RetentionPeriod:    plan.RetentionPeriod.ValueInt64Pointer(),
	}

	createDatabasesResponse, err := r.client.CreateClusterDatabaseWithResponse(ctx, r.accountID, r.clusterID, createDatabaseRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			"Could not create database, unexpected error: "+err.Error(),
		)
		return
	}

	if createDatabasesResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error creating database",
			fmt.Sprintf("Status: %s", createDatabasesResponse.Status()),
		)
		return
	}
	createDatabases := createDatabasesResponse.JSON200

	// Map response body to schema and populate Computed attribute values
	plan.AccountId = types.StringValue(createDatabases.AccountId.String())
	plan.ClusterId = types.StringValue(createDatabases.ClusterId.String())
	plan.Name = types.StringValue(createDatabases.Name)
	plan.MaxTables = types.Int64Value(int64(createDatabases.MaxTables))
	plan.MaxColumnsPerTable = types.Int64Value(int64(createDatabases.MaxColumnsPerTable))
	plan.RetentionPeriod = types.Int64Value(createDatabases.RetentionPeriod)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *DatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state DatabaseModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get refreshed database value from InfluxDB
	readDatabasesResponse, err := r.client.GetClusterDatabasesWithResponse(ctx, r.accountID, r.clusterID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting database",
			err.Error(),
		)
		return
	}

	if readDatabasesResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error getting database",
			fmt.Sprintf("Status: %s", readDatabasesResponse.Status()),
		)
		return
	}

	// Check if the database exists
	readDatabase := getDatabaseByName(*readDatabasesResponse, state.Name.ValueString())
	if readDatabase == nil {
		resp.Diagnostics.AddError(
			"Database not found",
			fmt.Sprintf("Database with name %s not found", state.Name.ValueString()),
		)
		return
	}

	// Overwrite items with refreshed state
	state = *readDatabase

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *DatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DatabaseModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	maxTables := int32(plan.MaxTables.ValueInt64())
	maxColumnsPerTable := int32(plan.MaxColumnsPerTable.ValueInt64())
	updateDatabaseRequest := influxdb3.UpdateClusterDatabaseJSONRequestBody{
		MaxTables:          &maxTables,
		MaxColumnsPerTable: &maxColumnsPerTable,
		RetentionPeriod:    plan.RetentionPeriod.ValueInt64Pointer(),
	}

	// Update existing database
	updateDatabaseResponse, err := r.client.UpdateClusterDatabaseWithResponse(ctx, r.accountID, r.clusterID, plan.Name.ValueString(), updateDatabaseRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating database",
			"Could not update database, unexpected error: "+err.Error(),
		)
		return
	}

	if updateDatabaseResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error updating database",
			fmt.Sprintf("Status: %s", updateDatabaseResponse.Status()),
		)
		return
	}
	updateDatabase := updateDatabaseResponse.JSON200

	// Map response body to schema and populate Computed attribute values
	plan.AccountId = types.StringValue(updateDatabase.AccountId.String())
	plan.ClusterId = types.StringValue(updateDatabase.ClusterId.String())
	plan.Name = types.StringValue(updateDatabase.Name)
	plan.MaxTables = types.Int64Value(int64(updateDatabase.MaxTables))
	plan.MaxColumnsPerTable = types.Int64Value(int64(updateDatabase.MaxColumnsPerTable))
	plan.RetentionPeriod = types.Int64Value(updateDatabase.RetentionPeriod)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *DatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DatabaseModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing database
	deleteDatabasesResponse, err := r.client.DeleteClusterDatabaseWithResponse(ctx, r.accountID, r.clusterID, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			"Could not delete database, unexpected error: "+err.Error(),
		)
		return
	}

	if deleteDatabasesResponse.StatusCode() != 204 {
		resp.Diagnostics.AddError(
			"Error deleting database",
			fmt.Sprintf("Status: %s", deleteDatabasesResponse.Status()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.accountID = pd.accountID
	r.client = pd.client
	r.clusterID = pd.clusterID
}

func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
