package provider_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
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
					subdomain = false
					url = "http://localhost:13337"
					healthcheck {
						url = "http://localhost:13337/healthz"
						interval = 5
						threshold = 6
					}
					group = "Apps"
					order = 4
					hidden = false
					open_in = "slim-window"
					tooltip = "You need to [Install Coder Desktop](https://coder.com/docs/user-guides/desktop#install-coder-desktop) to use this button."
				}
				`,
				Check: func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 2)
					resource := state.Modules[0].Resources["coder_app.code-server"]
					require.NotNil(t, resource)
					for _, key := range []string{
						"agent_id",
						"slug",
						"display_name",
						"icon",
						"subdomain",
						// Should be set by default even though it isn't
						// specified.
						"share",
						"url",
						"healthcheck.0.url",
						"healthcheck.0.interval",
						"healthcheck.0.threshold",
						"group",
						"order",
						"hidden",
						"open_in",
						"tooltip",
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
	})

	t.Run("External", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			config      string
			external    bool
			expectError *regexp.Regexp
		}{{
			name: "Valid",
			config: `
			provider "coder" {}
			resource "coder_agent" "dev" {
				os = "linux"
				arch = "amd64"
			}
			resource "coder_app" "test" {
				agent_id = coder_agent.dev.id
				slug = "test"
				display_name = "Testing"
				url = "https://google.com"
				external = true
				open_in = "slim-window"
			}
			`,
			external: true,
		}, {
			name: "ConflictsWithSubdomain",
			config: `
			provider "coder" {}
			resource "coder_agent" "dev" {
				os = "linux"
				arch = "amd64"
			}
			resource "coder_app" "test" {
				agent_id = coder_agent.dev.id
				slug = "test"
				display_name = "Testing"
				url = "https://google.com"
				external = true
				subdomain = true
				open_in = "slim-window"
			}
			`,
			expectError: regexp.MustCompile("conflicts with subdomain"),
		}}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config: tc.config,
						Check: func(state *terraform.State) error {
							require.Len(t, state.Modules, 1)
							require.Len(t, state.Modules[0].Resources, 2)
							resource := state.Modules[0].Resources["coder_app.test"]
							require.NotNil(t, resource)
							require.Equal(t, strconv.FormatBool(tc.external), resource.Primary.Attributes["external"])
							return nil
						},
						ExpectError: tc.expectError,
					}},
				})
			})
		}
	})

	t.Run("SharingLevel", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			value       string
			expectValue string
			expectError *regexp.Regexp
		}{
			{
				name:        "Default",
				value:       "", // default
				expectValue: "owner",
			},
			{
				name:        "InvalidValue",
				value:       "blah",
				expectError: regexp.MustCompile(`invalid app share "blah"`),
			},
			{
				name:        "ExplicitOwner",
				value:       "owner",
				expectValue: "owner",
			},
			{
				name:        "ExplicitAuthenticated",
				value:       "authenticated",
				expectValue: "authenticated",
			},
			{
				name:        "ExplicitPublic",
				value:       "public",
				expectValue: "public",
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				sharingLine := ""
				if c.value != "" {
					sharingLine = fmt.Sprintf("share = %q", c.value)
				}
				config := fmt.Sprintf(`
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
					%s
					healthcheck {
						url = "http://localhost:13337/healthz"
						interval = 5
						threshold = 6
					}
					open_in = "slim-window"
				}
				`, sharingLine)

				checkFn := func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 2)
					resource := state.Modules[0].Resources["coder_app.code-server"]
					require.NotNil(t, resource)

					// Read share and ensure it matches the expected
					// value.
					value := resource.Primary.Attributes["share"]
					require.Equal(t, c.expectValue, value)
					return nil
				}
				if c.expectError != nil {
					checkFn = nil
				}

				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config:      config,
						Check:       checkFn,
						ExpectError: c.expectError,
					}},
				})
			})
		}
	})

	t.Run("OpenIn", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			value       string
			expectValue string
			expectError *regexp.Regexp
		}{
			{
				name:        "default",
				value:       "", // default
				expectValue: "slim-window",
			},
			{
				name:        "InvalidValue",
				value:       "nonsense",
				expectError: regexp.MustCompile(`invalid "coder_app" open_in value, must be one of "tab", "slim-window": "nonsense"`),
			},
			{
				name:        "ExplicitSlimWindow",
				value:       "slim-window",
				expectValue: "slim-window",
			},
			{
				name:        "ExplicitTab",
				value:       "tab",
				expectValue: "tab",
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				config := `
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
					healthcheck {
						url = "http://localhost:13337/healthz"
						interval = 5
						threshold = 6
					}`

				if c.value != "" {
					config += fmt.Sprintf(`
					open_in = %q
					`, c.value)
				}

				config += `
				}
				`

				checkFn := func(state *terraform.State) error {
					require.Len(t, state.Modules, 1)
					require.Len(t, state.Modules[0].Resources, 2)
					resource := state.Modules[0].Resources["coder_app.code-server"]
					require.NotNil(t, resource)

					// Read share and ensure it matches the expected
					// value.
					value := resource.Primary.Attributes["open_in"]
					require.Equal(t, c.expectValue, value)
					return nil
				}
				if c.expectError != nil {
					checkFn = nil
				}

				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config:      config,
						Check:       checkFn,
						ExpectError: c.expectError,
					}},
				})
			})
		}
	})

	t.Run("Hidden", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name   string
			config string
			hidden bool
			openIn string
		}{{
			name: "Is Hidden",
			config: `
			provider "coder" {}
			resource "coder_agent" "dev" {
				os = "linux"
				arch = "amd64"
			}
			resource "coder_app" "test" {
				agent_id = coder_agent.dev.id
				slug = "test"
				display_name = "Testing"
				url = "https://google.com"
				external = true
				hidden = true
				open_in = "slim-window"
			}
			`,
			hidden: true,
			openIn: "slim-window",
		}, {
			name: "Is Not Hidden",
			config: `
			provider "coder" {}
			resource "coder_agent" "dev" {
				os = "linux"
				arch = "amd64"
			}
			resource "coder_app" "test" {
				agent_id = coder_agent.dev.id
				slug = "test"
				display_name = "Testing"
				url = "https://google.com"
				external = true
				hidden = false
				open_in = "tab"
			}
			`,
			hidden: false,
			openIn: "tab",
		}}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config: tc.config,
						Check: func(state *terraform.State) error {
							require.Len(t, state.Modules, 1)
							require.Len(t, state.Modules[0].Resources, 2)
							resource := state.Modules[0].Resources["coder_app.test"]
							require.NotNil(t, resource)
							require.Equal(t, strconv.FormatBool(tc.hidden), resource.Primary.Attributes["hidden"])
							require.Equal(t, tc.openIn, resource.Primary.Attributes["open_in"])
							return nil
						},
						ExpectError: nil,
					}},
				})
			})
		}
	})

	t.Run("DisplayName", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			displayName string
			expectValue string
			expectError *regexp.Regexp
		}{
			{
				name:        "Empty",
				displayName: "",
			},
			{
				name:        "Regular",
				displayName: "Regular Application",
			},
			{
				name:        "DisplayNameStillOK",
				displayName: "0123456789012345678901234567890123456789012345678901234567890123",
			},
			{
				name:        "DisplayNameTooLong",
				displayName: "01234567890123456789012345678901234567890123456789012345678901234",
				expectError: regexp.MustCompile("display name is too long"),
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				config := fmt.Sprintf(`
				provider "coder" {
				}
				resource "coder_agent" "dev" {
					os = "linux"
					arch = "amd64"
				}
				resource "coder_app" "code-server" {
					agent_id = coder_agent.dev.id
					slug = "code-server"
					display_name = "%s"
					url = "http://localhost:13337"
					open_in = "slim-window"
				}
				`, c.displayName)

				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config:      config,
						ExpectError: c.expectError,
					}},
				})
			})
		}
	})

	t.Run("Icon", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			icon        string
			expectError *regexp.Regexp
		}{
			{
				name: "Empty",
				icon: "",
			},
			{
				name: "ValidURL",
				icon: "/icon/region.svg",
			},
			{
				name:        "InvalidURL",
				icon:        "/icon%.svg",
				expectError: regexp.MustCompile("invalid URL escape"),
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				config := fmt.Sprintf(`
				provider "coder" {}
				resource "coder_agent" "dev" {
					os = "linux"
					arch = "amd64"
				}
				resource "coder_app" "code-server" {
					agent_id = coder_agent.dev.id
					slug = "code-server"
					display_name = "Testing"
					url = "http://localhost:13337"
					open_in = "slim-window"
					icon = "%s"
				}
				`, c.icon)

				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config:      config,
						ExpectError: c.expectError,
					}},
				})
			})
		}
	})

	t.Run("Tooltip", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name        string
			tooltip     string
			expectError *regexp.Regexp
		}{
			{
				name:    "Empty",
				tooltip: "",
			},
			{
				name: "ValidTooltip",
				tooltip: "You need to [Install Coder Desktop](https://coder.com/docs/user-guides/desktop" +
					"#install-coder-desktop) to use this button.",
			},
			{
				name: "TooltipMaxLength", // < 512 characters
				tooltip: "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut" +
					"labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco" +
					"laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in" +
					"voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat" +
					"non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. Sed ut" +
					"perspiciatis unde omnis iste natus error sit voluptatem accusant",
			},
			{
				name: "TooltipTooLong", // > 512 characters
				tooltip: "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor " +
					"incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud " +
					"exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor " +
					"in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint " +
					"occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum. " +
					"Sed ut perspiciatis unde omnis iste natus error sit voluptatem accusantium doloremque.",
				expectError: regexp.MustCompile("tooltip is too long"),
			},
		}

		for _, c := range cases {
			c := c

			t.Run(c.name, func(t *testing.T) {
				t.Parallel()

				config := fmt.Sprintf(`
				provider "coder" {}
				resource "coder_agent" "dev" {
					os = "linux"
					arch = "amd64"
				}
				resource "coder_app" "code-server" {
					agent_id = coder_agent.dev.id
					slug = "code-server"
					display_name = "Testing"
					url = "http://localhost:13337"
					open_in = "slim-window"
					tooltip = "%s"
				}
				`, c.tooltip)

				resource.Test(t, resource.TestCase{
					ProviderFactories: coderFactory(),
					IsUnitTest:        true,
					Steps: []resource.TestStep{{
						Config:      config,
						ExpectError: c.expectError,
					}},
				})
			})
		}
	})
}
