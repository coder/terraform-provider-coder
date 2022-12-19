package provider_test

import (
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
				provider "coder" {
					url = "https://example.com"
				}
				resource "coder_agent" "new" {
					os = "linux"
					arch = "amd64"
					auth = "aws-instance-identity"
					dir = "/tmp"
					env = {
						hi = "test"
					}
					startup_script = "echo test"
					troubleshooting_url = "https://example.com/troubleshoot"
					motd_file = "/etc/motd"
					shutdown_script = "echo bye bye"
				}
				`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["coder_agent.new"]
				require.NotNil(t, resource)
				for _, key := range []string{
					"token",
					"os",
					"arch",
					"auth",
					"dir",
					"env.hi",
					"startup_script",
					"connection_timeout",
					"troubleshooting_url",
					"motd_file",
					"shutdown_script",
				} {
					value := resource.Primary.Attributes[key]
					t.Logf("%q = %q", key, value)
					require.NotNil(t, value)
					require.Greater(t, len(value), 0)
				}
				return nil
			},
		}},
	})
}

func TestAgentInstance(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
				provider "coder" {
					url = "https://example.com"
				}
				resource "coder_agent" "dev" {
					os = "linux"
					arch = "amd64"
				}
				resource "coder_agent_instance" "new" {
					agent_id = coder_agent.dev.id
					instance_id = "hello"
				}
				`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 2)
				resource := state.Modules[0].Resources["coder_agent_instance.new"]
				require.NotNil(t, resource)
				for _, key := range []string{
					"agent_id",
					"instance_id",
				} {
					value := resource.Primary.Attributes[key]
					t.Logf("%q = %q", key, value)
					require.NotNil(t, value)
					require.Greater(t, len(value), 0)
				}
				return nil
			},
		}},
	})
}
