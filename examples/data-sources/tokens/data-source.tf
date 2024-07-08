terraform {
  required_providers {
    influxdb3 = {
      source = "komminarlabs/influxdb3"
    }
  }
}

data "influxdb3_tokens" "all" {}

output "all_tokens" {
  value     = data.influxdb3_tokens.all.tokens
  sensitive = true
}
