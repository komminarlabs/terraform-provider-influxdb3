output "all_tokens" {
  value     = data.influxdb3_tokens.all.tokens
  sensitive = true
}
