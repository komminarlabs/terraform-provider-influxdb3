package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/thulasirajkomminar/influxdb3-management-go"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DatabaseResource{}
	_ resource.ResourceWithConfigure   = &DatabaseResource{}
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
				Description: "The ID of the account that the database belongs to.",
			},
			"cluster_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the cluster that the database belongs to.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster database. The Length should be between `[ 1 .. 64 ]` characters. See the full naming restrictions [here](https://docs.influxdata.com/influxdb3/cloud-dedicated/admin/databases/create/#database-naming-restrictions). <br> **Note:** [Renaming](https://docs.influxdata.com/influxdb3/cloud-dedicated/admin/databases/rename/) a database does not change the database ID, modify data in the database, or update database tokens. After renaming a database, any existing database tokens will stop working and you must create new tokens with permissions for the renamed database. If you create a new database using the previous database name, tokens associated with that database name will grant access to the newly created database.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-/]*$`),
						"must start with a letter or number and only contain alphanumeric characters, underscores (_), dashes (-), and forward-slashes (/)",
					),
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
			"partition_template": schema.ListNestedAttribute{
				Computed:    true,
				Optional:    true,
				Default:     listdefault.StaticValue(types.ListNull(DatabasePartitionTemplateModel{}.GetAttrType())),
				Description: "A template for [partitioning](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) a cluster database. **Note:** A partition template can include up to 7 total tag and tag bucket parts and only 1 time part. You can only apply a partition template when creating a database. You [can't update a partition template](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/create/#partition-templates-can-only-be-applied-on-create) on an existing database. An update will result in resource replacement.",
				Validators: []validator.List{
					listvalidator.UniqueValues(),
					listvalidator.SizeBetween(1, 8),
					partitionTemplateValidator{},
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of template part. Valid values are `bucket`, `tag` or `time`.",
							Validators: []validator.String{
								stringvalidator.OneOf([]string{"bucket", "tag", "time"}...),
							},
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value of template part. **Note:** For `bucket` partition template type use `jsonencode()` function to encode the value to a string.",
						},
					},
				},
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
	partitionTemplates, err := buildPartitionTemplate(plan.PartitionTemplate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database partition template",
			err.Error(),
		)
		return
	}

	maxTables := int32(plan.MaxTables.ValueInt64())
	maxColumnsPerTable := int32(plan.MaxColumnsPerTable.ValueInt64())
	createDatabaseRequest := influxdb3.CreateClusterDatabaseJSONRequestBody{
		MaxTables:          &maxTables,
		MaxColumnsPerTable: &maxColumnsPerTable,
		Name:               plan.Name.ValueString(),
		PartitionTemplate:  &partitionTemplates,
		RetentionPeriod:    plan.RetentionPeriod.ValueInt64Pointer(),
	}

	createDatabaseResponse, err := r.client.CreateClusterDatabaseWithResponse(ctx, r.accountID, r.clusterID, createDatabaseRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			"Could not create database, unexpected error: "+err.Error(),
		)
		return
	}

	if createDatabaseResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error creating database",
			formatErrorResponse(createDatabaseResponse, createDatabaseResponse.StatusCode()),
		)
		return
	}
	createDatabase := createDatabaseResponse.JSON200
	if createDatabase == nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			formatEmptyResponse(createDatabaseResponse, createDatabaseResponse.StatusCode()),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.AccountId = types.StringValue(createDatabase.AccountId.String())
	plan.ClusterId = types.StringValue(createDatabase.ClusterId.String())
	plan.MaxTables = types.Int64Value(int64(createDatabase.MaxTables))
	plan.MaxColumnsPerTable = types.Int64Value(int64(createDatabase.MaxColumnsPerTable))
	plan.Name = types.StringValue(createDatabase.Name)
	plan.RetentionPeriod = types.Int64Value(createDatabase.RetentionPeriod)

	partitionTemplate, err := getPartitionTemplate(createDatabase.PartitionTemplate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting database partition template",
			"Could not create database, unexpected error: "+err.Error(),
		)
		return
	}
	plan.PartitionTemplate = partitionTemplate

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
			formatErrorResponse(readDatabasesResponse, readDatabasesResponse.StatusCode()),
		)
		return
	}

	// Check if the database exists
	readDatabase, err := getDatabaseByName(*readDatabasesResponse, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting database",
			err.Error(),
		)
		return
	}
	if readDatabase == nil {
		// The database no longer exists; remove it from state so
		// Terraform can plan to recreate it.
		tflog.Warn(ctx, "Database not found, removing from state", map[string]any{"name": state.Name.ValueString()})
		resp.State.RemoveResource(ctx)
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
	var state DatabaseModel

	// Read Terraform plan and state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nameChanged := !plan.Name.Equal(state.Name)
	otherFieldsChanged := !plan.MaxTables.Equal(state.MaxTables) ||
		!plan.MaxColumnsPerTable.Equal(state.MaxColumnsPerTable) ||
		!plan.RetentionPeriod.Equal(state.RetentionPeriod)

	// Check if name changed then call rename API
	if nameChanged {
		tflog.Debug(ctx, "Name change detected, calling rename database API")

		renameDatabaseRequest := influxdb3.RenameClusterDatabaseJSONRequestBody{
			Name: plan.Name.ValueString(),
		}

		renameDatabaseResponse, err := r.client.RenameClusterDatabaseWithResponse(ctx, r.accountID, r.clusterID, state.Name.ValueString(), renameDatabaseRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error renaming database",
				"Could not rename database, unexpected error: "+err.Error(),
			)
			return
		}

		if renameDatabaseResponse.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"Error renaming database",
				formatErrorResponse(renameDatabaseResponse, renameDatabaseResponse.StatusCode()),
			)
			return
		}

		if renameDatabaseResponse.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Error renaming database",
				formatEmptyResponse(renameDatabaseResponse, renameDatabaseResponse.StatusCode()),
			)
			return
		}

		plan.Name = types.StringValue(renameDatabaseResponse.JSON200.Name)

		// Save state after rename so that if the update call below fails,
		// Terraform still knows the database has been renamed.
		if otherFieldsChanged {
			state.Name = plan.Name
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		tflog.Debug(ctx, "Rename Database API call succeeded")
	}

	// Only call update API if non-name fields changed
	if otherFieldsChanged {
		tflog.Debug(ctx, "Non-name fields changed, calling update database API", map[string]interface{}{})

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
				formatErrorResponse(updateDatabaseResponse, updateDatabaseResponse.StatusCode()),
			)
			return
		}
		updateDatabase := updateDatabaseResponse.JSON200
		if updateDatabase == nil {
			resp.Diagnostics.AddError(
				"Error updating database",
				formatEmptyResponse(updateDatabaseResponse, updateDatabaseResponse.StatusCode()),
			)
			return
		}

		// Map response body to schema and populate Computed attribute values
		plan.AccountId = types.StringValue(updateDatabase.AccountId.String())
		plan.ClusterId = types.StringValue(updateDatabase.ClusterId.String())
		plan.MaxTables = types.Int64Value(int64(updateDatabase.MaxTables))
		plan.MaxColumnsPerTable = types.Int64Value(int64(updateDatabase.MaxColumnsPerTable))
		plan.Name = types.StringValue(updateDatabase.Name)
		plan.RetentionPeriod = types.Int64Value(updateDatabase.RetentionPeriod)

		tflog.Debug(ctx, "Update Database API call succeeded")
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Database update complete")
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
			formatErrorResponse(deleteDatabasesResponse, deleteDatabasesResponse.StatusCode()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *DatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	pd, ok := newProviderData(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	r.accountID = pd.accountID
	r.client = pd.client
	r.clusterID = pd.clusterID
}

func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}
