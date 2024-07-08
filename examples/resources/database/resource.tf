terraform {
  required_providers {
    influxdb3 = {
      source = "komminarlabs/influxdb3"
    }
  }
}

provider "influxdb3" {}

resource "influxdb3_database" "signals" {
  name             = "signals"
  retention_period = 604800
}

output "signals_database" {
  value = influxdb3_database.signals
}
