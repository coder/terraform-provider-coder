package provider_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestDevcontainer(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_devcontainer" "example" {
				agent_id = "king"
				workspace_folder = "/workspace"
				config_path = "/workspace/devcontainer.json"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				script := state.Modules[0].Resources["coder_devcontainer.example"]
				require.NotNil(t, script)
				t.Logf("script attributes: %#v", script.Primary.Attributes)
				for key, expected := range map[string]string{
					"agent_id":         "king",
					"workspace_folder": "/workspace",
					"config_path":      "/workspace/devcontainer.json",
				} {
					require.Equal(t, expected, script.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestDevcontainerNoConfigPath(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_devcontainer" "example" {
				agent_id = "king"
				workspace_folder = "/workspace"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				script := state.Modules[0].Resources["coder_devcontainer.example"]
				require.NotNil(t, script)
				t.Logf("script attributes: %#v", script.Primary.Attributes)
				for key, expected := range map[string]string{
					"agent_id":         "king",
					"workspace_folder": "/workspace",
				} {
					require.Equal(t, expected, script.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestDevcontainerNoWorkspaceFolder(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_devcontainer" "example" {
				agent_id = ""
			}
			`,
			ExpectError: regexp.MustCompile(`The argument "workspace_folder" is required, but no definition was found.`),
		}},
	})
}
