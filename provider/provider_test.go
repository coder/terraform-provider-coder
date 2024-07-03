package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	"github.com/coder/terraform-provider-coder/provider"
)

func TestProvider(t *testing.T) {
	t.Parallel()
	tfProvider := provider.New()
	err := tfProvider.InternalValidate()
	require.NoError(t, err)
}

// TestProviderEmpty ensures that the provider can be configured without
// any actual input data. This is important for adding new fields
// with backwards compatibility guarantees.
func TestProviderEmpty(t *testing.T) {
	t.Parallel()
	resource.Test(t, resource.TestCase{
		Providers: map[string]*schema.Provider{
			"coder": provider.New(),
		},
		IsUnitTest: true,
		Steps: []resource.TestStep{{
			Config: `
			provider "coder" {}
			data "coder_provisioner" "me" {}
			data "coder_workspace" "me" {}
			data "coder_workspace_owner" "me" {}
			data "coder_external_auth" "git" {
				id = "git"
			}
			data "coder_parameter" "param" {
				name = "hey"
			}`,
			Check: func(state *terraform.State) error {
				return nil
			},
		}},
	})
}
