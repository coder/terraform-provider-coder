package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestScriptOrder(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
				provider "coder" {}
				data "coder_script_order" "startup" {
					rule {
						run = [
							"coder_script.install_tools",
							"coder_script.configure_shell",
						]
						after = [
							"coder_script.clone_repo",
							"coder_script.authenticate",
						]
					}
					rule {
						run      = ["coder_script.setup[\"api\"]"]
						after    = ["module.bootstrap"]
						requires = "completion"
					}
					rule {
						run   = ["not valid selector syntax"]
						after = ["also not valid"]
					}
				}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				order := state.Modules[0].Resources["data.coder_script_order.startup"]
				require.NotNil(t, order)
				require.NotEmpty(t, order.Primary.ID)

				attributes := order.Primary.Attributes
				for key, expected := range map[string]string{
					"rule.#":          "3",
					"rule.0.run.#":    "2",
					"rule.0.run.0":    "coder_script.install_tools",
					"rule.0.run.1":    "coder_script.configure_shell",
					"rule.0.after.#":  "2",
					"rule.0.after.0":  "coder_script.clone_repo",
					"rule.0.after.1":  "coder_script.authenticate",
					"rule.0.requires": "success",
					"rule.1.run.0":    `coder_script.setup["api"]`,
					"rule.1.after.0":  "module.bootstrap",
					"rule.1.requires": "completion",
					"rule.2.run.0":    "not valid selector syntax",
					"rule.2.after.0":  "also not valid",
					"rule.2.requires": "success",
				} {
					require.Equal(t, expected, attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestScriptOrderValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		expectError string
	}{
		{
			name:        "NoRules",
			body:        "",
			expectError: `Insufficient rule blocks`,
		},
		{
			name: "MissingRun",
			body: `
				rule {
					after = ["coder_script.clone_repo"]
				}
			`,
			expectError: `The argument "run" is required, but no definition was found.`,
		},
		{
			name: "MissingAfter",
			body: `
				rule {
					run = ["coder_script.install_tools"]
				}
			`,
			expectError: `The argument "after" is required, but no definition was found.`,
		},
		{
			name: "EmptyRun",
			body: `
				rule {
					run   = []
					after = ["coder_script.clone_repo"]
				}
			`,
			expectError: `Attribute rule.0.run requires 1 item minimum`,
		},
		{
			name: "EmptyAfter",
			body: `
				rule {
					run   = ["coder_script.install_tools"]
					after = []
				}
			`,
			expectError: `Attribute rule.0.after requires 1 item minimum`,
		},
		{
			name: "EmptyRunSelector",
			body: `
				rule {
					run   = [""]
					after = ["coder_script.clone_repo"]
				}
			`,
			expectError: `expected "rule.0.run.0" to not be an empty string`,
		},
		{
			name: "EmptyAfterSelector",
			body: `
				rule {
					run   = ["coder_script.install_tools"]
					after = [""]
				}
			`,
			expectError: `expected "rule.0.after.0" to not be an empty string`,
		},
		{
			name: "InvalidRequirement",
			body: `
				rule {
					run      = ["coder_script.install_tools"]
					after    = ["coder_script.clone_repo"]
					requires = "starts"
				}
			`,
			expectError: `expected rule.0.requires to be one of \["success" "completion"\]`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: `
						provider "coder" {}
						data "coder_script_order" "startup" {
					` + test.body + `
						}
					`,
					ExpectError: regexp.MustCompile(test.expectError),
				}},
			})
		})
	}
}
