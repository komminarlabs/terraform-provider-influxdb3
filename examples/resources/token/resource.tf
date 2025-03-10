data "influxdb3_database" "signals" {
  name = "signals"
}

resource "influxdb3_token" "signals" {
  description = "Access signals database"

  permissions = [
    {
      action   = "read"
      resource = data.influxdb3_database.signals.name
    },
    {
      action   = "write"
      resource = data.influxdb3_database.signals.name
    }
  ]
}
