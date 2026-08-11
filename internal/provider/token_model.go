package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/thulasirajkomminar/influxdb3-management-go/cloud"
)

// TokenModel maps InfluxDB database token schema data.
type TokenModel struct {
	AccessToken types.String           `tfsdk:"access_token"`
	AccountId   types.String           `tfsdk:"account_id"`
	CreatedAt   types.String           `tfsdk:"created_at"`
	ClusterId   types.String           `tfsdk:"cluster_id"`
	Description types.String           `tfsdk:"description"`
	ExpiresAt   timetypes.RFC3339      `tfsdk:"expires_at"`
	Id          types.String           `tfsdk:"id"`
	Permissions []TokenPermissionModel `tfsdk:"permissions"`
	RevokedAt   timetypes.RFC3339      `tfsdk:"revoked_at"`
}

// TokenPermissionModel maps InfluxDB database token permission schema data.
type TokenPermissionModel struct {
	Action   types.String `tfsdk:"action"`
	Resource types.String `tfsdk:"resource"`
}

// rfc3339NoSubsecondsValidator ensures a timestamp has no fractional seconds.
// The RFC3339 format itself is validated by the timetypes.RFC3339 custom type.
type rfc3339NoSubsecondsValidator struct{}

func (v rfc3339NoSubsecondsValidator) Description(ctx context.Context) string {
	return "value must be an RFC3339 timestamp without fractional seconds"
}

func (v rfc3339NoSubsecondsValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v rfc3339NoSubsecondsValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	if strings.Contains(value, ".") {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"RFC3339 Timestamp Must Not Include Subseconds",
			fmt.Sprintf("The value must be in RFC3339 format without fractional seconds (e.g., 2020-01-01T00:00:00Z), but got: %s", value),
		)
	}
}

func buildPermissions(permissions []TokenPermissionModel) ([]influxdb3cloud.DatabaseTokenPermission, error) {
	var permissionsRequest []influxdb3cloud.DatabaseTokenPermission
	for _, permission := range permissions {
		resource := influxdb3cloud.DatabaseTokenPermissionResource{}

		var err error
		if permission.Resource.ValueString() == string(influxdb3cloud.Asterisk) {
			err = resource.FromDatabaseTokenResourceAllDatabases(influxdb3cloud.Asterisk)
		} else {
			err = resource.FromClusterDatabaseName(permission.Resource.ValueString())
		}
		if err != nil {
			return nil, err
		}

		permissionsRequest = append(permissionsRequest, influxdb3cloud.DatabaseTokenPermission{
			Action:   permission.Action.ValueStringPointer(),
			Resource: &resource,
		})
	}
	return permissionsRequest, nil
}

func getPermissions(permissions []influxdb3cloud.DatabaseTokenPermission) []TokenPermissionModel {
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
