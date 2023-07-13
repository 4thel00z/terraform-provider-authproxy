// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestTenantResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: tenantResourceExampleConfig("lidl"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tenant.test", "name", "lidl"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "tenant.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: tenantResourceExampleConfig("lidl"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tenant.test", "name", "lidl"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func tenantResourceExampleConfig(name string) string {
	return fmt.Sprintf(`
resource "tenant" "test" {
  name = %[1]q
}
`, name)
}
