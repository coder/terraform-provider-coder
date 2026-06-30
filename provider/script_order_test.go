package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestScriptOrder(t *testing.T) {
	t.Parallel()

	t.Run("ArrayRunAndAfter", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				resource "coder_agent" "main" {
					os   = "linux"
					arch = "amd64"
				}
				resource "coder_script" "clone" {
					agent_id     = coder_agent.main.id
					display_name = "Clone repository"
					run_on_start = true
					script       = "git clone https://github.com/coder/coder ~/coder"
				}
				resource "coder_script" "install" {
					agent_id     = coder_agent.main.id
					display_name = "Install dependencies"
					run_on_start = true
					script       = "cd ~/coder && make install"
				}
				data "coder_script_order" "startup" {
					rule {
						run   = ["coder_script.install"]
						after = ["coder_script.clone"]
					}
				}`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					res := state.Modules[0].Resources["data.coder_script_order.startup"]
					require.NotNil(t, res)

					attribs := res.Primary.Attributes
					require.Equal(t, "1", attribs["rule.#"])
					require.Equal(t, "1", attribs["rule.0.run.#"])
					require.Equal(t, "coder_script.install", attribs["rule.0.run.0"])
					require.Equal(t, "1", attribs["rule.0.after.#"])
					require.Equal(t, "coder_script.clone", attribs["rule.0.after.0"])
					require.Equal(t, "success", attribs["rule.0.requires"])
					return nil
				},
			}},
		})
	})

	t.Run("MultipleRunSelectorsAndExplicitRequires", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				data "coder_script_order" "startup" {
					rule {
						run      = ["coder_script.a", "coder_script.b"]
						after    = ["module.git_clone"]
						requires = "completion"
					}
				}`,
				Check: func(state *terraform.State) error {
					res := state.Modules[0].Resources["data.coder_script_order.startup"]
					require.NotNil(t, res)

					attribs := res.Primary.Attributes
					require.Equal(t, "2", attribs["rule.0.run.#"])
					require.Equal(t, "coder_script.a", attribs["rule.0.run.0"])
					require.Equal(t, "coder_script.b", attribs["rule.0.run.1"])
					require.Equal(t, "module.git_clone", attribs["rule.0.after.0"])
					require.Equal(t, "completion", attribs["rule.0.requires"])
					return nil
				},
			}},
		})
	})

	t.Run("InvalidRequiresRejected", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				data "coder_script_order" "startup" {
					rule {
						run      = ["coder_script.install"]
						after    = ["coder_script.clone"]
						requires = "maybe"
					}
				}`,
				ExpectError: regexp.MustCompile(`expected rule\.0\.requires to be one of \["success" "completion"\]`),
			}},
		})
	})

	t.Run("EmptyRunRejected", func(t *testing.T) {
		t.Parallel()
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
				}
				data "coder_script_order" "startup" {
					rule {
						run   = []
						after = ["coder_script.clone"]
					}
				}`,
				ExpectError: regexp.MustCompile(`Attribute rule\.0\.run requires 1 item minimum`),
			}},
		})
	})
}
