package azurerm

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
)

const k8sVersionRX = `[0-9]+\.[0-9]+\.[0-9]*`

func TestAccDataSourceAzureRMContainerOrchestratorVersions_basic(t *testing.T) {
	dataSourceName := "data.azurerm_container_orchestrator_versions.test"
	location := testLocation()
	kvrx := regexp.MustCompile(k8sVersionRX)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAzureRMContainerOrchestratorVersions_basic(location),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "versions.#"),
					resource.TestMatchResourceAttr(dataSourceName, "versions.0", kvrx),
					resource.TestCheckResourceAttrSet(dataSourceName, "latest_version"),
					resource.TestMatchResourceAttr(dataSourceName, "latest_version", kvrx),
				),
			},
		},
	})
}

func TestAccDataSourceAzureRMContainerOrchestratorVersions_filtered(t *testing.T) {
	dataSourceName := "data.azurerm_container_orchestrator_versions.test"
	location := testLocation()
	kvrx := regexp.MustCompile(k8sVersionRX)

	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceAzureRMContainerOrchestratorVersions_filtered(location),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(dataSourceName, "versions.#"),
					resource.TestMatchResourceAttr(dataSourceName, "versions.0", kvrx),
					resource.TestCheckResourceAttrSet(dataSourceName, "latest_version"),
					resource.TestMatchResourceAttr(dataSourceName, "latest_version", kvrx),
				),
			},
		},
	})
}

func testAccDataSourceAzureRMContainerOrchestratorVersions_basic(location string) string {
	return fmt.Sprintf(`
data "azurerm_container_orchestrator_versions" "test" {
  location = "%s"
}
`, location)
}

func testAccDataSourceAzureRMContainerOrchestratorVersions_filtered(location string) string {
	return fmt.Sprintf(`
data "azurerm_container_orchestrator_versions" "test" {
  location       = "%s"
  version_prefix = "1."
}
`, location)
}