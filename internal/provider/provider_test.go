package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/internal/provider"
)

func TestProvider(t *testing.T) {
	t.Parallel()
	tfProvider := provider.New()
	err := tfProvider.InternalValidate()
	require.NoError(t, err)
}

func TestWorkspace(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_OWNER", "owner123")
	t.Setenv("CODER_WORKSPACE_OWNER_EMAIL", "owner123@example.com")

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
			data "coder_workspace" "me" {
			}`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["data.coder_workspace.me"]
				require.NotNil(t, resource)

				attribs := resource.Primary.Attributes
				value := attribs["transition"]
				require.NotNil(t, value)
				t.Log(value)
				require.Equal(t, "owner123", attribs["owner"])
				require.Equal(t, "owner123@example.com", attribs["owner_email"])
				return nil
			},
		}},
	})
}

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

func TestAgentInstance(t *testing.T) {
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

func TestApp(t *testing.T) {
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
					name = "code-server"
					icon = "builtin:vim"
					relative_path = true
					url = "http://localhost:13337"
				}
				`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 2)
				resource := state.Modules[0].Resources["coder_app.code-server"]
				require.NotNil(t, resource)
				for _, key := range []string{
					"agent_id",
					"name",
					"icon",
					"relative_path",
					"url",
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

func TestMetadata(t *testing.T) {
	t.Parallel()
	prov := provider.New()
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": prov,
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
				resource "coder_metadata" "agent" {
					resource_id = coder_agent.dev.id
					item {
						key = "foo"
						value = "bar"
					}
					item {
						key = "secret"
						value = "squirrel"
						sensitive = true
					}
					item {
						key = "implicit_null"
					}
					item {
						key = "explicit_null"
						value = null
					}
					item {
						key = "empty"
						value = ""
					}
				}
				`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 2)
				agent := state.Modules[0].Resources["coder_agent.dev"]
				require.NotNil(t, agent)
				metadata := state.Modules[0].Resources["coder_metadata.agent"]
				require.NotNil(t, metadata)
				t.Logf("metadata attributes: %#v", metadata.Primary.Attributes)
				for key, expected := range map[string]string{
					"resource_id":      agent.Primary.Attributes["id"],
					"item.#":           "5",
					"item.0.key":       "foo",
					"item.0.value":     "bar",
					"item.0.sensitive": "false",
					"item.1.key":       "secret",
					"item.1.value":     "squirrel",
					"item.1.sensitive": "true",
					"item.2.key":       "implicit_null",
					"item.2.is_null":   "true",
					"item.2.sensitive": "false",
					"item.3.key":       "explicit_null",
					"item.3.is_null":   "true",
					"item.3.sensitive": "false",
					"item.4.key":       "empty",
					"item.4.value":     "",
					"item.4.is_null":   "false",
					"item.4.sensitive": "false",
				} {
					require.Equal(t, expected, metadata.Primary.Attributes[key])
				}
				return nil
			},
		}},
	})
}
