terraform {
  required_providers {
    influxdb3 = {
      source = "thulasirajkomminar/influxdb3"
    }
  }
}

provider "influxdb3" {}
