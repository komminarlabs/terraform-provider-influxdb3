terraform {
  required_providers {
    influxdb3 = {
      source = "thulasirajkomminar/influxdb3"
    }
  }
}

provider "influxdb3" {
  type       = "cloud" # optional, this is the default
  account_id = "11111111-2222-3333-4444-555555555555"
  cluster_id = "66666666-7777-8888-9999-000000000000"
  token      = var.influxdb_management_token
}
