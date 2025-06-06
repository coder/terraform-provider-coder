package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestWorkspacePreset(t *testing.T) {
	t.Parallel()
	type testcase struct {
		Name        string
		Config      string
		ExpectError *regexp.Regexp
		Check       func(state *terraform.State) error
	}
	testcases := []testcase{
		{
			Name: "Happy Path",
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
		},
		{
			Name: "Name field is not provided",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			// This validation is done by Terraform, but it could still break if we misconfigure the schema.
			// So we test it here to make sure we don't regress.
			ExpectError: regexp.MustCompile("The argument \"name\" is required, but no definition was found"),
		},
		{
			Name: "Name field is empty",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = ""
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			// This validation is done by Terraform, but it could still break if we misconfigure the schema.
			// So we test it here to make sure we don't regress.
			ExpectError: regexp.MustCompile("expected \"name\" to not be an empty string"),
		},
		{
			Name: "Name field is not a string",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = [1, 2, 3]
				parameters = {
					"region" = "us-east1-a"
				}
			}`,
			// This validation is done by Terraform, but it could still break if we misconfigure the schema.
			// So we test it here to make sure we don't regress.
			ExpectError: regexp.MustCompile("Incorrect attribute value type"),
		},
		{
			Name: "Parameters field is not provided",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
			}`,
			// This validation is done by Terraform, but it could still break if we misconfigure the schema.
			// So we test it here to make sure we don't regress.
			ExpectError: nil,
		},
		{
			Name: "Parameters field is empty",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {}
			}`,
			// This validation is *not* done by Terraform, because MinItems doesn't work with maps.
			// We've implemented the validation in ReadContext, so we test it here to make sure we don't regress.
			ExpectError: nil,
		},
		{
			Name: "Parameters field is not a map",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = "not a map"
			}`,
			// This validation is done by Terraform, but it could still break if we misconfigure the schema.
			// So we test it here to make sure we don't regress.
			ExpectError: regexp.MustCompile("Inappropriate value for attribute \"parameters\": map of string required"),
		},
		{
			Name: "Prebuilds is set, but not its required fields",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {}
			}`,
			ExpectError: regexp.MustCompile("The argument \"instances\" is required, but no definition was found."),
		},
		{
			Name: "Prebuilds is set, and so are its required fields",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
				}
			}`,
			ExpectError: nil,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["data.coder_workspace_preset.preset_1"]
				require.NotNil(t, resource)
				attrs := resource.Primary.Attributes
				require.Equal(t, attrs["name"], "preset_1")
				require.Equal(t, attrs["prebuilds.0.instances"], "1")
				return nil
			},
		},
		{
			Name: "Prebuilds is set with a expiration_policy field without its required fields",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
					expiration_policy {}
				}
			}`,
			ExpectError: regexp.MustCompile(`The argument "ttl" is required, but no definition was found.`),
		},
		{
			Name: "Prebuilds is set with a expiration_policy field with its required fields",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
					expiration_policy {
						ttl = 86400
					}
				}
			}`,
			ExpectError: nil,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["data.coder_workspace_preset.preset_1"]
				require.NotNil(t, resource)
				attrs := resource.Primary.Attributes
				require.Equal(t, attrs["name"], "preset_1")
				require.Equal(t, attrs["prebuilds.0.expiration_policy.0.ttl"], "86400")
				return nil
			},
		},
		{
			Name: "Prebuilds block with expiration_policy.ttl set to 0 seconds (disables expiration)",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
					expiration_policy {
						ttl = 0
					}
				}
			}`,
			ExpectError: nil,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["data.coder_workspace_preset.preset_1"]
				require.NotNil(t, resource)
				attrs := resource.Primary.Attributes
				require.Equal(t, attrs["name"], "preset_1")
				require.Equal(t, attrs["prebuilds.0.expiration_policy.0.ttl"], "0")
				return nil
			},
		},
		{
			Name: "Prebuilds block with expiration_policy.ttl set to 30 minutes (below 1 hour limit)",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
					expiration_policy {
						ttl = 1800
					}
				}
			}`,
			ExpectError: regexp.MustCompile(`"prebuilds.0.expiration_policy.0.ttl" must be 0 or between 3600 and 31536000, got 1800`),
		},
		{
			Name: "Prebuilds block with expiration_policy.ttl set to 2 years (exceeds 1 year limit)",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
					expiration_policy {
						ttl = 63072000
					}
				}
			}`,
			ExpectError: regexp.MustCompile(`"prebuilds.0.expiration_policy.0.ttl" must be 0 or between 3600 and 31536000, got 63072000`),
		},
		{
			Name: "Prebuilds is set with a expiration_policy field with its required fields and an unexpected argument",
			Config: `
			data "coder_workspace_preset" "preset_1" {
				name = "preset_1"
				parameters = {
					"region" = "us-east1-a"
				}
				prebuilds {
					instances = 1
					expiration_policy {
						ttl = 86400
						invalid_argument = "test"
					}
				}
			}`,
			ExpectError: regexp.MustCompile("An argument named \"invalid_argument\" is not expected here."),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.Name, func(t *testing.T) {
			t.Parallel()

			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config:      testcase.Config,
					ExpectError: testcase.ExpectError,
					Check:       testcase.Check,
				}},
			})
		})
	}
}
