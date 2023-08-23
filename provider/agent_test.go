package provider_test

import (
	"fmt"
	"regexp"
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
					startup_script_timeout = 120
					troubleshooting_url = "https://example.com/troubleshoot"
					motd_file = "/etc/motd"
					shutdown_script = "echo bye bye"
					shutdown_script_timeout = 120
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
					"startup_script_timeout",
					"connection_timeout",
					"troubleshooting_url",
					"motd_file",
					"shutdown_script",
					"shutdown_script_timeout",
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

func TestAgent_StartupScriptBehavior(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		Name        string
		Config      string
		ExpectError *regexp.Regexp
		Check       func(state *terraform.ResourceState)
	}{
		{
			Name: "blocking",
			Config: `
				resource "coder_agent" "new" {
					os = "linux"
					arch = "amd64"
					startup_script_behavior = "blocking"
				}
			`,
			Check: func(state *terraform.ResourceState) {
				require.Equal(t, "blocking", state.Primary.Attributes["startup_script_behavior"])
			},
		},
		{
			Name: "non-blocking",
			Config: `
				resource "coder_agent" "new" {
					os = "linux"
					arch = "amd64"
					startup_script_behavior = "non-blocking"
				}
			`,
			Check: func(state *terraform.ResourceState) {
				require.Equal(t, "non-blocking", state.Primary.Attributes["startup_script_behavior"])
			},
		},
		{
			Name: "login_before_ready (deprecated)",
			Config: `
				resource "coder_agent" "new" {
					os = "linux"
					arch = "amd64"
					login_before_ready = false
				}
			`,
			Check: func(state *terraform.ResourceState) {
				require.Equal(t, "false", state.Primary.Attributes["login_before_ready"])
				// startup_script_behavior must be empty, this indicates that
				// login_before_ready should be used instead.
				require.Equal(t, "", state.Primary.Attributes["startup_script_behavior"])
			},
		},
		{
			Name: "no login_before_ready with startup_script_behavior",
			Config: `
				resource "coder_agent" "new" {
					os = "linux"
					arch = "amd64"
					login_before_ready = false
					startup_script_behavior = "blocking"
				}
			`,
			ExpectError: regexp.MustCompile("conflicts with"),
		},
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				Providers: map[string]*schema.Provider{
					"coder": provider.New(),
				},
				IsUnitTest: true,
				Steps: []resource.TestStep{{
					Config:      tc.Config,
					ExpectError: tc.ExpectError,
					Check: func(state *terraform.State) error {
						require.Len(t, state.Modules, 1)
						require.Len(t, state.Modules[0].Resources, 1)
						resource := state.Modules[0].Resources["coder_agent.new"]
						require.NotNil(t, resource)
						if tc.Check != nil {
							tc.Check(resource)
						}
						return nil
					},
				}},
			})
		})
	}
}

func TestAgent_Instance(t *testing.T) {
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

func TestAgent_Metadata(t *testing.T) {
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
					metadata {
						key = "process_count"
						display_name = "Process Count"
						script = "ps aux | wc -l"
						interval = 5
						timeout = 1
					}
				}
				`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)

				resource := state.Modules[0].Resources["coder_agent.dev"]
				require.NotNil(t, resource)

				t.Logf("resource: %v", resource.Primary.Attributes)

				attr := resource.Primary.Attributes
				require.Equal(t, "1", attr["metadata.#"])
				require.Equal(t, "process_count", attr["metadata.0.key"])
				require.Equal(t, "Process Count", attr["metadata.0.display_name"])
				require.Equal(t, "ps aux | wc -l", attr["metadata.0.script"])
				require.Equal(t, "5", attr["metadata.0.interval"])
				require.Equal(t, "1", attr["metadata.0.timeout"])
				return nil
			},
		}},
	})
}

func TestAgent_DefaultApps(t *testing.T) {
	t.Parallel()
	t.Run("OK", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			Providers: map[string]*schema.Provider{
				"coder": provider.New(),
			},
			IsUnitTest: true,
			Steps: []resource.TestStep{{
				// Test the fields with non-default values.
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						display_apps {
							vscode = false
							vscode_insiders = true
							web_terminal = false
							port_forwarding_helper = false
							ssh_helper = false
						} 
					}
					`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 1)

					resource := state.Modules[0].Resources["coder_agent.dev"]
					require.NotNil(t, resource)

					t.Logf("resource: %v", resource.Primary.Attributes)

					for _, app := range []string{
						"web_terminal",
						"vscode_insiders",
						"vscode",
						"port_forwarding_helper",
						"ssh_helper",
					} {
						key := fmt.Sprintf("display_apps.0.%s", app)
						if app == "vscode_insiders" {
							require.Equal(t, "true", resource.Primary.Attributes[key])
						} else {
							require.Equal(t, "false", resource.Primary.Attributes[key])
						}
					}
					return nil
				},
			}},
		})

	})

	// Assert all the defaults are set correctly.
	t.Run("Omitted", func(t *testing.T) {
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
					`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 1)

					resource := state.Modules[0].Resources["coder_agent.dev"]
					require.NotNil(t, resource)

					t.Logf("resource: %v", resource.Primary.Attributes)

					for _, app := range []string{
						"web_terminal",
						"vscode_insiders",
						"vscode",
						"port_forwarding_helper",
						"ssh_helper",
					} {
						key := fmt.Sprintf("display_apps.0.%s", app)
						if app == "vscode_insiders" {
							require.Equal(t, "false", resource.Primary.Attributes[key])
						} else {
							require.Equal(t, "true", resource.Primary.Attributes[key])
						}
					}
					return nil
				},
			}},
		})
	})
}
