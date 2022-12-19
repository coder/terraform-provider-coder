package provider_test

import (
	"regexp"
	"testing"

	"github.com/coder/terraform-provider-coder/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

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
					icon = "/icon/storage.svg"
					daily_cost = 200
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
					"icon":             "/icon/storage.svg",
					"daily_cost":       "200",
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
					icon = "/icon/storage.svg"
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
