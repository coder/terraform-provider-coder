package provider_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAgent(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
					order = 4
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
					"order",
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
	} {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
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
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
						order = 7
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
				require.Equal(t, "7", attr["metadata.0.order"])
				return nil
			},
		}},
	})
}

func TestAgent_ResourcesMonitoring(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
				provider "coder" {
					url = "https://example.com"
				}
				resource "coder_agent" "dev" {
					os = "linux"
					arch = "amd64"
					resources_monitoring {
						memory {
							enabled = true
							threshold = 80
						}
						volume {
							path = "/volume1"
							enabled = true
							threshold = 80
						}
						volume {
							path = "/volume2"
							enabled = true
							threshold = 100
						}
					}
				}`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 1)

					resource := state.Modules[0].Resources["coder_agent.dev"]
					require.NotNil(t, resource)

					attr := resource.Primary.Attributes
					require.Equal(t, "1", attr["resources_monitoring.#"])
					require.Equal(t, "1", attr["resources_monitoring.0.memory.#"])
					require.Equal(t, "2", attr["resources_monitoring.0.volume.#"])
					require.Equal(t, "80", attr["resources_monitoring.0.memory.0.threshold"])
					require.Equal(t, "/volume1", attr["resources_monitoring.0.volume.0.path"])
					require.Equal(t, "100", attr["resources_monitoring.0.volume.1.threshold"])
					require.Equal(t, "/volume2", attr["resources_monitoring.0.volume.1.path"])
					return nil
				},
			}},
		})
	})

	t.Run("OnlyMemory", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							memory {
								enabled = true
								threshold = 80
							}
						}
					}`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 1)

					resource := state.Modules[0].Resources["coder_agent.dev"]
					require.NotNil(t, resource)

					attr := resource.Primary.Attributes
					require.Equal(t, "1", attr["resources_monitoring.#"])
					require.Equal(t, "1", attr["resources_monitoring.0.memory.#"])
					require.Equal(t, "80", attr["resources_monitoring.0.memory.0.threshold"])
					return nil
				},
			}},
		})
	})
	t.Run("MultipleMemory", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							memory {
								enabled = true
								threshold = 80
							}
							memory {
								enabled = true
								threshold = 90
							}
						}
					}`,
				ExpectError: regexp.MustCompile(`No more than 1 "memory" blocks are allowed`),
			}},
		})
	})

	t.Run("InvalidThreshold", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							memory {
								enabled = true
								threshold = 101
							}
						}
					}`,
				Check:       nil,
				ExpectError: regexp.MustCompile(`expected resources_monitoring\.0\.memory\.0\.threshold to be in the range \(0 - 100\), got 101`),
			}},
		})
	})

	t.Run("DuplicatePaths", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							volume {
								path = "/volume1"
								enabled = true
								threshold = 80
							}
							volume {
								path = "/volume1"
								enabled = true
								threshold = 100
							}
						}
					}`,
				ExpectError: regexp.MustCompile("duplicate volume path"),
			}},
		})
	})

	t.Run("NoPath", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							volume {
								enabled = true
								threshold = 80
							}
						}
					}`,
				ExpectError: regexp.MustCompile(`The argument "path" is required, but no definition was found.`),
			}},
		})
	})

	t.Run("NonAbsPath", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							volume {
								path = "tmp"
								enabled = true
								threshold = 80
							}
						}
					}`,
				Check:       nil,
				ExpectError: regexp.MustCompile(`volume path must be an absolute path`),
			}},
		})
	})

	t.Run("EmptyPath", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							volume {
								path = ""
								enabled = true
								threshold = 80
							}
						}
					}`,
				Check:       nil,
				ExpectError: regexp.MustCompile(`volume path must not be empty`),
			}},
		})
	})

	t.Run("ThresholdMissing", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							volume {
								path = "/volume1"
								enabled = true
							}
						}
					}`,
				Check:       nil,
				ExpectError: regexp.MustCompile(`The argument "threshold" is required, but no definition was found.`),
			}},
		})
	})
	t.Run("EnabledMissing", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
			Steps: []resource.TestStep{{
				Config: `
					provider "coder" {
						url = "https://example.com"
					}
					resource "coder_agent" "dev" {
						os = "linux"
						arch = "amd64"
						resources_monitoring {
							memory {
								threshold = 80
							}
						}
					}`,
				Check:       nil,
				ExpectError: regexp.MustCompile(`The argument "enabled" is required, but no definition was found.`),
			}},
		})
	})
}

func TestAgent_MetadataDuplicateKeys(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
					metadata {
						key = "process_count"
						display_name = "Process Count"
						script = "ps aux | wc -l"
						interval = 5
						timeout = 1
					}
				}
				`,
			ExpectError: regexp.MustCompile("duplicate agent metadata key"),
			PlanOnly:    true,
		}},
	})
}

func TestAgent_DisplayApps(t *testing.T) {
	t.Parallel()
	t.Run("OK", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
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

	t.Run("Subset", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
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
							vscode_insiders = true
							web_terminal = true
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
						require.Equal(t, "true", resource.Primary.Attributes[key])
					}
					return nil
				},
			}},
		})
	})

	// Assert all the defaults are set correctly.
	t.Run("Omitted", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
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

	t.Run("InvalidApp", func(t *testing.T) {
		resource.Test(t, resource.TestCase{
			ProviderFactories: coderFactory(),
			IsUnitTest:        true,
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
							fake_app = false
							vscode_insiders = true
							web_terminal = false
							port_forwarding_helper = false
							ssh_helper = false
						}
					}
					`,
				ExpectError: regexp.MustCompile(`An argument named "fake_app" is not expected here.`),
			}},
		})
	})

}
