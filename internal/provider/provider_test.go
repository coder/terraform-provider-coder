package provider_test

import (
	"regexp"
	"runtime"
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
				url = "https://example.com:8080"
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
				require.Equal(t, "8080", attribs["access_port"])
				require.Equal(t, "owner123", attribs["owner"])
				require.Equal(t, "owner123@example.com", attribs["owner_email"])
				return nil
			},
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
				url = "https://example.com:8080"
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
				require.Equal(t, "https://example.com:8080", attribs["access_url"])
				require.Equal(t, "owner123", attribs["owner"])
				require.Equal(t, "owner123@example.com", attribs["owner_email"])
				return nil
			},
		}},
	})
}

func TestProvisioner(t *testing.T) {
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {
			}
			data "coder_provisioner" "me" {
			}`,
			Check: func(state *terraform.State) error {
				require.Len(t, state.Modules, 1)
				require.Len(t, state.Modules[0].Resources, 1)
				resource := state.Modules[0].Resources["data.coder_provisioner.me"]
				require.NotNil(t, resource)

				attribs := resource.Primary.Attributes
				require.Equal(t, runtime.GOOS, attribs["os"])
				require.Equal(t, runtime.GOARCH, attribs["arch"])
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
					"name",
					"icon",
					"relative_path",
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
					hide = true
					icon = "/icons/storage.svg"
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
					"hide":             "true",
					"icon":             "/icons/storage.svg",
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

func TestMetadataDuplicateKeys(t *testing.T) {
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
					hide = true
					icon = "/icons/storage.svg"
					item {
						key = "foo"
						value = "bar"
					}
					item {
						key = "foo"
						value = "bar"
					}
				}
				`,
			ExpectError: regexp.MustCompile("duplicate metadata key"),
		}},
	})
}

func TestParameter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		Name        string
		Config      string
		ExpectError *regexp.Regexp
		Check       func(state *terraform.ResourceState)
	}{{
		Name: "NumberValidation",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "number"
}
`,
	}, {
		Name: "DefaultNotNumber",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "number"
	default = true
}
`,
		ExpectError: regexp.MustCompile("is not a number"),
	}, {
		Name: "DefaultNotBool",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "bool"
	default = 5
}
`,
		ExpectError: regexp.MustCompile("is not a bool"),
	}, {
		Name: "OptionNotBool",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "bool"
	option {
		value = 1
		name = 1
	}
	option {
		value = 2
		name = 2
	}
}`,
		ExpectError: regexp.MustCompile("\"2\" is not a bool"),
	}, {
		Name: "MultipleOptions",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	option {
		name = "1"
		value = "1"
		icon = "/icon/code.svg"
		description = "Something!"
	}
	option {
		name = "2"
		value = "2"
	}
}

data "google_compute_regions" "regions" {}

data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	icon = "/icon/asdasd.svg"
	option {
		name = "United States"
		value = "us-central1-a"
		icon = "/icon/usa.svg"
		description = "If you live in America, select this!"
	}
	option {
		name = "Europe"
		value = "2"
	}
}
`,
		Check: func(state *terraform.ResourceState) {
			for key, expected := range map[string]string{
				"name":                 "Region",
				"option.#":             "2",
				"option.0.name":        "1",
				"option.0.value":       "1",
				"option.0.icon":        "/icon/code.svg",
				"option.0.description": "Something!",
			} {
				require.Equal(t, expected, state.Primary.Attributes[key])
			}
		},
	}, {
		Name: "DefaultWithOption",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	default = "hi"
	option {
		name = "1"
		value = "1"
	}
	option {
		name = "2"
		value = "2"
	}
}
`,
		ExpectError: regexp.MustCompile("Invalid combination of arguments"),
	}, {
		Name: "SingleOption",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	option {
		name = "1"
		value = "1"
	}
}
`,
	}, {
		Name: "DuplicateOptionDisplayName",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	option {
		name = "1"
		value = "1"
	}
	option {
		name = "1"
		value = "2"
	}
}
`,
		ExpectError: regexp.MustCompile("cannot have the same display name"),
	}, {
		Name: "DuplicateOptionValue",
		Config: `
provider "coder" {}
data "coder_parameter" "region" {
	name = "Region"
	type = "string"
	option {
		name = "1"
		value = "1"
	}
	option {
		name = "2"
		value = "1"
	}
}
`,
		ExpectError: regexp.MustCompile("cannot have the same value"),
	}} {
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
						param := state.Modules[0].Resources["data.coder_parameter.region"]
						require.NotNil(t, param)
						t.Logf("parameter attributes: %#v", param.Primary.Attributes)
						if tc.Check != nil {
							tc.Check(param)
						}
						return nil
					},
				}},
			})
		})
	}
}
