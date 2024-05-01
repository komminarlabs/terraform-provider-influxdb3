package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccDatabaseResourceWithRetentionConfig("test", "test database", "0"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb3_database.test", "name", "test"),
					resource.TestCheckResourceAttr("influxdb3_database.test", "description", "test database"),
					resource.TestCheckResourceAttr("influxdb3_database.test", "retention_period", "0"),
				),
			},
			// ImportState testing
			{
				ResourceName: "influxdb3_database.test",
				ImportState:  true,
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccDatabaseResourceConfig("test-database", "test-database"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("influxdb3_database.test", "name", "test-database"),
					resource.TestCheckResourceAttr("influxdb3_database.test", "description", "test-database"),
					resource.TestCheckResourceAttr("influxdb3_database.test", "retention_period", "2592000"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccDatabaseResourceWithRetentionConfig(name string, description string, retention_period string) string {
	return fmt.Sprintf(`
resource "influxdb3_database" "test" {
  name = %[1]q
  description = %[2]q
  retention_period = %[3]q
  org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
}
`, name, description, retention_period)
}

func testAccDatabaseResourceConfig(name string, description string) string {
	return fmt.Sprintf(`
resource "influxdb3_database" "test" {
  name = %[1]q
  description = %[2]q
  org_id = "`+os.Getenv("INFLUXDB_ORG_ID")+`"
}
`, name, description)
}
