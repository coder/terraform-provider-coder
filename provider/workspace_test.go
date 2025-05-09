package provider_test

import (
	"regexp"
	"testing"

	"github.com/coder/terraform-provider-coder/v2/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkspace(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "templateID")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", "template123")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_VERSION", "v1.2.3")

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
				assert.Equal(t, "templateID", attribs["template_id"])
				assert.Equal(t, "template123", attribs["template_name"])
				assert.Equal(t, "v1.2.3", attribs["template_version"])
				return nil
			},
		}},
	})
}

func TestWorkspace_UndefinedOwner(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "templateID")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_NAME", "template123")
	t.Setenv("CODER_WORKSPACE_TEMPLATE_VERSION", "v1.2.3")

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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
				assert.Equal(t, "templateID", attribs["template_id"])
				assert.Equal(t, "template123", attribs["template_name"])
				assert.Equal(t, "v1.2.3", attribs["template_version"])
				// Skip other asserts
				return nil
			},
		}},
	})
}

func TestWorkspace_MissingTemplateName(t *testing.T) {
	t.Setenv("CODER_WORKSPACE_BUILD_ID", "1") // Let's pretend this is a workspace build

	t.Setenv("CODER_WORKSPACE_TEMPLATE_ID", "templateID")
	// CODER_WORKSPACE_TEMPLATE_NAME is missing
	t.Setenv("CODER_WORKSPACE_TEMPLATE_VERSION", "v1.2.3")

	resource.Test(t, resource.TestCase{
		ProviderFactories: coderFactory(),
		IsUnitTest:        true,
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

// TestWorkspace_PrebuildEnv validates that our handling of input environment variables is correct.
func TestWorkspace_PrebuildEnv(t *testing.T) {
	cases := []struct {
		name  string
		envs  map[string]string
		check func(state *terraform.State, resource *terraform.ResourceState) error
	}{
		{
			name: "unused",
			envs: map[string]string{},
			check: func(state *terraform.State, resource *terraform.ResourceState) error {
				attribs := resource.Primary.Attributes
				assert.Equal(t, "false", attribs["is_prebuild"])
				assert.Equal(t, "0", attribs["prebuild_count"])
				assert.Equal(t, "false", attribs["is_prebuild_claim"])
				return nil
			},
		},
		{
			name: "prebuild=true",
			envs: map[string]string{
				provider.IsPrebuildEnvironmentVariable(): "true",
			},
			check: func(state *terraform.State, resource *terraform.ResourceState) error {
				attribs := resource.Primary.Attributes
				assert.Equal(t, "true", attribs["is_prebuild"])
				assert.Equal(t, "1", attribs["prebuild_count"])
				assert.Equal(t, "false", attribs["is_prebuild_claim"])
				return nil
			},
		},
		{
			name: "prebuild=false",
			envs: map[string]string{
				provider.IsPrebuildEnvironmentVariable(): "false",
			},
			check: func(state *terraform.State, resource *terraform.ResourceState) error {
				attribs := resource.Primary.Attributes
				assert.Equal(t, "false", attribs["is_prebuild"])
				assert.Equal(t, "0", attribs["prebuild_count"])
				assert.Equal(t, "false", attribs["is_prebuild_claim"])
				return nil
			},
		},
		{
			name: "prebuild_claim=true",
			envs: map[string]string{
				provider.IsPrebuildClaimEnvironmentVariable(): "true",
			},
			check: func(state *terraform.State, resource *terraform.ResourceState) error {
				attribs := resource.Primary.Attributes
				assert.Equal(t, "false", attribs["is_prebuild"])
				assert.Equal(t, "0", attribs["prebuild_count"])
				assert.Equal(t, "true", attribs["is_prebuild_claim"])
				return nil
			},
		},
		{
			name: "prebuild_claim=false",
			envs: map[string]string{
				provider.IsPrebuildClaimEnvironmentVariable(): "false",
			},
			check: func(state *terraform.State, resource *terraform.ResourceState) error {
				attribs := resource.Primary.Attributes
				assert.Equal(t, "false", attribs["is_prebuild"])
				assert.Equal(t, "0", attribs["prebuild_count"])
				assert.Equal(t, "false", attribs["is_prebuild_claim"])
				return nil
			},
		},
		{
			// Should not ever happen, but let's ensure our defensive check is activated. We can't ever have both flags
			// being true.
			name: "prebuild=true,prebuild_claim=true",
			envs: map[string]string{
				provider.IsPrebuildEnvironmentVariable():      "true",
				provider.IsPrebuildClaimEnvironmentVariable(): "true",
			},
			check: func(state *terraform.State, resource *terraform.ResourceState) error {
				attribs := resource.Primary.Attributes
				assert.Equal(t, "false", attribs["is_prebuild"])
				assert.Equal(t, "0", attribs["prebuild_count"])
				assert.Equal(t, "true", attribs["is_prebuild_claim"])
				return nil
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envs {
				t.Setenv(k, v)
			}

			resource.Test(t, resource.TestCase{
				ProviderFactories: coderFactory(),
				IsUnitTest:        true,
				Steps: []resource.TestStep{{
					Config: `
provider "coder" {
	url = "https://example.com:8080"
}
data "coder_workspace" "me" {
}`,
					Check: func(state *terraform.State) error {
						// Baseline checks
						require.Len(t, state.Modules, 1)
						require.Len(t, state.Modules[0].Resources, 1)
						resource := state.Modules[0].Resources["data.coder_workspace.me"]
						require.NotNil(t, resource)

						return tc.check(state, resource)
					},
				}},
			})
		})
	}
}
