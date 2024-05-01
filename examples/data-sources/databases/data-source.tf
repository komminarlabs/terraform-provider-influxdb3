terraform {
  required_providers {
    influxdb3 = {
      source = "komminarlabs/influxdb3"
    }
  }
}

data "influxdb3_databases" "all" {}

output "all_databases" {
  value = data.influxdb3_databases.all
}
