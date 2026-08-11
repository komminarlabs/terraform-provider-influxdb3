package provider

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/thulasirajkomminar/influxdb3-management-go/cloud"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource              = &TableResource{}
	_ resource.ResourceWithConfigure = &TableResource{}
	_ resource.ResourceWithMoveState = &TableResource{}
)

// NewTableResource returns the deprecated influxdb3_table alias of the
// influxdb3_cloud_table resource.
func NewTableResource() resource.Resource {
	return &TableResource{aliasedType: aliasedType{typeSuffix: "_table", deprecated: true}}
}

// NewCloudTableResource is a helper function to simplify the provider implementation.
func NewCloudTableResource() resource.Resource {
	return &TableResource{aliasedType: aliasedType{typeSuffix: "_cloud_table"}}
}

// TableResource defines the resource implementation.
type TableResource struct {
	aliasedType
	accountID influxdb3cloud.UuidV4
	client    influxdb3cloud.ClientWithResponses
	clusterID influxdb3cloud.UuidV4
}

// TableModel maps InfluxDB database table schema data.
type TableModel struct {
	AccountId         types.String                     `tfsdk:"account_id"`
	ClusterId         types.String                     `tfsdk:"cluster_id"`
	DatabaseName      types.String                     `tfsdk:"database_name"`
	Name              types.String                     `tfsdk:"name"`
	PartitionTemplate []DatabasePartitionTemplateModel `tfsdk:"partition_template"`
}

// Metadata returns the resource type name.
func (r *TableResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.typeSuffix
}

// Schema defines the schema for the resource.
func (r *TableResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	defer r.applyResourceDeprecation(resp)
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Creates and manages a table in a cluster database. **Note:** The InfluxDB V3 Management API does not provide an endpoint to read tables, so this resource cannot detect drift and does not support import.",

		Attributes: map[string]schema.Attribute{
			"account_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the account that the table belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the cluster that the table belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"database_name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the cluster database that the table belongs to.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the table. The Length should be between `[ 1 .. 128 ]` characters. **Note:** Renaming a table does not modify data in the table.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_\-/]*$`),
						"must start with a letter or number and only contain alphanumeric characters, underscores (_), dashes (-), and forward-slashes (/)",
					),
				},
			},
			"partition_template": schema.ListNestedAttribute{
				Computed:    true,
				Optional:    true,
				Description: "A template for [partitioning](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) the table. If not set, the table inherits the partition template of the database. **Note:** A partition template can include up to 7 total tag and tag bucket parts and only 1 time part. You can only apply a partition template when creating a table; an update will result in resource replacement.",
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
func (r *TableResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TableModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan
	createTableRequest := influxdb3cloud.CreateClusterDatabaseTableJSONRequestBody{
		Name: plan.Name.ValueString(),
	}

	if len(plan.PartitionTemplate) > 0 {
		partitionTemplates, err := buildPartitionTemplate(plan.PartitionTemplate)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error creating table partition template",
				err.Error(),
			)
			return
		}
		createTableRequest.PartitionTemplate = &partitionTemplates
	}

	createTableResponse, err := r.client.CreateClusterDatabaseTableWithResponse(ctx, r.accountID, r.clusterID, plan.DatabaseName.ValueString(), createTableRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating table",
			"Could not create table, unexpected error: "+err.Error(),
		)
		return
	}

	if createTableResponse.StatusCode() != 200 {
		resp.Diagnostics.AddError(
			"Error creating table",
			formatErrorResponse(createTableResponse, createTableResponse.StatusCode()),
		)
		return
	}
	createTable := createTableResponse.JSON200
	if createTable == nil {
		resp.Diagnostics.AddError(
			"Error creating table",
			formatEmptyResponse(createTableResponse, createTableResponse.StatusCode()),
		)
		return
	}

	// Map response body to schema and populate Computed attribute values
	plan.AccountId = types.StringValue(createTable.AccountId.String())
	plan.ClusterId = types.StringValue(createTable.ClusterId.String())
	plan.DatabaseName = types.StringValue(createTable.DatabaseName)
	plan.Name = types.StringValue(createTable.Name)

	partitionTemplate, err := getPartitionTemplate(createTable.PartitionTemplate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error getting table partition template",
			"Could not create table, unexpected error: "+err.Error(),
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
// The InfluxDB V3 Management API does not provide an endpoint to read tables,
// so the state is left as-is and drift cannot be detected.
func (r *TableResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "The Management API does not support reading tables, skipping refresh")
}

// Update updates the resource and sets the updated Terraform state on success.
// All attributes other than the table name require replacement, so an update
// is always a rename.
func (r *TableResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TableModel
	var state TableModel

	// Read Terraform plan and state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) {
		renameTableRequest := influxdb3cloud.RenameClusterDatabaseTableJSONRequestBody{
			Name: plan.Name.ValueString(),
		}

		renameTableResponse, err := r.client.RenameClusterDatabaseTableWithResponse(ctx, r.accountID, r.clusterID, state.DatabaseName.ValueString(), state.Name.ValueString(), renameTableRequest)
		if err != nil {
			resp.Diagnostics.AddError(
				"Error renaming table",
				"Could not rename table, unexpected error: "+err.Error(),
			)
			return
		}

		if renameTableResponse.StatusCode() != 200 {
			resp.Diagnostics.AddError(
				"Error renaming table",
				formatErrorResponse(renameTableResponse, renameTableResponse.StatusCode()),
			)
			return
		}

		if renameTableResponse.JSON200 == nil {
			resp.Diagnostics.AddError(
				"Error renaming table",
				formatEmptyResponse(renameTableResponse, renameTableResponse.StatusCode()),
			)
			return
		}

		plan.Name = types.StringValue(renameTableResponse.JSON200.Name)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *TableResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TableModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing table
	deleteTableResponse, err := r.client.DeleteClusterDatabaseTableWithResponse(ctx, r.accountID, r.clusterID, state.DatabaseName.ValueString(), state.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting table",
			"Could not delete table, unexpected error: "+err.Error(),
		)
		return
	}

	if deleteTableResponse.StatusCode() != 204 {
		resp.Diagnostics.AddError(
			"Error deleting table",
			formatErrorResponse(deleteTableResponse, deleteTableResponse.StatusCode()),
		)
		return
	}
}

// Configure adds the provider configured client to the resource.
func (r *TableResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	pd, ok := newProviderData(req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	if !pd.requireDeploymentType(r.typeName(), &resp.Diagnostics, typeCloud) {
		return
	}

	r.accountID = pd.accountID
	r.client = pd.client
	r.clusterID = pd.clusterID
}

// MoveState enables a moved block from the deprecated influxdb3_table
// resource to influxdb3_cloud_table.
func (r *TableResource) MoveState(ctx context.Context) []resource.StateMover {
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	return aliasStateMover(r.legacyTypeName(), schemaResp.Schema)
}
