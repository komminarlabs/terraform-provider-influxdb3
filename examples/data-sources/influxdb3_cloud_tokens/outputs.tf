output "all_tokens" {
  value     = data.influxdb3_cloud_tokens.all.tokens
  sensitive = true
}
