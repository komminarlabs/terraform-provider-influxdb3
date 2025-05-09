---
page_title: "influxdb3_database Resource - terraform-provider-influxdb3"
subcategory: ""
description: |-
  Creates and manages a database.
---

# influxdb3_database (Resource)

Creates and manages a database.

## Example Usage

```terraform
resource "influxdb3_database" "signals" {
  name             = "signals"
  retention_period = 604800

  partition_template = [
    {
      type  = "tag"
      value = "line"
    },
    {
      type  = "tag"
      value = "station"
    },
    {
      type  = "time"
      value = "%Y-%m-%d"
    },
    {
      type = "bucket"
      value = jsonencode({
        "tagName" : "temperature",
        "numberOfBuckets" : 10
      })
    },
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the cluster database. The Length should be between `[ 1 .. 64 ]` characters. **Note:** Database names can't be updated. An update will result in resource replacement. After a database is deleted, you cannot [reuse](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/delete/#cannot-reuse-database-names) the same name for a new database.

### Optional

- `max_columns_per_table` (Number) The maximum number of columns per table for the cluster database. The default is `200`
- `max_tables` (Number) The maximum number of tables for the cluster database. The default is `500`
- `partition_template` (Attributes List) A template for [partitioning](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/custom-partitions/partition-templates/) a cluster database. **Note:** A partition template can include up to 7 total tag and tag bucket parts and only 1 time part. You can only apply a partition template when creating a database. You [can't update a partition template](https://docs.influxdata.com/influxdb/cloud-dedicated/admin/databases/create/#partition-templates-can-only-be-applied-on-create) on an existing database. An update will result in resource replacement. (see [below for nested schema](#nestedatt--partition_template))
- `retention_period` (Number) The retention period of the cluster database in nanoseconds. The default is `0`. If the retention period is not set or is set to `0`, the database will have infinite retention.

### Read-Only

- `account_id` (String) The ID of the account that the database belongs to.
- `cluster_id` (String) The ID of the cluster that the database belongs to.

<a id="nestedatt--partition_template"></a>
### Nested Schema for `partition_template`

Required:

- `type` (String) The type of template part. Valid values are `bucket`, `tag` or `time`.
- `value` (String) The value of template part. **Note:** For `bucket` partition template type use `jsonencode()` function to encode the value to a string.
