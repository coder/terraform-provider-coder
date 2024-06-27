package provider_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestWorkspace(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_OWNER", "owner123")
	t.Setenv("CODER_WORKSPACE_OWNER_ID", "11111111-1111-1111-1111-111111111111")
	t.Setenv("CODER_WORKSPACE_OWNER_NAME", "Mr Owner")
	t.Setenv("CODER_WORKSPACE_OWNER_EMAIL", "owner123@example.com")
	t.Setenv("CODER_WORKSPACE_OWNER_SESSION_TOKEN", "abc123")
	t.Setenv("CODER_WORKSPACE_OWNER_GROUPS", `["group1", "group2"]`)
	t.Setenv("CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN", "supersecret")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "templateID")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", "template123")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_VERSION", "v1.2.3")

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
				assert.Equal(t, "https://example.com:8080", attribs["access_url"])
				assert.Equal(t, "8080", attribs["access_port"])
				assert.Equal(t, "owner123", attribs["owner"])
				assert.Equal(t, "11111111-1111-1111-1111-111111111111", attribs["owner_id"])
				assert.Equal(t, "Mr Owner", attribs["owner_name"])
				assert.Equal(t, "owner123@example.com", attribs["owner_email"])
				assert.Equal(t, "group1", attribs["owner_groups.0"])
				assert.Equal(t, "group2", attribs["owner_groups.1"])
				assert.Equal(t, "templateID", attribs["template_id"])
				assert.Equal(t, "template123", attribs["template_name"])
				assert.Equal(t, "v1.2.3", attribs["template_version"])
				assert.Equal(t, "supersecret", attribs["owner_oidc_access_token"])
				return nil
			},
		}},
	})
}

func TestWorkspace_UndefinedOwner(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_OWNER", "owner123")
	t.Setenv("CODER_WORKSPACE_OWNER_SESSION_TOKEN", "abc123")
	t.Setenv("CODER_WORKSPACE_OWNER_GROUPS", `["group1", "group2"]`)
	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "templateID")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", "template123")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_VERSION", "v1.2.3")

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
				assert.Equal(t, "owner123", attribs["owner"])
				assert.Equal(t, "default@example.com", attribs["owner_email"])
				// Skip other asserts
				return nil
			},
		}},
	})
}

func TestWorkspace_MissingTemplateName(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_BUILD_ID", "1") // Let's pretend this is a workspace build

	t.Setenv("CODER_WORKSPACE_OWNER", "owner123")
	t.Setenv("CODER_WORKSPACE_OWNER_ID", "11111111-1111-1111-1111-111111111111")
	t.Setenv("CODER_WORKSPACE_OWNER_NAME", "Mr Owner")
	t.Setenv("CODER_WORKSPACE_OWNER_EMAIL", "owner123@example.com")
	t.Setenv("CODER_WORKSPACE_OWNER_SESSION_TOKEN", "abc123")
	t.Setenv("CODER_WORKSPACE_OWNER_GROUPS", `["group1", "group2"]`)
	t.Setenv("CODER_WORKSPACE_OWNER_OIDC_ACCESS_TOKEN", "supersecret")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "templateID")
	// CODER_WORKSPACE_TEMPLATE_NAME is missing
	t.Setenv("CODER_WORKSPACE_TEMPLATE_VERSION", "v1.2.3")

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
			ExpectError: regexp.MustCompile("CODER_WORKSPACE_TEMPLATE_NAME is required"),
		}},
	})
}
