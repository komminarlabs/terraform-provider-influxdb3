package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/komminarlabs/influxdb3"
)

// DatabaseModel maps InfluxDB database schema data.
type DatabaseModel struct {
	AccountId          types.String `tfsdk:"account_id"`
	ClusterId          types.String `tfsdk:"cluster_id"`
	Name               types.String `tfsdk:"name"`
	MaxTables          types.Int64  `tfsdk:"max_tables"`
	MaxColumnsPerTable types.Int64  `tfsdk:"max_columns_per_table"`
	RetentionPeriod    types.Int64  `tfsdk:"retention_period"`
}

func getDatabaseByName(databases influxdb3.GetClusterDatabasesResponse, name string) *DatabaseModel {
	for _, database := range *databases.JSON200 {
		if database.Name == name {
			db := DatabaseModel{
				AccountId:          types.StringValue(database.AccountId.String()),
				ClusterId:          types.StringValue(database.ClusterId.String()),
				Name:               types.StringValue(database.Name),
				MaxTables:          types.Int64Value(int64(database.MaxTables)),
				MaxColumnsPerTable: types.Int64Value(int64(database.MaxColumnsPerTable)),
				RetentionPeriod:    types.Int64Value(database.RetentionPeriod),
			}
			return &db
		}
	}
	return nil
}
