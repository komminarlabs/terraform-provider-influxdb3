## [1.2.0] - 2024-08-21

### Added:

* Added [partition_template](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) support. **Note:** Database and table partitions can only be defined on create. You [cannot update the partition strategy](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/create/#partition-templates-can-only-be-applied-on-create) of a database or table after it has been created. An update will result in resource replacement. 
  
## [1.0.1] - 2024-08-13

### Updated:

* Fixed unsafe pointer bug in token resource during create and update
* Set require replacement attribute in database name as the api does not support [updating a database](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/update/#database-names-cant-be-updated)
  
## [1.0.0] - 2024-08-05

### Added:

* Replaced internal management API client library with a [client library](https://github.com/komminarlabs/influxdb3) that was generated from OpenAPI spec.

### Removed:

* Removed the [partition_template](https://registry.terraform.io/providers/komminarlabs/influxdb3/latest/docs/resources/database#partition_template) support and will be implemented in the future version. This is because the current version did not support all types of the `partition_template` and in the future versions this will be fully implemented. 
  
## [0.1.0] - 2024-05-01

### Added:

* **New Data Source:** `influxdb3_database`
* **New Data Source:** `influxdb3_databases`
* **New Data Source:** `influxdb3_token`
* **New Data Source:** `influxdb3_tokens`

* **New Resource:** `influxdb3_database`
* **New Resource:** `influxdb3_token`
