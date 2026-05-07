package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestDLPPolicy(t *testing.T) {
	t.Parallel()

	t.Run("AllFieldsSet", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_dlp_policy" "test" {
					ssh_access             = true
					web_terminal_access    = true
					port_forwarding_access = true
					desktop_access         = true
					allowed_applications   = ["code-server", "vscode-desktop"]
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					res := state.Modules[0].Resources["coder_dlp_policy.test"]
					require.NotNil(t, res)

					require.NotEmpty(t, res.Primary.Attributes["id"])
					require.Equal(t, "true", res.Primary.Attributes["ssh_access"])
					require.Equal(t, "true", res.Primary.Attributes["web_terminal_access"])
					require.Equal(t, "true", res.Primary.Attributes["port_forwarding_access"])
					require.Equal(t, "true", res.Primary.Attributes["desktop_access"])
					require.Equal(t, "2", res.Primary.Attributes["allowed_applications.#"])
					require.Equal(t, "code-server", res.Primary.Attributes["allowed_applications.0"])
					require.Equal(t, "vscode-desktop", res.Primary.Attributes["allowed_applications.1"])
					return nil
				},
			}},
		})
	})

	t.Run("Defaults", func(t *testing.T) {
		t.Parallel()

		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_dlp_policy" "test" {
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					res := state.Modules[0].Resources["coder_dlp_policy.test"]
					require.NotNil(t, res)

					require.NotEmpty(t, res.Primary.Attributes["id"])
					require.Equal(t, "false", res.Primary.Attributes["ssh_access"])
					require.Equal(t, "false", res.Primary.Attributes["web_terminal_access"])
					require.Equal(t, "false", res.Primary.Attributes["port_forwarding_access"])
					require.Equal(t, "false", res.Primary.Attributes["desktop_access"])
					// Omitting the field entirely leaves allowed_applications.# blank
					// in state rather than "0"; both mean "no allowed apps".
					require.Empty(t, res.Primary.Attributes["allowed_applications.#"])
					return nil
				},
			}},
		})
	})
}
