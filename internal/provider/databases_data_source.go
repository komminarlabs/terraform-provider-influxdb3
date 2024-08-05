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
	_ datasource.DataSource              = &DatabasesDataSource{}
	_ datasource.DataSourceWithConfigure = &DatabasesDataSource{}
)

// NewDatabasesDataSource is a helper function to simplify the provider implementation.
func NewDatabasesDataSource() datasource.DataSource {
	return &DatabasesDataSource{}
}

// DatabasesDataSource is the data source implementation.
type DatabasesDataSource struct {
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
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
							Description: "The name of the cluster database.",
						},
						"max_tables": schema.Int64Attribute{
							Computed:    true,
							Description: "The maximum number of tables for the cluster database.",
						},
						"max_columns_per_table": schema.Int64Attribute{
							Computed:    true,
							Description: "The maximum number of columns per table for the cluster database.",
						},
						"retention_period": schema.Int64Attribute{
							Computed:    true,
							Description: "The retention period of the cluster database in nanoseconds.",
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
func (d *DatabasesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DatabasesDataSourceModel

	readDatabasesResponse, err := d.client.GetClusterDatabasesWithResponse(ctx, d.accountID, d.clusterID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting Databases",
			err.Error(),
		)
		return
	}

	if readDatabasesResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error getting Databases",
			fmt.Sprintf("Status: %s", readDatabasesResponse.Status()),
		)
		return
	}

	// Map response body to model
	for _, database := range *readDatabasesResponse.JSON200 {
		databaseState := DatabaseModel{
			AccountId:          types.StringValue(database.AccountId.String()),
			ClusterId:          types.StringValue(database.ClusterId.String()),
			Name:               types.StringValue(database.Name),
			MaxTables:          types.Int64Value(int64(database.MaxTables)),
			MaxColumnsPerTable: types.Int64Value(int64(database.MaxColumnsPerTable)),
			RetentionPeriod:    types.Int64Value(database.RetentionPeriod),
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
