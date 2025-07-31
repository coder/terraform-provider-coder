package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestExternalAgent(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				
				resource "coder_external_agent" "main" {
					token       = "token"
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					resource := state.Modules[0].Resources["coder_external_agent.main"]
					require.NotNil(t, resource)
					value := resource.Primary.Attributes["token"]
					require.NotNil(t, value)
					require.Greater(t, len(value), 0)
					return nil
				},
			}},
		})
	})
}
