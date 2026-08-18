package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestWorkspaceAIAgentDatasource(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		t.Setenv("CODER_WORKSPACE_AI_AGENT_ID", "22222222-2222-2222-2222-222222222222")
		t.Setenv("CODER_WORKSPACE_AI_AGENT_SESSION_TOKEN", "ai-agent-session-token")

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
			provider "coder" {}
			data "coder_workspace_ai_agent" "me" {}
			`,
				Check: func(s *terraform.State) error {
					require.Len(t, s.Modules, 1)
					require.Len(t, s.Modules[0].Resources, 1)
					resource := s.Modules[0].Resources["data.coder_workspace_ai_agent.me"]
					require.NotNil(t, resource)
					attrs := resource.Primary.Attributes
					require.Equal(t, "22222222-2222-2222-2222-222222222222", attrs["id"])
					require.Equal(t, "ai-agent-session-token", attrs["session_token"])
					return nil
				},
			}},
		})
	})

	t.Run("NotProvisioned", func(t *testing.T) {
		// The token is empty when the deployment does not provision an AI
		// agent identity; the data source must not fail.
		t.Setenv("CODER_WORKSPACE_AI_AGENT_SESSION_TOKEN", "")

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
			provider "coder" {}
			data "coder_workspace_ai_agent" "me" {}
			`,
				Check: func(s *terraform.State) error {
					resource := s.Modules[0].Resources["data.coder_workspace_ai_agent.me"]
					require.NotNil(t, resource)
					require.Empty(t, resource.Primary.Attributes["session_token"])
					require.NotEmpty(t, resource.Primary.Attributes["id"])
					return nil
				},
			}},
		})
	})
}
