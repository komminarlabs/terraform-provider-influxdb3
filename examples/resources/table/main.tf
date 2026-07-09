resource "influxdb3_database" "signals" {
  name             = "signals"
  retention_period = 604800
}

resource "influxdb3_table" "sensors" {
  database_name = influxdb3_database.signals.name
  name          = "sensors"

  partition_template = [
    {
      type  = "tag"
      value = "line"
    },
    {
      type  = "time"
      value = "%Y-%m-%d"
    },
    {
      type = "bucket"
      value = jsonencode({
        "numberOfBuckets" : 10,
        "tagName" : "temperature"
      })
    },
  ]
}
