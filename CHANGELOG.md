# Changelog

All notable changes to this project will automatically be documented in this file.

The format is based on vKeep a Changelog(https://keepachangelog.com/en/1.0.0/),
and this project adheres to vSemantic Versioning(https://semver.org/spec/v2.0.0.html).

## v1.5.0 - 2025-11-05

### What's Changed

* Added new `expires_at` attribute in token resource and data source to set and get the expiration time of the token.

### Fixed

* Fixed the `created_at` attributes in token resource and data source to use RFC3339Nano format for better precision.

## v1.4.0 - 2025-10-30

### What's Changed

* Moved the repo from organization to personal account.

> [!Important]
>
> The older versions of the provider are still available under the `komminarlabs/influxdb3` namespace on the Terraform Registry. But the new versions (`v1.4.0` and above) will be available under the `thulasirajkomminar/influxdb3` namespace. Please update your provider source accordingly in your Terraform configurations.

## v1.3.0 - 2025-03-11

### What's Changed

* Removed the `url`(`INFLUXDB3_URL`) attribute from the provider configuration and environment variables. The URL is now constructed using predefined constants.
* Updated dependencies in `go.mod` to newer versions, including updates to go version and several hashicorp packages.
* Added error handling for `json.Unmarshal` and `MergeClusterDatabasePartitionTemplatePart` methods to ensure proper error reporting and handling. (`internal/provider/database_model.go`).
* Updated TF provider docs.

## v1.2.3 - 2024-09-26

### Fixed

* Fixed error handling.

## v1.2.2 - 2024-09-12

### Added

* Added [retry mechanism](github.com/hashicorp/go-retryablehttp) to retry InfluxDB v3 API request at least 3 times before erroring.

## v1.2.1 - 2024-08-21

### Fixed

* Fixed docs.
  
## v1.2.0 - 2024-08-21

### Added

* Added [partition_template](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) support. **Note:** Database and table partitions can only be defined on create. You [cannot update the partition strategy](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/create/#partition-templates-can-only-be-applied-on-create) of a database or table after it has been created. An update will result in resource replacement. 
  
## v1.0.1 - 2024-08-13

### Updated

* Fixed unsafe pointer bug in token resource during create and update
* Set require replacement attribute in database name as the api does not support [updating a database](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/update/#database-names-cant-be-updated)
  
## v1.0.0 - 2024-08-05

### Added

* Replaced internal management API client library with a [client library](https://github.com/komminarlabs/influxdb3) that was generated from OpenAPI spec.

### Removed

* Removed the [partition_template](https://registry.terraform.io/providers/komminarlabs/influxdb3/latest/docs/resources/database#partition_template) support and will be implemented in the future version. This is because the current version did not support all types of the `partition_template` and in the future versions this will be fully implemented.

## v0.1.0 - 2024-05-01

### Added

* **New Data Source:** `influxdb3_database`
* **New Data Source:** `influxdb3_databases`
* **New Data Source:** `influxdb3_token`
* **New Data Source:** `influxdb3_tokens`

* **New Resource:** `influxdb3_database`
* **New Resource:** `influxdb3_token`
