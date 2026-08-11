package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// providerAddressSuffix identifies this provider in moved-block requests, so
// state is only accepted from resources managed by this provider.
const providerAddressSuffix = "thulasirajkomminar/influxdb3"

// aliasedType is embedded in every resource and data source so a single
// implementation can be registered under two type names: the canonical
// influxdb3_cloud_* name and the deprecated unprefixed alias.
type aliasedType struct {
	typeSuffix string
	deprecated bool
}

// typeName returns the full Terraform type name of this registration.
func (a aliasedType) typeName() string {
	return "influxdb3" + a.typeSuffix
}

// legacyTypeName returns the deprecated unprefixed name for a canonical
// influxdb3_cloud_* name (and the name itself for a legacy registration).
func (a aliasedType) legacyTypeName() string {
	return strings.Replace(a.typeName(), "_cloud_", "_", 1)
}

// canonicalTypeName returns the influxdb3_cloud_* name for a legacy
// registration (and the name itself for a canonical registration).
func (a aliasedType) canonicalTypeName() string {
	return strings.Replace(a.legacyTypeName(), "influxdb3_", "influxdb3_cloud_", 1)
}

// applyResourceDeprecation marks the schema of a deprecated alias
// registration. Deferred at the top of every resource Schema method.
func (a aliasedType) applyResourceDeprecation(resp *resource.SchemaResponse) {
	if a.deprecated {
		resp.Schema.DeprecationMessage = deprecatedAliasMessage(a.typeName(), a.canonicalTypeName(), true)
	}
}

// applyDataSourceDeprecation marks the schema of a deprecated alias
// registration. Deferred at the top of every data source Schema method.
func (a aliasedType) applyDataSourceDeprecation(resp *datasource.SchemaResponse) {
	if a.deprecated {
		resp.Schema.DeprecationMessage = deprecatedAliasMessage(a.typeName(), a.canonicalTypeName(), false)
	}
}

// deprecatedAliasMessage returns the DeprecationMessage set on the schema of
// the legacy unprefixed resource and data source registrations.
func deprecatedAliasMessage(oldName, newName string, isResource bool) string {
	msg := "The " + oldName + " name is deprecated and will be removed in the next major version. Use " + newName + " instead"
	if isResource {
		msg += "; a moved block (Terraform 1.8+) migrates existing state without replacement"
	}
	return msg + ". See the provider's \"Deployment types and resource renaming\" guide."
}

// requireDeploymentType verifies during resource/data source Configure that
// the provider is configured with a deployment type the given resource
// supports. All resource types are always registered with Terraform, so this
// is where a mismatch (for example a cloud-only resource with type = "core")
// is reported.
func (pd providerData) requireDeploymentType(typeName string, diags *diag.Diagnostics, supported ...string) bool {
	for _, s := range supported {
		if pd.deploymentType == s {
			return true
		}
	}
	diags.AddError(
		"Unsupported Deployment Type",
		typeName+" is only supported for the following InfluxDB 3 deployment type(s): "+strings.Join(supported, ", ")+", "+
			"but the provider is configured with type = \""+pd.deploymentType+"\".",
	)
	return false
}

// aliasStateMover implements moved-block support between the deprecated
// unprefixed resource names and their influxdb3_cloud_* replacements. The two
// names share a single implementation and therefore an identical schema, so
// the state is copied as-is.
func aliasStateMover(sourceTypeName string, sourceSchema schema.Schema) []resource.StateMover {
	return []resource.StateMover{
		{
			SourceSchema: &sourceSchema,
			StateMover: func(ctx context.Context, req resource.MoveStateRequest, resp *resource.MoveStateResponse) {
				if req.SourceTypeName != sourceTypeName {
					return
				}
				if req.SourceProviderAddress != "" && !strings.HasSuffix(req.SourceProviderAddress, providerAddressSuffix) {
					return
				}
				if req.SourceState == nil {
					return
				}
				resp.TargetState.Raw = req.SourceState.Raw
			},
		},
	}
}
