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

func TestScript(t *testing.T) {
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
			resource "coder_script" "example" {
				agent_id = "some id"
				display_name = "Hey"
				script = "Wow"
				cron = "* * * * *"
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				script := state.Modules[0].Resources["coder_script.example"]
				require.NotNil(t, script)
				t.Logf("script attributes: %#v", script.Primary.Attributes)
				for key, expected := range map[string]string{
					"agent_id":     "some id",
					"display_name": "Hey",
					"script":       "Wow",
					"cron":         "* * * * *",
				} {
					require.Equal(t, expected, script.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}

func TestScriptNeverRuns(t *testing.T) {
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
			resource "coder_script" "example" {
				agent_id = ""
				display_name = "Hey"
				script = "Wow"
			}
			`,
			ExpectError: regexp.MustCompile(`at least one of "run_on_start", "run_on_stop", or "cron" must be set`),
		}},
	})
}

func TestScriptStartBlocksLoginRequiresRunOnStart(t *testing.T) {
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
			resource "coder_script" "example" {
				agent_id = ""
				display_name = "Hey"
				script = "Wow"
				run_on_stop = true
				start_blocks_login = true
			}
			`,
			ExpectError: regexp.MustCompile(`"start_blocks_login" can only be set if "run_on_start" is "true"`),
		}},
	})
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			resource "coder_script" "example" {
				agent_id = ""
				display_name = "Hey"
				script = "Wow"
				start_blocks_login = true
				run_on_start = true
			}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				script := state.Modules[0].Resources["coder_script.example"]
				require.NotNil(t, script)
				t.Logf("script attributes: %#v", script.Primary.Attributes)
				for key, expected := range map[string]string{
					"agent_id":           "",
					"display_name":       "Hey",
					"script":             "Wow",
					"start_blocks_login": "true",
					"run_on_start":       "true",
				} {
					require.Equal(t, expected, script.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}
