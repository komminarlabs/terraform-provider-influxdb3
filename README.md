# terraform-provider-influxdb3
Terraform provider to manage InfluxDB V3

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.20

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
      source = "komminarlabs/influxdb3"
    }
  }
}
```

Initialize the provider

```terraform
provider "influxdb3" {
  account_id = "*******"
  cluster_id = "*******"
  token      = "*******"
  url        = "https://console.influxdata.com/api/v0"
}
```

## Supported InfluxDB flavours

* InfluxDB Cloud Dedicated

## Available functionalities

### Data Sources

* `influxdb3_database`
* `influxdb3_databases`
* `influxdb3_token`
* `influxdb3_tokens`

### Resources

* `influxdb3_database`
* `influxdb3_token`

## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `make docs`.

In order to run the full suite of Acceptance tests, run `make testacc`.

*Note:* Acceptance tests create real resources, and often cost money to run.

```shell
make testacc
```
