---
page_title: "InfluxDB 3 deployment types and resource renaming"
subcategory: "Guides"
description: |-
  How to configure the provider for the cloud, core and enterprise deployment types, and how to migrate from the deprecated influxdb3_* resource names to influxdb3_cloud_*.
---

# InfluxDB 3 deployment types and resource renaming

The provider is expanding beyond InfluxDB Cloud Dedicated to also manage self-hosted [InfluxDB 3 Core](https://www.influxdata.com/products/influxdb/) and [InfluxDB 3 Enterprise](https://www.influxdata.com/products/influxdb-3-enterprise/) servers. This guide covers the two changes that come with that:

1. A new `type` provider attribute that selects the InfluxDB 3 deployment type.
2. Renaming of the existing resources and data sources from `influxdb3_*` to `influxdb3_cloud_*`, with the old names deprecated.

**No immediate action is required.** Existing configurations keep working unchanged: the deployment type defaults to `cloud` and the old resource names remain functional until the next major version. Using an old name prints a deprecation warning during `terraform plan` and `terraform apply`.

## Deployment types

InfluxDB 3 comes in different deployment types, each with its own management API. The provider supports:

| `type` | Product | Management API |
|---|---|---|
| `cloud` (default) | [InfluxDB Cloud Dedicated](https://www.influxdata.com/products/influxdb-cloud/dedicated/) | `https://console.influxdata.com/api/v0` |
| `core` | InfluxDB 3 Core (self-hosted) | Your server, e.g. `http://localhost:8181` |
| `enterprise` | InfluxDB 3 Enterprise (self-hosted) | Your server |

### Configuring the provider for Cloud Dedicated

Nothing changes for Cloud Dedicated; `type = "cloud"` is the default and can be omitted:

```terraform
provider "influxdb3" {
  type       = "cloud" # optional, this is the default
  account_id = "11111111-2222-3333-4444-555555555555"
  cluster_id = "66666666-7777-8888-9999-000000000000"
  token      = var.influxdb_management_token
}
```

The token is a [management token](https://docs.influxdata.com/influxdb3/cloud-dedicated/admin/tokens/management/) for your Cloud Dedicated cluster.

### Configuring the provider for Core

```terraform
provider "influxdb3" {
  type  = "core"
  host  = "http://localhost:8181"
  token = var.influxdb_admin_token # optional when the server runs without authentication
}
```

The token is an admin token (`apiv3_...`), created out-of-band with `influxdb3 create token --admin`. `account_id` and `cluster_id` are Cloud Dedicated concepts and must not be set.

### Configuring the provider for Enterprise

```terraform
provider "influxdb3" {
  type  = "enterprise"
  host  = "https://influxdb.example.com:8181"
  token = var.influxdb_admin_token
}
```

### Managing multiple deployment types together

Use [provider aliases](https://developer.hashicorp.com/terraform/language/providers/configuration#alias-multiple-provider-configurations) to manage several deployments from one configuration:

```terraform
provider "influxdb3" {
  account_id = var.account_id
  cluster_id = var.cluster_id
  token      = var.management_token
}

provider "influxdb3" {
  alias = "onprem"
  type  = "core"
  host  = "http://localhost:8181"
  token = var.admin_token
}
```

### Environment variables

The deployment type can also be set with `INFLUXDB3_TYPE`, alongside the existing `INFLUXDB3_ACCOUNT_ID`, `INFLUXDB3_CLUSTER_ID`, `INFLUXDB3_HOST` and `INFLUXDB3_TOKEN` variables.

## Resource renaming

With more than one deployment type, unprefixed names like `influxdb3_database` are ambiguous: a Cloud Dedicated database and a Core database have different APIs and different attributes. Going forward, every resource name carries a deployment-type prefix. The prefix names the *lowest* deployment type that supports the resource — future `influxdb3_core_*` resources will work on both Core and Enterprise, and `influxdb3_enterprise_*` resources on Enterprise only.

The existing Cloud Dedicated resources and data sources are therefore renamed:

| Deprecated name | New name |
|---|---|
| `influxdb3_database` (resource) | `influxdb3_cloud_database` |
| `influxdb3_table` (resource) | `influxdb3_cloud_table` |
| `influxdb3_token` (resource) | `influxdb3_cloud_token` |
| `influxdb3_database` (data source) | `influxdb3_cloud_database` |
| `influxdb3_databases` (data source) | `influxdb3_cloud_databases` |
| `influxdb3_token` (data source) | `influxdb3_cloud_token` |
| `influxdb3_tokens` (data source) | `influxdb3_cloud_tokens` |

The old and new names are the same implementation with identical schemas and behavior — only the name changes.

### Timeline

* **Now:** both names work. The old names print a deprecation warning.
* **Next major version:** the old names are removed.

### Migrating resources with `moved` blocks

Terraform 1.8 and later supports moving state between resource types. Rename the resource in your configuration and add a `moved` block:

```terraform
resource "influxdb3_cloud_database" "signals" {
  name             = "signals"
  retention_period = 604800
}

moved {
  from = influxdb3_database.signals
  to   = influxdb3_cloud_database.signals
}
```

The next `terraform plan` shows the resource being moved instead of destroyed and recreated:

```console
$ terraform plan
  # influxdb3_database.signals has moved to influxdb3_cloud_database.signals

Plan: 0 to add, 0 to change, 0 to destroy.
```

After applying, the `moved` block can be kept (it is a no-op) or removed.

~> `terraform state mv` does **not** support moving between different resource types; use a `moved` block as shown above.

### Migrating data sources

Data sources hold no state, so simply rename them in the configuration, e.g. `data "influxdb3_databases"` becomes `data "influxdb3_cloud_databases"`, and update any references.

## Frequently asked questions

**Do I have to migrate now?** No. The old names keep working until the next major version. Migrating early just silences the deprecation warnings.

**Does the move recreate my databases/tables/tokens?** No. A `moved` block is a pure state operation; no API calls are made and no infrastructure changes.

**Which resources work with `type = "core"` or `type = "enterprise"`?** The `influxdb3_cloud_*` (and deprecated `influxdb3_*`) resources require `type = "cloud"` and fail with a clear error otherwise. Resources for Core and Enterprise (`influxdb3_core_*`, `influxdb3_enterprise_*`) are being added in upcoming releases.
