package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestWorkspacePreset(t *testing.T) {
	// Happy Path:
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["data.coder_workspace_preset.preset_1"]
				require.NotNil(t, resource)
				attrs := resource.Primary.Attributes
				require.Equal(t, attrs["name"], "preset_1")
				require.Equal(t, attrs["parameters.region"], "us-east1-a")
				return nil
			},
		}},
	})

	// Given the Name field is not provided
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			// This is from terraform's validation based on our schema, not based on our validation in ReadContext:
			ExpectError: regexp.MustCompile("The argument \"name\" is required, but no definition was found"),
		}},
	})

	// Given the Name field is empty
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = ""
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			ExpectError: regexp.MustCompile("workspace preset name must be set"),
		}},
	})

	// Given the Name field is not a string
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = [1, 2, 3]
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			ExpectError: regexp.MustCompile("Incorrect attribute value type"),
		}},
	})

	// Given the Parameters field is not provided
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
			}`,
			ExpectError: regexp.MustCompile("The argument \"parameters\" is required, but no definition was found"),
		}},
	})

	// Given the Parameters field is empty
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {}
			}`,
			ExpectError: regexp.MustCompile("workspace preset must define a value for at least one parameter"),
		}},
	})

	// Given the Parameters field is not a map
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = "not a map"
			}`,
			// This is from terraform's validation based on our schema, not based on our validation in ReadContext:
			ExpectError: regexp.MustCompile("Inappropriate value for attribute \"parameters\": map of string required"),
		}},
	})
}
