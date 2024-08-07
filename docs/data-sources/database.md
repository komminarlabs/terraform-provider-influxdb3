---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "influxdb3_database Data Source - terraform-provider-influxdb3"
subcategory: ""
description: |-
  Retrieves a database. Use this data source to retrieve information for a specific database.
---

# influxdb3_database (Data Source)

Retrieves a database. Use this data source to retrieve information for a specific database.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the cluster database.

### Read-Only

- `account_id` (String) The ID of the account that the cluster belongs to.
- `cluster_id` (String) The ID of the cluster that you want to manage.
- `max_columns_per_table` (Number) The maximum number of columns per table for the cluster database.
- `max_tables` (Number) The maximum number of tables for the cluster database.
- `retention_period` (Number) The retention period of the cluster database in nanoseconds.
