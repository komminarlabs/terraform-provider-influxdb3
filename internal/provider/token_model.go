package provider

import "github.com/hashicorp/terraform-plugin-framework/types"

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
