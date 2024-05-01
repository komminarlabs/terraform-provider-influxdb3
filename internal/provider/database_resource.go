package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/terraform-provider-influxdb3/internal/sdk/influxdb3"
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
	client influxdb3.Client
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
				Description: "The name of the cluster database. The Length should be between `[ 1 .. 64 ]` characters. **Note:** After a database is deleted, you cannot [reuse](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/delete/#cannot-reuse-database-names) the same name for a new database.",
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
			"partition_template": schema.ListNestedAttribute{
				Computed:            true,
				Optional:            true,
				Default:             listdefault.StaticValue(types.ListValueMust(DatabasePartitionTemplateModel{}.GetAttrType(), []attr.Value{})),
				MarkdownDescription: "A [template](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) for partitioning a cluster database. API does not support updating partition template, so updating this will force resource replacement.",
				Validators: []validator.List{
					listvalidator.UniqueValues(),
				},
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
					listplanmodifier.RequiresReplace(), // API does not support updating partition template
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Required:    true,
							Description: "The type of the template part.",
						},
						"value": schema.StringAttribute{
							Required:    true,
							Description: "The value of the template part.",
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
	partitions := []influxdb3.PartitionTemplate{}
	for _, partitionData := range plan.PartitionTemplate {
		permission := influxdb3.PartitionTemplate{
			Type:  partitionData.Type.ValueString(),
			Value: partitionData.Value.ValueString(),
		}
		partitions = append(partitions, permission)
	}

	createDatabase := influxdb3.DatabaseParams{
		Name:               plan.Name.ValueString(),
		MaxTables:          int(plan.MaxTables.ValueInt64()),
		MaxColumnsPerTable: int(plan.MaxColumnsPerTable.ValueInt64()),
		RetentionPeriod:    plan.RetentionPeriod.ValueInt64(),
		PartitionTemplate:  partitions,
	}

	apiResponse, err := r.client.DatabaseAPI().CreateDatabase(ctx, &createDatabase)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating database",
			"Could not create database, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.AccountId = types.StringValue(apiResponse.AccountId)
	plan.ClusterId = types.StringValue(apiResponse.ClusterId)
	plan.Name = types.StringValue(apiResponse.Name)
	plan.MaxTables = types.Int64Value(apiResponse.MaxTables)
	plan.MaxColumnsPerTable = types.Int64Value(apiResponse.MaxColumnsPerTable)
	plan.RetentionPeriod = types.Int64Value(apiResponse.RetentionPeriod)
	plan.PartitionTemplate = getPartitionTemplate(apiResponse.PartitionTemplate)

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
	readDatabase, err := r.client.DatabaseAPI().GetDatabaseByName(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Database not found",
			err.Error(),
		)
		return
	}

	// Overwrite items with refreshed state
	state.AccountId = types.StringValue(readDatabase.AccountId)
	state.ClusterId = types.StringValue(readDatabase.ClusterId)
	state.Name = types.StringValue(readDatabase.Name)
	state.MaxTables = types.Int64Value(readDatabase.MaxTables)
	state.MaxColumnsPerTable = types.Int64Value(readDatabase.MaxColumnsPerTable)
	state.RetentionPeriod = types.Int64Value(readDatabase.RetentionPeriod)
	state.PartitionTemplate = getPartitionTemplate(readDatabase.PartitionTemplate)

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
	updateDatabase := influxdb3.DatabaseParams{
		Name:               plan.Name.ValueString(),
		MaxTables:          int(plan.MaxTables.ValueInt64()),
		MaxColumnsPerTable: int(plan.MaxColumnsPerTable.ValueInt64()),
		RetentionPeriod:    plan.RetentionPeriod.ValueInt64(),
	}

	// Update existing database
	apiResponse, err := r.client.DatabaseAPI().UpdateDatabase(ctx, &updateDatabase)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating database",
			"Could not update database, unexpected error: "+err.Error(),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.AccountId = types.StringValue(apiResponse.AccountId)
	plan.ClusterId = types.StringValue(apiResponse.ClusterId)
	plan.Name = types.StringValue(apiResponse.Name)
	plan.MaxTables = types.Int64Value(apiResponse.MaxTables)
	plan.MaxColumnsPerTable = types.Int64Value(apiResponse.MaxColumnsPerTable)
	plan.RetentionPeriod = types.Int64Value(apiResponse.RetentionPeriod)

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
	err := r.client.DatabaseAPI().DeleteDatabase(ctx, state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting database",
			"Could not delete database, unexpected error: "+err.Error(),
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

func (r *DatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func getPartitionTemplate(partitionTemplate []influxdb3.PartitionTemplate) []DatabasePartitionTemplateModel {
	partitions := []DatabasePartitionTemplateModel{}
	for _, partitionData := range partitionTemplate {
		partition := DatabasePartitionTemplateModel{
			Type:  types.StringValue(partitionData.Type),
			Value: types.StringValue(partitionData.Value),
		}
		partitions = append(partitions, partition)
	}
	return partitions
}
