package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DatabaseModel maps InfluxDB database schema data.
type DatabaseModel struct {
	AccountId          types.String                     `tfsdk:"account_id"`
	ClusterId          types.String                     `tfsdk:"cluster_id"`
	Name               types.String                     `tfsdk:"name"`
	MaxTables          types.Int64                      `tfsdk:"max_tables"`
	MaxColumnsPerTable types.Int64                      `tfsdk:"max_columns_per_table"`
	RetentionPeriod    types.Int64                      `tfsdk:"retention_period"`
	PartitionTemplate  []DatabasePartitionTemplateModel `tfsdk:"partition_template"`
}

// DatabasePartitionTemplateModel maps InfluxDB database partition template schema data.
type DatabasePartitionTemplateModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

// GetAttrType returns the attribute type for the DatabasePartitionTemplateModel.
func (d DatabasePartitionTemplateModel) GetAttrType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	}}
}
