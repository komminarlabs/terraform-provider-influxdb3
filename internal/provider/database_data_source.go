package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/komminarlabs/influxdb3"
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
	accountID influxdb3.UuidV4
	client    influxdb3.ClientWithResponses
	clusterID influxdb3.UuidV4
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
			"partition_template": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The template partitioning of the cluster database.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:    true,
							Description: "The type of template part.",
						},
						"value": schema.StringAttribute{
							Computed:    true,
							Description: "The value of template part.",
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

	readDatabasesResponse, err := d.client.GetClusterDatabasesWithResponse(ctx, d.accountID, d.clusterID)
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
	readDatabase, err := getDatabaseByName(*readDatabasesResponse, databaseName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting database",
			"Unexpected error: "+err.Error(),
		)
		return
	}
	if readDatabase == nil {
		resp.Diagnostics.AddError(
			"Database not found",
			fmt.Sprintf("Database with name %s not found", databaseName.ValueString()),
		)
		return
	}

	// Map response body to model
	state = *readDatabase

	// Set state
	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
