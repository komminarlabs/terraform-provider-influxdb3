package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thulasirajkomminar/influxdb3-management-go"
)

// TokenModel maps InfluxDB database token schema data.
type TokenModel struct {
	AccessToken types.String           `tfsdk:"access_token"`
	AccountId   types.String           `tfsdk:"account_id"`
	CreatedAt   types.String           `tfsdk:"created_at"`
	ClusterId   types.String           `tfsdk:"cluster_id"`
	Description types.String           `tfsdk:"description"`
	Id          types.String           `tfsdk:"id"`
	Permissions []TokenPermissionModel `tfsdk:"permissions"`
}

// TokenPermissionModel maps InfluxDB database token permission schema data.
type TokenPermissionModel struct {
	Action   types.String `tfsdk:"action"`
	Resource types.String `tfsdk:"resource"`
}

func getPermissions(permissions []influxdb3.DatabaseTokenPermission) []TokenPermissionModel {
	permissionsState := []TokenPermissionModel{}
	for _, permission := range permissions {
		resource, _ := permission.Resource.AsClusterDatabaseName()
		permissionState := TokenPermissionModel{
			Action:   types.StringPointerValue(permission.Action),
			Resource: types.StringValue(resource),
		}
		permissionsState = append(permissionsState, permissionState)
	}
	return permissionsState
}
