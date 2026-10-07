package provider_test

import (
	"maps"
	"regexp"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/v2/provider"
)

func TestScriptOrder(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
		Steps: []resource.TestStep{{
			Config: `
				provider "coder" {}
				data "coder_script_order" "startup" {
					rule {
						run = [
							"coder_script.install_tools",
							"coder_script.configure_shell",
						]
						after = [
							"coder_script.clone_repo",
							"coder_script.authenticate",
						]
					}
					rule {
						run      = ["coder_script.setup[\"api\"]"]
						after    = ["module.bootstrap"]
						requires = "completion"
					}
					rule {
						run   = ["module.application"]
						after = ["module.checkout"]
						phase = "stop"
					}
					rule {
						run   = ["not valid selector syntax"]
						after = ["also not valid"]
					}
				}
			`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				order := state.Modules[0].Resources["data.coder_script_order.startup"]
				require.NotNil(t, order)
				require.NotEmpty(t, order.Primary.ID)

				attributes := order.Primary.Attributes
				for key, expected := range map[string]string{
					"rule.#":          "4",
					"rule.0.run.#":    "2",
					"rule.0.run.0":    "coder_script.install_tools",
					"rule.0.run.1":    "coder_script.configure_shell",
					"rule.0.after.#":  "2",
					"rule.0.after.0":  "coder_script.clone_repo",
					"rule.0.after.1":  "coder_script.authenticate",
					"rule.0.requires": "success",
					"rule.0.phase":    "",
					"rule.1.run.0":    `coder_script.setup["api"]`,
					"rule.1.after.0":  "module.bootstrap",
					"rule.1.requires": "completion",
					"rule.1.phase":    "",
					"rule.2.run.0":    "module.application",
					"rule.2.after.0":  "module.checkout",
					"rule.2.requires": "success",
					"rule.2.phase":    "stop",
					"rule.3.run.0":    "not valid selector syntax",
					"rule.3.after.0":  "also not valid",
					"rule.3.requires": "success",
				} {
					require.Equal(t, expected, attributes[key], key)
				}
				return nil
			},
		}},
	})
}

func TestScriptOrderValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		expectError string
	}{
		{
			name:        "NoRules",
			body:        "",
			expectError: `Insufficient rule blocks`,
		},
		{
			name: "MissingRun",
			body: `
				rule {
					after = ["coder_script.clone_repo"]
				}
			`,
			expectError: `The argument "run" is required, but no definition was found.`,
		},
		{
			name: "MissingAfter",
			body: `
				rule {
					run = ["coder_script.install_tools"]
				}
			`,
			expectError: `The argument "after" is required, but no definition was found.`,
		},
		{
			name: "EmptyRun",
			body: `
				rule {
					run   = []
					after = ["coder_script.clone_repo"]
				}
			`,
			expectError: `Attribute rule.0.run requires 1 item minimum`,
		},
		{
			name: "EmptyAfter",
			body: `
				rule {
					run   = ["coder_script.install_tools"]
					after = []
				}
			`,
			expectError: `Attribute rule.0.after requires 1 item minimum`,
		},
		{
			name: "EmptyRunSelector",
			body: `
				rule {
					run   = [""]
					after = ["coder_script.clone_repo"]
				}
			`,
			expectError: `expected "rule.0.run.0" to not be an empty string`,
		},
		{
			name: "EmptyAfterSelector",
			body: `
				rule {
					run   = ["coder_script.install_tools"]
					after = [""]
				}
			`,
			expectError: `expected "rule.0.after.0" to not be an empty string`,
		},
		{
			name: "InvalidRequires",
			body: `
				rule {
					run      = ["coder_script.install_tools"]
					after    = ["coder_script.clone_repo"]
					requires = "starts"
				}
			`,
			expectError: `expected rule.0.requires to be one of \["success" "completion"\], got starts`,
		},
		{
			// Pins what the SDK does with an explicit empty string. Coder
			// treats "" as "success", so either outcome is safe; the test
			// exists so an SDK upgrade cannot change it unnoticed.
			name: "EmptyRequires",
			body: `
				rule {
					run      = ["coder_script.install_tools"]
					after    = ["coder_script.clone_repo"]
					requires = ""
				}
			`,
			expectError: `expected rule.0.requires to be one of \["success" "completion"\], got `,
		},
		{
			name: "InvalidPhase",
			body: `
				rule {
					run   = ["coder_script.install_tools"]
					after = ["coder_script.clone_repo"]
					phase = "both"
				}
			`,
			expectError: `expected rule.0.phase to be one of \["start" "stop"\], got both`,
		},
		{
			name: "InvalidPhaseCase",
			body: `
				rule {
					run   = ["coder_script.install_tools"]
					after = ["coder_script.clone_repo"]
					phase = "Start"
				}
			`,
			expectError: `expected rule.0.phase to be one of \["start" "stop"\], got Start`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: `
						provider "coder" {}
						data "coder_script_order" "startup" {
					` + test.body + `
						}
					`,
					ExpectError: regexp.MustCompile(test.expectError),
				}},
			})
		})
	}
}

// Coder's provisioner decodes these names from the plan JSON
// (coder/coder provisioner/terraform/scriptorder.go). Renaming one here
// breaks ordering there without a compile error on either side.
func TestScriptOrderContract(t *testing.T) {
	t.Parallel()

	ds := provider.New().DataSourcesMap["coder_script_order"]
	require.NotNil(t, ds)

	rule, ok := ds.Schema["rule"].Elem.(*schema.Resource)
	require.True(t, ok)
	require.ElementsMatch(t,
		[]string{"run", "after", "requires", "phase"},
		slices.Collect(maps.Keys(rule.Schema)),
	)
	require.Equal(t, "success", rule.Schema["requires"].Default)
	require.Nil(t, rule.Schema["phase"].Default)
}
