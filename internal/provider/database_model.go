package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thulasirajkomminar/influxdb3-management-go"
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
	Type  types.String `json:"type" tfsdk:"type"`
	Value types.String `json:"value" tfsdk:"value"`
}

// GetAttrType returns the attribute type for the DatabasePartitionTemplateModel.
func (d DatabasePartitionTemplateModel) GetAttrType() attr.Type {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"value": types.StringType,
	}}
}

func getDatabaseByName(databases influxdb3.GetClusterDatabasesResponse, name string) (*DatabaseModel, error) {
	for _, database := range *databases.JSON200 {
		if database.Name == name {
			partitionTemplate, err := getPartitionTemplate(database.PartitionTemplate)
			if err != nil {
				return nil, err
			}

			db := DatabaseModel{
				AccountId:          types.StringValue(database.AccountId.String()),
				ClusterId:          types.StringValue(database.ClusterId.String()),
				Name:               types.StringValue(database.Name),
				MaxTables:          types.Int64Value(int64(database.MaxTables)),
				MaxColumnsPerTable: types.Int64Value(int64(database.MaxColumnsPerTable)),
				PartitionTemplate:  partitionTemplate,
				RetentionPeriod:    types.Int64Value(database.RetentionPeriod),
			}
			return &db, nil
		}
	}
	return nil, nil
}

func getPartitionTemplate(partitionTemplates *influxdb3.ClusterDatabasePartitionTemplate) ([]DatabasePartitionTemplateModel, error) {
	if partitionTemplates == nil {
		return nil, nil
	}

	partitionTemplateModels := make([]DatabasePartitionTemplateModel, 0)
	for _, v := range *partitionTemplates {
		partitionTemplate := make(map[string]any)
		b, err := v.MarshalJSON()
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(b, &partitionTemplate)
		if err != nil {
			return nil, err
		}

		if partitionType, ok := partitionTemplate["type"].(string); ok && (partitionType == "time" || partitionType == "tag") {
			if partitionValue, ok := partitionTemplate["value"].(string); ok {
				partitionTemplateModels = append(partitionTemplateModels, DatabasePartitionTemplateModel{
					Type:  types.StringValue(partitionType),
					Value: types.StringValue(partitionValue),
				})
			}
		} else if partitionTemplate["type"] == "bucket" {
			jsonEncoded, err := json.Marshal(partitionTemplate["value"])
			if err != nil {
				return nil, err
			}

			partitionTemplateModels = append(partitionTemplateModels, DatabasePartitionTemplateModel{
				Type:  types.StringValue(partitionType),
				Value: types.StringValue(string(jsonEncoded)),
			})
		}
	}
	return partitionTemplateModels, nil
}
