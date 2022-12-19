package provider_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestApp(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
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
			}
			`,
			expectError: regexp.MustCompile("conflicts with subdomain"),
		}}
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				resource.Test(t, resource.TestCase{
					Providers: map[string]*schema.Provider{
						"coder": provider.New(),
					},
					IsUnitTest: true,
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
					Providers: map[string]*schema.Provider{
						"coder": provider.New(),
					},
					IsUnitTest: true,
					Steps: []resource.TestStep{{
						Config:      config,
						Check:       checkFn,
						ExpectError: c.expectError,
					}},
				})
			})
		}
	})
}
