package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thulasirajkomminar/influxdb3-management-go/cloud"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DatabasesDataSource{}
	_ datasource.DataSourceWithConfigure = &DatabasesDataSource{}
)

// NewDatabasesDataSource returns the deprecated influxdb3_databases alias of
// the influxdb3_cloud_databases data source.
func NewDatabasesDataSource() datasource.DataSource {
	return &DatabasesDataSource{aliasedType: aliasedType{typeSuffix: "_databases", deprecated: true}}
}

// NewCloudDatabasesDataSource is a helper function to simplify the provider implementation.
func NewCloudDatabasesDataSource() datasource.DataSource {
	return &DatabasesDataSource{aliasedType: aliasedType{typeSuffix: "_cloud_databases"}}
}

// DatabasesDataSource is the data source implementation.
type DatabasesDataSource struct {
	aliasedType
	accountID influxdb3cloud.UuidV4
	client    influxdb3cloud.ClientWithResponses
	clusterID influxdb3cloud.UuidV4
}

// DatabasesDataSourceModel describes the data source data model.
type DatabasesDataSourceModel struct {
	Databases []DatabaseModel `tfsdk:"databases"`
}

// Metadata returns the data source type name.
func (d *DatabasesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + d.typeSuffix
}

// Schema defines the schema for the data source.
func (d *DatabasesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	defer d.applyDataSourceDeprecation(resp)
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
							Description: "The ID of the account that the database belongs to.",
						},
						"cluster_id": schema.StringAttribute{
							Computed:    true,
							Description: "The ID of the cluster that the database belongs to.",
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
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *DatabasesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	pd, ok := newProviderData(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !pd.requireDeploymentType(d.typeName(), &resp.Diagnostics, typeCloud) {
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
			"Error getting databases",
			err.Error(),
		)
		return
	}

	if readDatabasesResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error getting databases",
			formatErrorResponse(readDatabasesResponse, readDatabasesResponse.StatusCode()),
		)
		return
	}

	if readDatabasesResponse.JSON200 == nil {
		resp.Diagnostics.AddError(
			"Error getting databases",
			formatEmptyResponse(readDatabasesResponse, readDatabasesResponse.StatusCode()),
		)
		return
	}

	// Map response body to model
	for _, database := range *readDatabasesResponse.JSON200 {
		partitionTemplate, err := getPartitionTemplate(database.PartitionTemplate)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error getting databases",
				err.Error(),
			)
			return
		}

		databaseState := DatabaseModel{
			AccountId:          types.StringValue(database.AccountId.String()),
			ClusterId:          types.StringValue(database.ClusterId.String()),
			MaxTables:          types.Int64Value(int64(database.MaxTables)),
			MaxColumnsPerTable: types.Int64Value(int64(database.MaxColumnsPerTable)),
			Name:               types.StringValue(database.Name),
			PartitionTemplate:  partitionTemplate,
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
