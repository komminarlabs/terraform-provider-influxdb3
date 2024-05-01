package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTokenDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccTokenDataSourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.influxdb3_token.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("data.influxdb3_token.test", "description", "Access test bucket"),
				),
			},
		},
	})
}

func testAccTokenDataSourceConfig() string {
	return `
resource "influxdb3_bucket" "test" {
	name = "test"
	org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
  }

resource "influxdb3_token" "test" {
	org_id      = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
	description = "Access test bucket"
  
	permissions = [{
	  action = "read"
	  resource = {
		id     = influxdb3_bucket.test.id
		org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
		type   = "buckets"
	  }
	  },
	  {
		action = "write"
		resource = {
		  id     = influxdb3_bucket.test.id
		  org_id = "` + os.Getenv("INFLUXDB_ORG_ID") + `"
		  type   = "buckets"
		}
	}]
  }

  data "influxdb3_token" "test" {
	id = influxdb3_token.test.id
  }
`
}
