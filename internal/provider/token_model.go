package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	ExpiresAt   types.String           `tfsdk:"expires_at"`
	Id          types.String           `tfsdk:"id"`
	Permissions []TokenPermissionModel `tfsdk:"permissions"`
}

// TokenPermissionModel maps InfluxDB database token permission schema data.
type TokenPermissionModel struct {
	Action   types.String `tfsdk:"action"`
	Resource types.String `tfsdk:"resource"`
}

type rfc3339Validator struct{}

func (v rfc3339Validator) Description(ctx context.Context) string {
	return "value must be a valid RFC3339 timestamp"
}

func (v rfc3339Validator) MarkdownDescription(ctx context.Context) string {
	return "value must be a valid RFC3339 timestamp"
}

func (v rfc3339Validator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	_, err := time.Parse(time.RFC3339, value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid RFC3339 Timestamp",
			fmt.Sprintf("The value must be a valid RFC3339 timestamp (e.g., 2020-01-01T00:00:00Z). Error: %s", err.Error()),
		)
		return
	}

	if strings.Contains(value, ".") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"RFC3339 Timestamp Must Not Include Subseconds",
			fmt.Sprintf("The value must be in RFC3339 format without fractional seconds (e.g., 2020-01-01T00:00:00Z), but got: %s", value),
		)
	}
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
