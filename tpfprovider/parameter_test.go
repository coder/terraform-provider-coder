package tpfprovider

import (
	"fmt"
	"os"
	"testing"

	"github.com/coder/terraform-provider-coder/v2/provider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccParameterDataSource(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		t.Setenv(provider.ParameterEnvironmentVariable("test"), "test")
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{{
				Config: `
provider coder {}
data "coder_parameter" "test" {
	name = "test"
	type = ""
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testParameterEnv("test", "test"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "name", "test"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "type", ""),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value", "test"),
				),
			}},
		})
	})

	t.Run("number", func(t *testing.T) {
		t.Setenv(provider.ParameterEnvironmentVariable("test"), "3.14")
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{{
				Config: `
provider coder {}
data "coder_parameter" "test" {
	name = "test"
	type = 0
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testParameterEnv("test", "3.14"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "name", "test"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "type", "0"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value", "3.14"),
				),
			}},
		})
	})

	t.Run("bool", func(t *testing.T) {
		t.Setenv(provider.ParameterEnvironmentVariable("test"), "true")
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{{
				Config: `
provider coder {}
data "coder_parameter" "test" {
	name = "test"
	type = false
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testParameterEnv("test", "true"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "name", "test"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "type", "false"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value", "true"),
				),
			}},
		})
	})

	t.Run("list of string", func(t *testing.T) {
		t.Skip("TODO: not implemented yet")
		t.Setenv(provider.ParameterEnvironmentVariable("test"), `["a","b","c"]`)
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{{
				Config: `
provider coder {}
data "coder_parameter" "test" {
	name = "test"
	type = [""]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testParameterEnv("test", `["a","b","c"]`),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "name", "test"),
					resource.TestCheckTypeSetElemAttr("data.coder_parameter.test", "type*", "[]"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.#", "3"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.0", "a"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.1", "b"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.2", "c"),
				),
			}},
		})
	})

	t.Run("list of number", func(t *testing.T) {
		t.Skip("TODO: not implemented yet")
		t.Setenv(provider.ParameterEnvironmentVariable("test"), `[1, 2, 3]`)
		resource.Test(t, resource.TestCase{
			IsUnitTest:               true,
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{{
				Config: `
provider coder {}
data "coder_parameter" "test" {
	name = "test"
	type = [0]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					testParameterEnv("test", `[1,2,3]`),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "name", "test"),
					resource.TestCheckTypeSetElemAttr("data.coder_parameter.test", "type*", "[]"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.#", "3"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.0", "1"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.1", "2"),
					resource.TestCheckResourceAttr("data.coder_parameter.test", "value.2", "3"),
				),
			}},
		})
	})
}

func testParameterEnv(name, value string) func(*terraform.State) error {
	return func(*terraform.State) error {
		penv := provider.ParameterEnvironmentVariable("test")
		val, ok := os.LookupEnv(penv)
		if !ok {
			return fmt.Errorf("parameter environment variable %q not set", penv)
		}
		if val != value {
			return fmt.Errorf("parameter environment variable %q has unexpected value %q", penv, val)
		}
		return nil
	}
}
