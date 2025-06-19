package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAITask(t *testing.T) {
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
				resource "coder_app" "code-server" {
					agent_id = coder_agent.dev.id
					slug = "code-server"
					display_name = "code-server"
					icon = "builtin:vim"
					url = "http://localhost:13337"
					open_in = "slim-window"
				}
				resource "coder_ai_task" "test" {
					sidebar_app {
						id = coder_app.code-server.id
					}
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					resource := state.Modules[0].Resources["coder_ai_task.test"]
					require.NotNil(t, resource)
					for _, key := range []string{
						"id",
						"sidebar_app.#",
					} {
						value := resource.Primary.Attributes[key]
						require.NotNil(t, value)
						require.Greater(t, len(value), 0)
					}
					require.Equal(t, "1", resource.Primary.Attributes["sidebar_app.#"])
					return nil
				},
			}},
		})
	})

	t.Run("InvalidSidebarAppID", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_ai_task" "test" {
					sidebar_app {
						id = "not-a-uuid"
					}
				}
				`,
				ExpectError: regexp.MustCompile(`expected "sidebar_app.0.id" to be a valid UUID`),
			}},
		})
	})
}
