# terraform-provider-influxdb3

Terraform provider to manage InfluxDB V3

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.26

## Building The Provider

1. Clone the repository
1. Enter the repository directory
1. Build the provider using the Go `install` command:

```shell
go install
```

## Adding Dependencies

This provider uses [Go modules](https://github.com/golang/go/wiki/Modules).
Please see the Go documentation for the most up to date information about using Go modules.

To add a new dependency `github.com/author/dependency` to your Terraform provider:

```shell
go get github.com/author/dependency
go mod tidy
```

Then commit the changes to `go.mod` and `go.sum`.

## Using the provider

Add the below code to your configuration.

```terraform
terraform {
  required_providers {
    influxdb3 = {
      source = "thulasirajkomminar/influxdb3"
    }
  }
}
```

Initialize the provider for the InfluxDB 3 deployment type you manage.

For [InfluxDB Cloud Dedicated](https://www.influxdata.com/products/influxdb-cloud/dedicated/) (the default, `type = "cloud"`):

```terraform
provider "influxdb3" {
  account_id = "*******"
  cluster_id = "*******"
  token      = "*******" # management token
}
```

For self-hosted InfluxDB 3 Core or Enterprise:

```terraform
provider "influxdb3" {
  type  = "core" # or "enterprise"
  host  = "http://localhost:8181"
  token = "apiv3_*******" # admin token; optional when the server runs without authentication
}
```

All provider configuration values can also be set with environment variables: `INFLUXDB3_TYPE`, `INFLUXDB3_ACCOUNT_ID`, `INFLUXDB3_CLUSTER_ID`, `INFLUXDB3_TOKEN` and `INFLUXDB3_HOST` (for `type = "cloud"` the host defaults to `https://console.influxdata.com`).

## Supported InfluxDB 3 deployment types

- `cloud` — [InfluxDB Cloud Dedicated](https://www.influxdata.com/products/influxdb-cloud/dedicated/)
- `core` — InfluxDB 3 Core (self-hosted); resources coming in upcoming releases
- `enterprise` — InfluxDB 3 Enterprise (self-hosted); resources coming in upcoming releases

## Available functionalities

Resource names carry the deployment type as a prefix. The unprefixed `influxdb3_*` names still work but are deprecated and will be removed in the next major version — see the [deployment types and resource renaming guide](docs/guides/deployment-types-and-resource-renaming.md) for migration steps using `moved` blocks.

### Data Sources

- `influxdb3_cloud_database` (deprecated alias: `influxdb3_database`)
- `influxdb3_cloud_databases` (deprecated alias: `influxdb3_databases`)
- `influxdb3_cloud_token` (deprecated alias: `influxdb3_token`)
- `influxdb3_cloud_tokens` (deprecated alias: `influxdb3_tokens`)

### Resources

- `influxdb3_cloud_database` (deprecated alias: `influxdb3_database`)
- `influxdb3_cloud_table` (deprecated alias: `influxdb3_table`)
- `influxdb3_cloud_token` (deprecated alias: `influxdb3_token`)

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make docs`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
