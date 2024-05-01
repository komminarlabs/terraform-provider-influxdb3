terraform {
  required_providers {
    influxdb3 = {
      source = "komminarlabs/influxdb3"
    }
  }
}

data "influxdb3_database" "signals" {
  name = "signals"
}

output "signals_database" {
  value = data.influxdb3_database.signals
}
