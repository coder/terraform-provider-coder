package provider_test

import (
	"regexp"
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestEnv(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_env" "example" {
				agent_id = "king"
				name = "MESSAGE"
				value = "Believe in yourself and there will come a day when others will have no choice but to believe with you."
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				script := state.Modules[0].Resources["coder_env.example"]
				require.NotNil(t, script)
				t.Logf("script attributes: %#v", script.Primary.Attributes)
				for key, expected := range map[string]string{
					"agent_id": "king",
					"name":     "MESSAGE",
					"value":    "Believe in yourself and there will come a day when others will have no choice but to believe with you.",
				} {
					require.Equal(t, expected, script.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestEnvEmptyValue(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_env" "example" {
				agent_id = "king"
				name = "MESSAGE"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				script := state.Modules[0].Resources["coder_env.example"]
				require.NotNil(t, script)
				t.Logf("script attributes: %#v", script.Primary.Attributes)
				for key, expected := range map[string]string{
					"agent_id": "king",
					"name":     "MESSAGE",
					"value":    "",
				} {
					require.Equal(t, expected, script.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestEnvBadName(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_env" "example" {
				agent_id = ""
				name = "bad-name"
			}
			`,
			ExpectError: regexp.MustCompile(`must be a valid environment variable name`),
		}},
	})
}

func TestEnvNoName(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_env" "example" {
				agent_id = ""
			}
			`,
			ExpectError: regexp.MustCompile(`The argument "name" is required, but no definition was found.`),
		}},
	})
}
