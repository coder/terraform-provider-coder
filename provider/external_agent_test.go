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

				resource "coder_agent" "dev" {
					os = "linux"
					arch = "amd64"
				}
				
				resource "coder_external_agent" "dev" {
					agent_id = coder_agent.dev.id
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 2)

					agentResource := state.Modules[0].Resources["coder_agent.dev"]
					require.NotNil(t, agentResource)
					externalAgentResource := state.Modules[0].Resources["coder_external_agent.dev"]
					require.NotNil(t, externalAgentResource)

					require.Equal(t, agentResource.Primary.Attributes["id"], externalAgentResource.Primary.Attributes["agent_id"])
					return nil
				},
			}},
		})
	})
}
