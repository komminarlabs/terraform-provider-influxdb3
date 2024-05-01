package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/terraform-provider-influxdb3/internal/sdk/influxdb3"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DatabaseDataSource{}
	_ datasource.DataSourceWithConfigure = &DatabaseDataSource{}
)

// NewDatabaseDataSource is a helper function to simplify the provider implementation.
func NewDatabaseDataSource() datasource.DataSource {
	return &DatabaseDataSource{}
}

// DatabasesDataSource is the data source implementation.
type DatabaseDataSource struct {
	client influxdb3.Client
}

// Metadata returns the data source type name.
func (d *DatabaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database"
}

// Schema defines the schema for the data source.
func (d *DatabaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Retrieves a database. Use this data source to retrieve information for a specific database.",

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
				Description: "The name of the cluster database. The Length should be between `[ 1 .. 64 ]` characters.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"max_tables": schema.Int64Attribute{
				Computed:    true,
				Description: "The maximum number of tables for the cluster database. The default is `500`",
			},
			"max_columns_per_table": schema.Int64Attribute{
				Computed:    true,
				Description: "The maximum number of columns per table for the cluster database. The default is `200`",
			},
			"retention_period": schema.Int64Attribute{
				Computed:    true,
				Description: "The retention period of the cluster database in nanoseconds. The default is `0`. If the retention period is not set or is set to `0`, the database will have infinite retention.",
			},
			"partition_template": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "A [template](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) for partitioning a cluster database.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of the template part.",
						},
						"value": schema.StringAttribute{
							Computed:    true,
							Description: "The value of the template part.",
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DatabaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *DatabaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DatabaseModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	databaseName := state.Name
	if databaseName.IsNull() {
		resp.Diagnostics.AddError(
			"Name is empty",
			"Must set name",
		)
		return
	}

	readDatabase, err := d.client.DatabaseAPI().GetDatabaseByName(ctx, databaseName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Database not found",
			err.Error(),
		)
		return
	}

	// Map response body to model
	state.AccountId = types.StringValue(readDatabase.AccountId)
	state.ClusterId = types.StringValue(readDatabase.ClusterId)
	state.Name = types.StringValue(readDatabase.Name)
	state.MaxTables = types.Int64Value(readDatabase.MaxTables)
	state.MaxColumnsPerTable = types.Int64Value(readDatabase.MaxColumnsPerTable)
	state.RetentionPeriod = types.Int64Value(readDatabase.RetentionPeriod)
	state.PartitionTemplate = getPartitionTemplate(readDatabase.PartitionTemplate)

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
