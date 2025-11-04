package provider_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestScript(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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

func TestValidateCronExpression(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		cronExpr        string
		expectWarnings  bool
		expectErrors    bool
		warningContains string
	}{
		{
			name:           "valid 6-field expression",
			cronExpr:       "0 0 22 * * *",
			expectWarnings: false,
			expectErrors:   false,
		},
		{
			name:           "valid 6-field expression with seconds",
			cronExpr:       "30 0 9 * * 1-5",
			expectWarnings: false,
			expectErrors:   false,
		},
		{
			name:            "5-field Unix format - should warn",
			cronExpr:        "0 22 * * *",
			expectWarnings:  true,
			expectErrors:    false,
			warningContains: "appears to be in Unix 5-field format",
		},
		{
			name:            "5-field every 5 minutes - should warn",
			cronExpr:        "*/5 * * * *",
			expectWarnings:  true,
			expectErrors:    false,
			warningContains: "Consider prefixing with '0 '",
		},
		{
			name:         "invalid expression",
			cronExpr:     "invalid",
			expectErrors: true,
		},
		{
			name:         "too many fields",
			cronExpr:     "0 0 0 0 0 0 0",
			expectErrors: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := provider.ValidateCronExpression(tt.cronExpr)

			if tt.expectWarnings {
				require.NotEmpty(t, warnings, "Expected warnings but got none")
				if tt.warningContains != "" {
					require.Contains(t, warnings[0], tt.warningContains)
				}
			} else {
				require.Empty(t, warnings, "Expected no warnings but got: %v", warnings)
			}

			if tt.expectErrors {
				require.NotEmpty(t, errors, "Expected errors but got none")
			} else {
				require.Empty(t, errors, "Expected no errors but got: %v", errors)
			}
		})
	}
}
