package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/terraform-provider-influxdb3/internal/sdk/influxdb3"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DatabasesDataSource{}
	_ datasource.DataSourceWithConfigure = &DatabasesDataSource{}
)

// NewDatabasesDataSource is a helper function to simplify the provider implementation.
func NewDatabasesDataSource() datasource.DataSource {
	return &DatabasesDataSource{}
}

// DatabasesDataSource is the data source implementation.
type DatabasesDataSource struct {
	client influxdb3.Client
}

// DatabasesDataSourceModel describes the data source data model.
type DatabasesDataSourceModel struct {
	Databases []DatabaseModel `tfsdk:"databases"`
}

// Metadata returns the data source type name.
func (d *DatabasesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_databases"
}

// Schema defines the schema for the data source.
func (d *DatabasesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		Description: "Gets all databases for a cluster.",

		Attributes: map[string]schema.Attribute{
			"databases": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
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
							Computed:    true,
							Description: "The name of the cluster database. The Length should be between `[ 1 .. 64 ]` characters.",
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DatabasesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *DatabasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DatabasesDataSourceModel

	readDatabases, err := d.client.DatabaseAPI().GetDatabases(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to list databases",
			err.Error(),
		)
		return
	}

	// Map response body to model
	for _, database := range readDatabases {
		var partitionTemplateState []DatabasePartitionTemplateModel
		for _, permissionData := range database.PartitionTemplate {
			partition := DatabasePartitionTemplateModel{
				Type:  types.StringValue(permissionData.Type),
				Value: types.StringValue(permissionData.Value),
			}
			partitionTemplateState = append(partitionTemplateState, partition)
		}

		databaseState := DatabaseModel{
			AccountId:          types.StringValue(database.AccountId),
			ClusterId:          types.StringValue(database.ClusterId),
			Name:               types.StringValue(database.Name),
			MaxTables:          types.Int64Value(database.MaxTables),
			MaxColumnsPerTable: types.Int64Value(database.MaxColumnsPerTable),
			RetentionPeriod:    types.Int64Value(database.RetentionPeriod),
			PartitionTemplate:  partitionTemplateState,
		}
		state.Databases = append(state.Databases, databaseState)
	}

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
