package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDatabaseDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccDatabaseDataSourceConfig("_monitoring"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.influxdb3_database.test", "name", "_monitoring"),
					resource.TestCheckResourceAttr("data.influxdb3_database.test", "type", "system"),
				),
			},
		},
	})
}

func testAccDatabaseDataSourceConfig(name string) string {
	return fmt.Sprintf(`
data "influxdb3_database" "test" {
	name = %[1]q
}
`, name)
}
