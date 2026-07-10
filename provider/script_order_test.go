package provider_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestScriptOrder(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_script_order" "startup" {
				rule {
					run   = ["coder_script.install.id"]
					after = ["coder_script.clone.id"]
					state = "completes"
				}
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				order := state.Modules[0].Resources["coder_script_order.startup"]
				require.NotNil(t, order)
				t.Logf("script_order attributes: %#v", order.Primary.Attributes)
				for key, expected := range map[string]string{
					"rule.#":       "1",
					"rule.0.run.0": "coder_script.install.id",
					"rule.0.state": "completes",
				} {
					require.Equal(t, expected, order.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestScriptOrderEmptyRun(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_script_order" "startup" {
				rule {
					run = []
				}
			}
			`,
			ExpectError: regexp.MustCompile(`"run" must contain at least one coder_script id`),
		}},
	})
}
