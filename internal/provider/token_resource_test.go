package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccTokenResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccTokenResourceConfig("Access test bucket"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb3_token.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("influxdb3_token.test", "description", "Access test bucket"),
				),
			},
			// ImportState testing
			{
				ResourceName: "influxdb3_token.test",
				ImportState:  true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccTokenResourceConfig("RW access test bucket"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb3_token.test", "permissions.#", "2"),
					resource.TestCheckResourceAttr("influxdb3_token.test", "description", "RW access test bucket"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccTokenResourceConfig(description string) string {
	return fmt.Sprintf(`
resource "influxdb3_bucket" "test" {
	name = "test"
	org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
  }

resource "influxdb3_token" "test" {
	org_id      = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
	description = %[1]q
  
	permissions = [{
	  action = "read"
	  resource = {
		id   = influxdb3_bucket.test.id
		type = "buckets"
	  }
	  },
	  {
		action = "write"
		resource = {
		  id   = influxdb3_bucket.test.id
		  type = "buckets"
		}
	}]
  }
`, description)
}
