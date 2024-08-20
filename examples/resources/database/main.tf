resource "influxdb3_database" "signals" {
  name             = "signals"
  retention_period = 604800

  partition_template = [
    {
      type  = "tag"
      value = "line"
    },
    {
      type  = "tag"
      value = "station"
    },
    {
      type  = "time"
      value = "%Y-%m-%d"
    },
    {
      type = "bucket"
      value = jsonencode({
        "tagName" : "temperature",
        "numberOfBuckets" : 10
      })
    },
  ]
}
