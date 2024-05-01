terraform {
  required_providers {
    influxdb3 = {
      source = "komminarlabs/influxdb3"
    }
  }
}

data "influxdb3_token" "signals_token" {
  id = "7f7fa77d-b77e-77ba-7777-77cd077d0f7c"
}

output "signals_token" {
  value = data.influxdb3_token.signals_token.description
}
