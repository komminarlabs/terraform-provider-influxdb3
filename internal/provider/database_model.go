package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

// bucketPartitionValue is the JSON payload expected in the value of a
// bucket partition template part.
type bucketPartitionValue struct {
	NumberOfBuckets *int32  `json:"numberOfBuckets,omitempty"`
	TagName         *string `json:"tagName,omitempty"`
}

// buildPartitionTemplate converts partition template parts from the Terraform
// model into the API request representation.
func buildPartitionTemplate(parts []DatabasePartitionTemplateModel) ([]influxdb3.ClusterDatabasePartitionTemplatePart, error) {
	partitionTemplates := []influxdb3.ClusterDatabasePartitionTemplatePart{}
	for _, pt := range parts {
		t := influxdb3.ClusterDatabasePartitionTemplatePart{}
		switch pt.Type.ValueString() {
		case "time":
			timeTemplate := influxdb3.ClusterDatabasePartitionTemplatePartTimeFormat{
				Type:  (*influxdb3.ClusterDatabasePartitionTemplatePartTimeFormatType)(pt.Type.ValueStringPointer()),
				Value: pt.Value.ValueStringPointer(),
			}

			if err := t.MergeClusterDatabasePartitionTemplatePartTimeFormat(timeTemplate); err != nil {
				return nil, fmt.Errorf("failed to merge time template: %w", err)
			}
		case "tag":
			tagTemplate := influxdb3.ClusterDatabasePartitionTemplatePartTagValue{
				Type:  (*influxdb3.ClusterDatabasePartitionTemplatePartTagValueType)(pt.Type.ValueStringPointer()),
				Value: pt.Value.ValueStringPointer(),
			}

			if err := t.MergeClusterDatabasePartitionTemplatePartTagValue(tagTemplate); err != nil {
				return nil, fmt.Errorf("failed to merge tag template: %w", err)
			}
		case "bucket":
			var encodedJSONData struct {
				NumberOfBuckets *int32  `json:"numberOfBuckets,omitempty"`
				TagName         *string `json:"tagName,omitempty"`
			}
			if err := json.Unmarshal([]byte(pt.Value.ValueString()), &encodedJSONData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal JSON data: %w", err)
			}

			bucketTemplate := influxdb3.ClusterDatabasePartitionTemplatePartBucket{
				Type:  (*influxdb3.ClusterDatabasePartitionTemplatePartBucketType)(pt.Type.ValueStringPointer()),
				Value: &encodedJSONData,
			}

			if err := t.MergeClusterDatabasePartitionTemplatePartBucket(bucketTemplate); err != nil {
				return nil, fmt.Errorf("failed to merge bucket template: %w", err)
			}
		}
		partitionTemplates = append(partitionTemplates, t)
	}
	return partitionTemplates, nil
}

// partitionTemplateValidator validates bucket partition template parts at
// plan time: the value must be a JSON object such as
// {"numberOfBuckets": 10, "tagName": "tag"}.
type partitionTemplateValidator struct{}

func (v partitionTemplateValidator) Description(ctx context.Context) string {
	return "bucket partition template parts must have a JSON encoded value"
}

func (v partitionTemplateValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v partitionTemplateValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var parts []DatabasePartitionTemplateModel
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &parts, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	for i, pt := range parts {
		if pt.Type.ValueString() != "bucket" || pt.Value.IsNull() || pt.Value.IsUnknown() {
			continue
		}

		var bucketValue bucketPartitionValue
		if err := json.Unmarshal([]byte(pt.Value.ValueString()), &bucketValue); err != nil {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i).AtName("value"),
				"Invalid Bucket Partition Template Value",
				fmt.Sprintf("The value of a bucket partition template part must be a JSON object with numberOfBuckets and tagName keys, for example {\"numberOfBuckets\": 10, \"tagName\": \"tag\"}. Error: %s", err.Error()),
			)
		}
	}
}

func getDatabaseByName(databases influxdb3.GetClusterDatabasesResponse, name string) (*DatabaseModel, error) {
	if databases.JSON200 == nil {
		return nil, fmt.Errorf("the InfluxDB API returned a success status code without a valid JSON body")
	}

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
